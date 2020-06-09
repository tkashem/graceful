package test

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	"k8s.io/client-go/rest"
)

func DefaultReadonlyWorker(client kubernetes.Interface, namespace string, count int) []*WorkerConfig {
	worker := func() {
		if err := DefaultGetNamespace(namespace, client); err != nil {
			klog.Errorf("step error=%s", err.Error())
		}
	}

	configs := make([]*WorkerConfig, 0)
	for i := 1; i <=count; i++ {
		configs = append(configs, &WorkerConfig{
			Name:         fmt.Sprintf("default-steps-%d", i),
			WaitInterval: 1 * time.Millisecond,
			Worker:       worker,
		})
	}

	return configs
}

func WithNewConnectionWorker(config *rest.Config, namespace string, count int) []*WorkerConfig {
	worker := func(name string) Worker {
		return func() {
			if err := WithNewConnection(config, name, namespace); err != nil {
				klog.Errorf("step error=%s", err.Error())
			}
		}
	}

	configs := make([]*WorkerConfig, 0)
	for i := 1; i <=count; i++ {
		configs = append(configs, &WorkerConfig{
			Name:         fmt.Sprintf("default-steps-%d", i),
			WaitInterval: 1 * time.Millisecond,
			Worker:       worker(fmt.Sprintf("default-steps-%d", i)),
		})
	}

	return configs
}


func DefaultGetNamespace(namespace string, client kubernetes.Interface) error {
	_, err := client.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{})
	if err != nil {
		return err
	}

	return nil
}

func WithNewConnection(config *rest.Config, userAgent, namespace string) error {
	copy := rest.CopyConfig(config)
	copy = rest.AddUserAgent(copy, userAgent)

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}

	return DefaultGetNamespace(namespace, client)
}
