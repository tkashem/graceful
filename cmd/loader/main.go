package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/tkashem/graceful/pkg/core"
	"net/http"
	"sync"
	"time"

	"k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/tkashem/graceful/pkg/test"
	"github.com/tkashem/graceful/pkg/poddensity"
)

var (
	concurrency = flag.Int("concurrency", 100, "number of concurrent workers")
	burst = flag.Int("burst", 10, "burst size")
	delay = flag.Duration("delay", 1 * time.Second, "step delay")
	duration = flag.Duration("duration", 1 * time.Minute, "test duration after steady state")

	kubeConfigPath = flag.String("kubeconfig", "", "path to the kubeconfig file")
	port = flag.Int("metrics-port", 9000, "metrics port")
	timeout = flag.Duration("timeout", 5 * time.Minute, "how long to wait for deployment/pod to be ready")
)

func main() {
	flag.Parse()

	klog.Infof("kubeConfigPath=%s", *kubeConfigPath)
	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfigPath)
	if err != nil {
		return
	}
	config.QPS = 10000
	config.Burst = 20000
	klog.Infof("rest.Config.Host=%s", config.Host)

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	shutdown, cancel := context.WithCancel(context.TODO())
	shutdownHandler := server.SetupSignalHandler()
	go func() {
		defer cancel()

		<-shutdownHandler
		klog.Info("Received SIGTERM or SIGINT signal, initiating shutdown.")
	}()

	// initialize
	initializers := test.InitializerChain{
		// use component-base/metrics/prometheus/restclient instead
		test.ClientGoMetricsInitialize,
	}
	if err := initializers.Invoke(); err != nil {
		panic(err)
	}

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", *port), metricsMux)
		if err != nil {
			klog.Errorf("Metrics (http) serving failed: %v", err)
		}
	}()

	// setup a dummy worker
	worker := poddensity.NewWorker(client, *timeout)

	// need this to wait for all workers to exit.
	wg := &sync.WaitGroup{}
	test, testCancel := context.WithTimeout(shutdown, *duration)
	defer testCancel()

	tc := core.TestContext{
		TestCancel: test,
		WaitGroup:  wg,
	}

	// run this worker in parallel
	runner := core.NewRunnerWithDelay(1 * time.Millisecond)
	actions := runner.ToActions(&tc, *concurrency, worker, "pod-density")
	generator := core.NewSteppedLoadGenerator(*delay, *burst)

	go generator.Generate(actions)

	klog.Info("waiting for test to complete")
	<-test.Done()

	klog.Info("waiting for worker(s) to be done")
	wg.Wait()

	klog.Info("all worker(s) are done")
}
