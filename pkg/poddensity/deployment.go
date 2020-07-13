package poddensity

import (
	"fmt"
	"context"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/wait"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog"

	"github.com/tkashem/graceful/pkg/core"
)

func NewWorker(client kubernetes.Interface, timeout, longevity time.Duration) core.Worker {
	return func(wc *core.WorkerContext) {
		prefix := "test-"
		ctx := context.TODO()
		o, err := create(ctx, client, prefix)
		if err != nil {
			klog.Errorf("[worker:%s] create error: %s", wc.Name, err.Error())
		}

		wc.WaitGroup.Add(1)
		go func() {
			defer wc.WaitGroup.Done()
			ctx := context.TODO()

			if o.deployment != nil {
				d := o.deployment
				err := wait.Poll(time.Second, timeout, func() (done bool, pollErr error) {
					deployment, err := client.AppsV1().Deployments(d.GetNamespace()).Get(ctx, d.GetName(), metav1.GetOptions{})
					if err != nil {
						if !k8serrors.IsNotFound(err) {
							pollErr = err
							return
						}
					}

					available, err := GetDeploymentStatus(deployment)
					if !available {
						return
					}

					done = true
					return
				})

				if err != nil {
					klog.Errorf("[worker:%s] error while polling for deployment readiness: %s", wc.Name, err.Error())
				} else {
					// we give the Pod some time to live
					wait := wait.Jitter(longevity, 1.0)
					<-time.After(wait)
				}

				if err := client.AppsV1().Deployments(d.GetNamespace()).Delete(ctx, d.GetName(), metav1.DeleteOptions{}); err != nil {
					klog.Errorf("[worker:%s] error deleting deployment: %s", wc.Name, err.Error())
				}
			}

			if s := o.secret; s != nil {
				if err := client.CoreV1().Secrets(s.GetNamespace()).Delete(ctx, s.GetName(), metav1.DeleteOptions{}); err != nil {
					klog.Errorf("[worker:%s] error deleting secret: %s", wc.Name, err.Error())
				}
			}

			if cm := o.cm; cm != nil {
				if err := client.CoreV1().ConfigMaps(cm.GetNamespace()).Delete(ctx, cm.GetName(), metav1.DeleteOptions{}); err != nil {
					klog.Errorf("[worker:%s] error deleting cm: %s", wc.Name, err.Error())
				}
			}

			if sa := o.sa; sa != nil {
				if err := client.CoreV1().ServiceAccounts(sa.GetNamespace()).Delete(ctx, sa.GetName(), metav1.DeleteOptions{}); err != nil {
					klog.Errorf("[worker:%s] error deleting cm: %s", wc.Name, err.Error())
				}
			}

			if ns := o.ns; ns != nil {
				if err := client.CoreV1().Namespaces().Delete(ctx, ns.GetName(), metav1.DeleteOptions{}); err != nil {
					klog.Errorf("[worker:%s] error deleting namespace: %s", wc.Name, err.Error())
				}
			}
		}()
	}
}

type output struct {
	ns *corev1.Namespace
	sa *corev1.ServiceAccount
	secret *corev1.Secret
	cm *corev1.ConfigMap
	deployment *appsv1.Deployment
}

func create(ctx context.Context, client kubernetes.Interface, prefix string) (o *output, err error) {
	o = &output{}

	ns, err := client.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: prefix,
			Labels: map[string]string{
				"clusterloader": "true",
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return
	}

	o.ns = ns
	namespace := o.ns.GetName()

	sa, err := client.CoreV1().ServiceAccounts(namespace).Create(ctx, &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: prefix,
			Labels: map[string]string{
				"clusterloader": "true",
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return
	}

	o.sa = sa
	_, err = client.CoreV1().ServiceAccounts(namespace).Get(ctx, o.sa.GetName(), metav1.GetOptions{})
	if err != nil {
		return
	}

	secret, err := client.CoreV1().Secrets(namespace).Create(ctx, &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: prefix,
			Labels: map[string]string{
				"clusterloader": "true",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"key1": []byte("aGVsbG8gd29ybGQgMQo="),
			"key2": []byte("aGVsbG8gd29ybGQgMgo="),
		},
	}, metav1.CreateOptions{})
	if err != nil {
		return
	}

	o.secret = secret
	_, err = client.CoreV1().Secrets(namespace).Get(ctx, o.secret.GetName(), metav1.GetOptions{})
	if err != nil {
		return
	}

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
		return
	}

	o.cm = cm
	_, err = client.CoreV1().ConfigMaps(namespace).Get(ctx, o.cm.GetName(), metav1.GetOptions{})
	if err != nil {
		return
	}

	deployment := new(namespace, prefix, o.sa.GetName(), o.cm.GetName(), o.secret.GetName())
	deployment, err = client.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return
	}

	o.deployment = deployment
	return
}

func new(namespace, prefix, sa, cm, secret string) *appsv1.Deployment {
	var replicas int32 = 1
	name := fmt.Sprintf("%s-%s", prefix, rand.String(10))

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name: name,
			Labels: map[string]string{
				"clusterloader": "true",
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"clusterloader": "true",
					"selector": name,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
					Labels: map[string]string{
						"clusterloader": "true",
						"selector": name,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: sa,
					Containers: []corev1.Container{
						{
							Name:            "clusterloader",
							Image:           "k8s.gcr.io/pause:3.1",
							ImagePullPolicy: corev1.PullIfNotPresent,
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU: resource.MustParse("10m"),
									corev1.ResourceMemory: resource.MustParse("10m"),
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "configmap",
									MountPath: "/var/configmap",
								},
								{
									Name:      "secret",
									MountPath: "/var/secret",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "secret",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: secret,
									DefaultMode: func() *int32 {
										v := int32(420)
										return &v
									}(),
								},
							},
						},
						{
							Name: "configmap",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: cm,
									},
								},
							},
						},
					},
					Tolerations: []corev1.Toleration{
						{
							Key: "node.kubernetes.io/not-ready",
							Operator: corev1.TolerationOpExists,
							Effect: corev1.TaintEffectNoExecute,
							TolerationSeconds: func() *int64 {
								v := int64(900)
								return &v
							}(),
						},
						{
							Key: "node.kubernetes.io/unreachable",
							Operator: corev1.TolerationOpExists,
							Effect: corev1.TaintEffectNoExecute,
							TolerationSeconds: func() *int64 {
								v := int64(900)
								return &v
							}(),
						},
					},
				},
			},
		},
	}
}
