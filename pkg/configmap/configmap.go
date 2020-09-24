package configmap

import (
	"context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	"github.com/tkashem/graceful/pkg/core"
	"github.com/tkashem/graceful/pkg/namespace"
)

func NewWorker(client kubernetes.Interface, getter namespace.Getter) core.Worker {
	return func(wc *core.WorkerContext) {
		prefix := "test-"
		ctx := context.TODO()

		// find an available namespace here
		namespace, done, err := getter()
		if err != nil {
			klog.Errorf("[worker:%s] error getting namespace - %s", wc.Name, err.Error())
			return
		}
		defer done()

		err = func() error {
			cm, err := client.CoreV1().ConfigMaps(namespace).Create(ctx, &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: prefix,
					Labels: map[string]string{
						"clusterloader": "true",
					},
				},
				Data: map[string]string{
					"key1": "foo",
					"key2": "bar",
				},
			}, metav1.CreateOptions{})
			if err != nil {
				return err
			}

			current, err := client.CoreV1().ConfigMaps(namespace).Get(ctx, cm.GetName(), metav1.GetOptions{})
			if err != nil {
				return err
			}

			current.Data["key3"] = "baz"
			updated, err := client.CoreV1().ConfigMaps(namespace).Update(ctx, current, metav1.UpdateOptions{})
			if err != nil {
				return err
			}

			if err := client.CoreV1().ConfigMaps(cm.GetNamespace()).Delete(ctx, updated.GetName(), metav1.DeleteOptions{}); err != nil {
				return err
			}

			return nil
		}()

		if err != nil {
			klog.Errorf("[worker:%s] error: %s", wc.Name, err.Error())
		}
	}
}
