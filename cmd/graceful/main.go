package main

import (
	"context"
	"flag"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
	"net/http"
	"sync"

	"github.com/tkashem/graceful/pkg/test"
)

var (
	kubeConfigPath = flag.String("kubeconfig", "", "path to the kubeconfig file")
	kubeletkubeConfigPath = flag.String("kubelet-kubeconfig", "", "path to the kubeconfig file used by kubelet")
	kubeAPIServerPodName = flag.String("kube-apiserver-pod-name", "", "kube-apiserver pod name on the node")
)

func main() {
	flag.Parse()

	klog.Infof("kubeConfigPath=%s", *kubeConfigPath)
	klog.Infof("kubeletkubeConfigPath=%s", *kubeletkubeConfigPath)
	klog.Infof("kubeAPIServerPodName=%s", *kubeAPIServerPodName)

	config, err := useAPIURLUsedByKuelet(*kubeConfigPath, *kubeletkubeConfigPath)
	if err != nil {
		panic(err)
	}

	klog.Infof("rest.Config.Host=%s", config.Host)

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}

	initializer, monitor, err := test.NewKubeAPIServerMonitor(client, *kubeAPIServerPodName)
	if err != nil {
		panic(err)
	}

	// initialize
	initializers := test.InitializerChain{
		test.ClientGoMetricsInitialize,
		initializer,
	}
	if err := initializers.Invoke(); err != nil {
		panic(err)
	}

	shutdown, cancel := context.WithCancel(context.TODO())
	shutdownHandler := server.SetupSignalHandler()
	go func() {
		defer cancel()

		<-shutdownHandler
		klog.Info("Received SIGTERM or SIGINT signal, initiating shutdown.")
	}()

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.ListenAndServe(":9090", metricsMux)
		if err != nil {
			klog.Errorf("Metrics (http) serving failed: %v", err)
		}
	}()

	workers := test.WorkerChain{
		monitor,
		test.SlowCall(client),
	}
	workers = append(workers, test.FastCalls(client, 6)...)

	// launch workers
	wg := &sync.WaitGroup{}
	workers.Invoke(shutdown, wg)

	<-shutdown.Done()

	klog.Info("waiting for worker to be done")
	wg.Wait()
	klog.Info("all worker(s) are done")
}

func useAPIURLUsedByKuelet(kubeConfigPath, kubeletkubeConfigPath string) (config *rest.Config, err error) {
	bytes, err := ioutil.ReadFile(kubeletkubeConfigPath)
	if err != nil {
		return
	}

	kubelet, err := clientcmd.RESTConfigFromKubeConfig(bytes)
	if err != nil {
		return
	}

	config, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	if err != nil {
		return
	}

	config.Host = kubelet.Host
	return
}
