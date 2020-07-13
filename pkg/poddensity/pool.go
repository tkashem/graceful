package poddensity

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/rand"

)

type NamespacePool []*corev1.Namespace

func NewNamespacePool(client kubernetes.Interface, size int) (NamespacePool, error){
	pool := NamespacePool{}
	prefix := "test-"

	for i := 1; i <= size; i++ {
		ns, err := client.CoreV1().Namespaces().Create(context.TODO(), &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: prefix,
				Labels: map[string]string{
					"clusterloader": "true",
				},
			},
		}, metav1.CreateOptions{})
		if err != nil {
			return nil, err
		}

		pool = append(pool, ns)
	}

	return pool, nil
}

func (p NamespacePool) GetRandom() string {
	if len(p) == 0 {
		return ""
	}

	i := rand.Intn( len(p) )
	return p[i].GetName()
}

func (p NamespacePool) Cleanup(client kubernetes.Interface) error {
	for i := range p {
		ns := p[i]
		if err := client.CoreV1().Namespaces().Delete(context.TODO(), ns.GetName(), metav1.DeleteOptions{}); err != nil {
			if k8serrors.IsNotFound(err) {
				continue
			}

			return fmt.Errorf("deleted=%d target=%d error deleting namespace: %s", i+1, len(p) - i,  err.Error())
		}
	}

	return nil
}


