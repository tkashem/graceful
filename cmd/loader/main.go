package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/tkashem/graceful/pkg/core"
	"net/http"
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
	longevity = flag.Duration("pod-longevity", 30 * time.Second, "how long we want pod to live")
	pool = flag.Int("namespaces", 1, "fixed namespace pool size")
)

func main() {
	flag.Parse()

	klog.Infof("[main] kubeConfigPath=%s", *kubeConfigPath)
	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfigPath)
	if err != nil {
		return
	}
	config.QPS = 10000
	config.Burst = 20000
	klog.Infof("[main] rest.Config.Host=%s", config.Host)

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	shutdown, cancel := context.WithCancel(context.TODO())
	shutdownHandler := server.SetupSignalHandler()
	go func() {
		defer cancel()

		<-shutdownHandler
		klog.Info("[main] Received SIGTERM or SIGINT signal, initiating shutdown.")
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
			klog.Errorf("[main] Metrics (http) serving failed: %v", err)
		}
	}()

	// create a test context
	tc, testCancel := core.NewTestContext(shutdown, *duration)
	defer testCancel()

	klog.Infof("[main] creating a pool of namespace size=%d", *pool)
	pool, err := poddensity.NewNamespacePool(client, *pool)
	if err != nil {
		panic(err)
	}

	// setup a dummy worker
	worker := poddensity.NewWorker(client, pool, *timeout, *longevity)

	// run this worker in parallel
	runner := core.NewRunnerWithDelay(1 * time.Millisecond)
	actions := runner.ToActions(tc, *concurrency, worker, "pod-density")
	generator := core.NewSteppedLoadGenerator(*delay, *burst)

	go generator.Generate(actions)

	klog.Infof("[main] waiting for test to complete, duration=%s", *duration)
	<-tc.TestCancel.Done()

	klog.Info("[main] test duration elapsed, waiting for worker(s) to be done")
	tc.WaitGroup.Wait()
	klog.Info("[main] all worker(s) are done")

	// cleaning up namespaces
	klog.Info("[main] cleaning up namespace pool")
	if err := pool.Cleanup(client); err != nil {
		klog.Errorf("[main] namespace cleanup failed - %s", err.Error())
	}
}
