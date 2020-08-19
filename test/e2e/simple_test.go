package e2e

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/stretchr/testify/require"
)

func TestSimple(t *testing.T) {
	client, err := kubernetes.NewForConfig(options.config)
	require.NoErrorf(t, err, "failed to construct client for kubernetes - %v", err)

	// client.CoreV1().Namespaces().Get( context.TODO(), "kube-system", metav1.GetOptions{})

	options := metav1.ListOptions{}
	watcher, err := client.CoreV1().ConfigMaps("test").Watch(context.TODO(), options)
	if err != nil {
		t.Fatalf("failed to establish watch - %s", err)
	}

	ch := watcher.ResultChan()
	for event := range ch {
		t.Logf("type=%s object: %v", string(event.Type), event.Object)
	}
}

func TestConnectivity(t *testing.T) {
	client, err := kubernetes.NewForConfig(options.config)
	require.NoErrorf(t, err, "failed to construct client for kubernetes - %v", err)

	client.CoreV1().Namespaces().Get( context.TODO(), "kube-system", metav1.GetOptions{})
}
