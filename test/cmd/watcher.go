package main

import (
	"context"
	"flag"
	"os"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)
var (
	kubeConfigPath = flag.String(
		"kubeconfig", "", "path to the kubeconfig file")
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()

	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfigPath)
	if err != nil {
		klog.Errorf("failed to load configuration - %v", err)
		os.Exit(-1)
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Errorf("failed to construct client for kubernetes - %v", err)
		os.Exit(-1)
	}



	go watch(client)
	go watch(client)
	go watch(client)

	block := make(chan struct{})
	<-block
}

func watch(client kubernetes.Interface) {
	options := metav1.ListOptions{}
	watcher, err := client.CoreV1().ConfigMaps("test").Watch(context.TODO(), options)
	if err != nil {
		klog.Errorf("failed to establish watch - %v", err)
		os.Exit(-1)
	}

	ch := watcher.ResultChan()
	for event := range ch {
		klog.Infof("type=%s object: %v", string(event.Type), event.Object)
	}
}