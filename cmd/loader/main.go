package main

import (
	"context"
	"flag"
	"fmt"
	"reflect"
	"net/http"
	"sync"
	"io/ioutil"

	"k8s.io/apiserver/pkg/server"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/tkashem/graceful/pkg/test"
)

var (
	kubeConfigPath = flag.String("kubeconfig", "", "path to the kubeconfig file")
	port = flag.Int("metrics-port", 9000, "metrics port")
	kubeletkubeConfigPath = flag.String("kubelet-kubeconfig", "", "path to the kubeconfig file used by kubelet")
	kubeAPIServerPodName = flag.String("kube-apiserver-pod-name", "", "kube-apiserver pod name on the node")
	kubeAPIServerNamespace = flag.String("kube-apiserver-namespace", "openshift-kube-apiserver", "kube-apiserver namespace")
	concurrency = flag.Int("concurrent", 1, "number of concurrent workers")
)

func main() {
	// klog.InitFlags(nil)
	flag.Parse()

	klog.Infof("kubeConfigPath=%s", *kubeConfigPath)
	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfigPath)
	if err != nil {
		return
	}
	config.QPS = 10000
	config.Burst = 20000
	if err = setHostForConfig(config, *kubeletkubeConfigPath); err != nil {
		panic(err)
	}

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

	// setup a namespace
	ns, err := client.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "load-test",
			Labels: map[string]string{
				"load-test": "true",
			},
		},
	}, metav1.CreateOptions{} )
	if err != nil {
		panic(err)
	}
	klog.Infof("setup test namespace - namespace=%s", ns.GetName())

	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%d", *port), metricsMux)
		if err != nil {
			klog.Errorf("Metrics (http) serving failed: %v", err)
		}
	}()

	workers := test.WorkerChain{
		test.SlowCall(client),
	}
	workers = append(workers, test.DefaultStepsWorker(client, ns.GetName(), *concurrency)...)

	// launch workers
	wg := &sync.WaitGroup{}
	workers.Invoke(shutdown, wg)

	<-shutdown.Done()

	klog.Info("waiting for worker to be done")
	wg.Wait()
	klog.Info("all worker(s) are done")

	klog.Info("cleaning up")
}

func setHostForConfig(config *rest.Config, kubeConfigPath string) error {
	if len(kubeConfigPath) == 0 {
		return nil
	}

	bytes, err := ioutil.ReadFile(kubeConfigPath)
	if err != nil {
		return err
	}

	kubelet, err := clientcmd.RESTConfigFromKubeConfig(bytes)
	if err != nil {
		return err
	}

	config.Host = kubelet.Host
	return nil
}

func startInformers(shutdown context.Context, factory informers.SharedInformerFactory) error {
	factory.Start(shutdown.Done())
	status := factory.WaitForCacheSync(shutdown.Done())
	if names := check(status); len(names) > 0 {
		return fmt.Errorf("WaitForCacheSync did not successfully complete resources=%s", names)
	}

	return nil
}

func check(status map[reflect.Type]bool) []string {
	names := make([]string, 0)

	for objType, synced := range status {
		if !synced {
			names = append(names, objType.Name())
		}
	}

	return names
}