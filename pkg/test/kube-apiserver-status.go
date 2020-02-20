package test

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

var (
	kubeapiserver = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "graceful_test_kube_apiserver",
			Help: "number of kube-apiserver(s)",
		},
		[]string{"name"},
	)
)

func NewKubeAPIServerMonitor(client kubernetes.Interface, podName string) (initializer Initializer, config *WorkerConfig, err error) {
	initializer = func() error {
		if err := prometheus.Register(kubeapiserver); err != nil {
			return err
		}

		return nil
	}

	config = &WorkerConfig{
		Name: "kube-apiserver-pod-monitor",
		WaitInterval: 2 * time.Second,
		Worker: func() {
			status(client, podName)
		},
	}

	return
}

func status(client kubernetes.Interface, name string) {
	namespace := "openshift-kube-apiserver"
	pod, err := client.CoreV1().Pods(namespace).Get(name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("name=%s/%s failed to get kube-apiserver pod - %s", namespace, name, err.Error())

		if !k8serrors.IsNotFound(err) {
			return
		}
	}

	at := float64(0)
	if pod != nil && pod.Status.Phase == corev1.PodRunning {
		at = 1
	}

	kubeapiserver.WithLabelValues(name).Set(at)
}
