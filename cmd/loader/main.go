package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/tkashem/graceful/pkg/core"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/tkashem/graceful/pkg/test"
	"github.com/tkashem/graceful/pkg/namespace"
	"github.com/tkashem/graceful/pkg/poddensity"
	"github.com/tkashem/graceful/pkg/configmap"
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
	fixedPool = flag.Int("namespaces", 1, "fixed namespace pool size")
	podsPerNamespace = flag.Int("pods-per-namespace", 0, "number of pods per namespace")
	workloadType = flag.String("workload", "pod", "workload type, supported values pod, configmap")
)

func main() {
	flag.Parse()

	klog.Infof("[main] kubeConfigPath=%s", *kubeConfigPath)
	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfigPath)
	if err != nil {
		return
	}
	config.QPS = 15000
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

	var pool namespace.Pool
	if *podsPerNamespace > 0 {
		klog.Infof("[main] using a churning namespace pool pods-per-namespace=%d", *podsPerNamespace)
		p, err := namespace.NewPoolWithChurn(config, *podsPerNamespace)
		if err != nil {
			panic(err)
		}

		pool = p
	} else {
		klog.Infof("[main] using a fixed namespace pool  size=%d", *fixedPool)
		p, err := namespace.NewFixedPool(client, *fixedPool)
		if err != nil {
			panic(err)
		}

		pool= p
	}

	// setup the worker
	var worker core.Worker
	switch *workloadType {
	case "pod":
		worker = poddensity.NewWorker(client, pool.GetNamespace, *timeout, *longevity)
	case "configmap":
		worker = configmap.NewWorker(client, pool.GetNamespace)
	default:
		panic(fmt.Sprintf("workload type not supported workload=%s", *workloadType))
	}

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
	if err := pool.Dispose(); err != nil {
		klog.Errorf("[main] namespace cleanup failed - %s", err.Error())
	}
}
