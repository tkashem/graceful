## Background
The control plane component(s) such as `kubelet` uses an internal load balancer to talk to `kube-apiserver`. While 
`kube-apiserver` is rolling out we want to ensure that:
* New connection(s) are accepted within a certain grace period.
* The in-flight connections are given enough time to complete. 
 
Related BZ: https://bugzilla.redhat.com/show_bug.cgi?id=1802534

In GCE we use an `Internal TCP Load Balancer` to route internal traffic to the API server. GCP internal load balancer has the following characteristics:
* It is not `proxy` based.
* A request is always sent to the VM that makes the request, and health check information is ignored. This implies that any request originating from a control plane component on a master node is always routed to the `kube-apiserver` on the same node irrespective of whether `/readyz` reports a failure.

## Objective
We have the following objectives:
* Write a test suite that we can use to reproduce this issue consistently.
* Find a solution to the issue.

## Configuration:
`kube-apiserver` has the following configuration(s) to deal with graceful termination.
* `shutdown-delay-duration` is set to 70s
```bash
kubectl -n openshift-kube-apiserver get cm config -o json | jq -r '.data."config.yaml"' | jq '.apiServerArguments."shutdown-delay-duration"'
[
  "70s"
]
```

* `RequestTimeout` is by default set to `60s`.
https://github.com/kubernetes/kubernetes/blob/master/staging/src/k8s.io/apiserver/pkg/server/config.go#L300

* `terminationGracePeriodSeconds` is set to `135s`.
https://github.com/openshift/cluster-kube-apiserver-operator/blob/master/bindata/v4.1.0/kube-apiserver/pod.yaml#L154
```
terminationGracePeriodSeconds: 135 # bit more than 70s (minimal termination period) + 60s (apiserver graceful termination)
```

* no `preStopHook` defined.


### Test Strategy:
The test will have the following characteristics:
* The test will run inside a `Pod`.
* The test `Pod` will have `hostNetwork` enabled (`hostnetwork: true`) to utilize the host network used by the control
  plane components like kubelet. This ensures that the test avoids kubernetes `Service` network and uses the host's network
  to reach the `kube-apiserver`. 
* We want to use the address of the internal load balancer to reach the `kube-apiserver`. We access the `kubeconfig` 
  used by kubelet on the node and use the `Host` URL. This is always set to the internal load balancer URL.
```yaml
apiVersion: apps/v1
kind: DaemonSet
spec:
  template:
    spec:
      hostNetwork: true
      securityContext:
        runAsUser: 0
      containers:
        - name: graceful-test
          securityContext:
            privileged: true
          command:
            - /usr/bin/graceful-termination-test
            - -kubelet-kubeconfig=/var/lib/kubelet/kubeconfig
          volumeMounts:
          - mountPath: /var/lib/kubelet
            name: kubeconfig
      volumes:
      - name: kubeconfig
        hostPath:
          path: /var/lib/kubelet
```
* The test will run on each master node and will concurrently (10 go routines) issue request(s) to the API server. 
    * When `kube-apiserver` restarts on a particular node, `/readyz` will not report `200` and the internal load balancer
      should not forward any traffic to it until the new process starts reporting `200` on `readyz` again. In essence, 
      we should not see any new or exisitng connections being dropped by this API server.
* While the test is running, we will force a `kube-apiserver` roll out.

## Test Runs (Test with Delete, Post, Get)
### GCP 4.5 
![error on a master node](gcp-4.5-with-route-fix.png)

The errors we see:
* `unexpected EOF`
* `http2: server sent GOAWAY and closed the connection; LastStreamID=3602153, ErrCode=NO_ERROR, debug=""`

#### Broken Pipe Error
```
I0608 21:41:20.732640       1 kube-apiserver-rollout.go:71] kube-apiserver roll out event: event=Started pod=kube-apiserver-tkashem-ctllb-master-1.c.openshift-gce-devel.internal
E0608 21:45:21.938909       1 steps.go:17] step error=Delete https://api-int.tkashem.gcp.devcluster.openshift.com:6443/api/v1/namespaces/graceful-testxgt57/serviceaccounts/test-5wb8h: write tcp 10.0.0.6:52950->10.0.0.2:6443: write: broken pipe
E0608 21:45:21.939157       1 steps.go:17] step error=Post https://api-int.tkashem.gcp.devcluster.openshift.com:6443/api/v1/namespaces/graceful-testxgt57/secrets: write tcp 10.0.0.6:52950->10.0.0.2:6443: write: broken pipe
E0608 21:45:22.114469       1 steps.go:17] step error=Delete https://api-int.tkashem.gcp.devcluster.openshift.com:6443/api/v1/namespaces/graceful-testxgt57/serviceaccounts/test-wcs55: http2: server sent GOAWAY and closed the connection; LastStreamID=188667, ErrCode=NO_ERROR, debug=""
E0608 21:45:22.131398       1 steps.go:17] step error=Post https://api-int.tkashem.gcp.devcluster.openshift.com:6443/api/v1/namespaces/graceful-testxgt57/secrets: http2: server sent GOAWAY and closed the connection; LastStreamID=188667, ErrCode=NO_ERROR, debug=""
```

The following error `write tcp 10.0.0.6:52950->10.0.0.2:6443: write: broken pipe` may be related to the gcp route. `10.0.0.6` 
is the `master-1` node  and `10.0.0.2` is the internal load balancer address.


### AWS 4.5
![error on a master node](aws-4.5.png)
Error:
* `read tcp 10.0.160.64:36290->10.0.185.197:6443: read: connection reset by peer`

### GCP 4.4.7 
* we saw `connection reset by peer` error. This error occurred on each master node while kube-apiserver was rolling out
* The calls that failed are `delete` and `create`.   

**master-0 Node:**

*Test Log:*
```
I0609 13:26:44.833938       1 kube-apiserver-rollout.go:71] kube-apiserver roll out event: event=BackOff pod=kube-apiserver-ci-ln-x2y9dyk-f76d1-rg58w-master-0
E0609 13:28:50.763229       1 request.go:975] Unexpected error when reading response body: read tcp 10.0.0.2:35314->10.0.0.2:6443: read: connection reset by peer
E0609 13:28:50.763476       1 calls.go:41] getAllConfigMaps error=unexpected error when reading response body. Please retry. Original error: read tcp 10.0.0.2:35314->10.0.0.2:6443: read: connection reset by peer
E0609 13:28:50.763534       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-testcbdrd/serviceaccounts/test-rxfq6: read tcp 10.0.0.2:35314->10.0.0.2:6443: read: connection reset by peer
E0609 13:28:50.763604       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-testcbdrd/serviceaccounts/test-cq8dc: read tcp 10.0.0.2:35314->10.0.0.2:6443: read: connection reset by peer
E0609 13:28:50.763664       1 steps.go:17] step error=Post https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-testcbdrd/serviceaccounts: read tcp 10.0.0.2:35314->10.0.0.2:6443: read: connection reset by peer
E0609 13:28:50.763889       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-testcbdrd/serviceaccounts/test-g4gls: read tcp 10.0.0.2:35314->10.0.0.2:6443: read: connection reset by peer
E0609 13:28:50.763992       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-testcbdrd/secrets/test-mqqdv: read tcp 10.0.0.2:35314->10.0.0.2:6443: read: connection reset by peer
E0609 13:28:50.764121       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-testcbdrd/secrets/test-7v5cr: read tcp 10.0.0.2:35314->10.0.0.2:6443: read: connection reset by peer
E0609 13:28:50.764272       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-testcbdrd/configmaps/test-zkgrp: read tcp 10.0.0.2:35314->10.0.0.2:6443: read: connection reset by peer
E0609 13:28:50.764484       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-testcbdrd/configmaps/test-4nxfh: read tcp 10.0.0.2:35314->10.0.0.2:6443: read: connection reset by peer
E0609 13:28:50.764561       1 steps.go:17] step error=Post https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-testcbdrd/serviceaccounts: read tcp 10.0.0.2:35314->10.0.0.2:6443: read: connection reset by peer
I0609 13:28:51.035238       1 kube-apiserver-rollout.go:71] kube-apiserver roll out event: event=TerminationMinimalShutdownDurationFinished pod=kube-apiserver-ci-ln-x2y9dyk-f76d1-rg58w-master-0
```

![gcp-4.4.7-master-0](gcp-4.4.7-master-0-node.png)

#### master-1 Node:
*Test Log:*
```
E0609 13:31:02.580874       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-test8k45q/serviceaccounts/test-s8d99: read tcp 10.0.0.2:58314->10.0.0.2:6443: read: connection reset by peer
E0609 13:31:02.580975       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-test8k45q/secrets/test-kg49j: read tcp 10.0.0.2:58314->10.0.0.2:6443: read: connection reset by peer
E0609 13:31:02.581015       1 steps.go:17] step error=Post https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-test8k45q/configmaps: read tcp 10.0.0.2:58314->10.0.0.2:6443: read: connection reset by peer
E0609 13:31:02.581049       1 steps.go:17] step error=Post https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-test8k45q/serviceaccounts: read tcp 10.0.0.2:58314->10.0.0.2:6443: read: connection reset by peer
E0609 13:31:02.581080       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-test8k45q/secrets/test-kx7hb: read tcp 10.0.0.2:58314->10.0.0.2:6443: read: connection reset by peer
E0609 13:31:02.581111       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-test8k45q/configmaps/test-n7qfn: read tcp 10.0.0.2:58314->10.0.0.2:6443: read: connection reset by peer
E0609 13:31:02.581144       1 steps.go:17] step error=Post https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-test8k45q/configmaps: read tcp 10.0.0.2:58314->10.0.0.2:6443: read: connection reset by peer
E0609 13:31:02.581199       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-test8k45q/configmaps/test-zx5m2: read tcp 10.0.0.2:58314->10.0.0.2:6443: read: connection reset by peer
E0609 13:31:02.581230       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-test8k45q/secrets/test-xt2gd: read tcp 10.0.0.2:58314->10.0.0.2:6443: read: connection reset by peer
E0609 13:31:02.581267       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-test8k45q/serviceaccounts/test-p87jn: read tcp 10.0.0.2:58314->10.0.0.2:6443: read: connection reset by peer
I0609 13:31:02.851438       1 kube-apiserver-rollout.go:71] kube-apiserver roll out event: event=TerminationMinimalShutdownDurationFinished pod=kube-apiserver-ci-ln-x2y9dyk-f76d1-rg58w-master-1
```

![gcp-4.4.7-master-1](gcp-4.4.7-master-1-node.png)

#### master-2 Node:
*Test Log:*
```
I0609 13:24:18.763652       1 kube-apiserver-rollout.go:71] kube-apiserver roll out event: event=BackOff pod=kube-apiserver-ci-ln-x2y9dyk-f76d1-rg58w-master-2
E0609 13:26:01.508029       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-test84z29/serviceaccounts/test-5zr5t: read tcp 10.0.0.2:38082->10.0.0.2:6443: read: connection reset by peer
E0609 13:26:01.512939       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-test84z29/secrets/test-gwwlc: read tcp 10.0.0.2:38082->10.0.0.2:6443: read: connection reset by peer
E0609 13:26:01.529066       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-test84z29/configmaps/test-m6prt: read tcp 10.0.0.2:38082->10.0.0.2:6443: read: connection reset by peer
E0609 13:26:01.529440       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-test84z29/serviceaccounts/test-k4vb2: read tcp 10.0.0.2:38082->10.0.0.2:6443: read: connection reset by peer
E0609 13:26:01.529604       1 steps.go:17] step error=Post https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-test84z29/configmaps: read tcp 10.0.0.2:38082->10.0.0.2:6443: read: connection reset by peer
E0609 13:26:01.530024       1 steps.go:17] step error=Delete https://api-int.ci-ln-x2y9dyk-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-test84z29/serviceaccounts/test-gx86b: read tcp 10.0.0.2:38082->10.0.0.2:6443: read: connection reset by peer
I0609 13:26:01.712843       1 kube-apiserver-rollout.go:71] kube-apiserver roll out event: event=TerminationMinimalShutdownDurationFinished pod=kube-apiserver-ci-ln-x2y9dyk-f76d1-rg58w-master-2
```

![gcp-4.4.7-master-2](gcp-4.4.7-master-2-node.png)


## Test (GET Only)

### GCP 4.4.7
```
E0609 21:03:39.162204       1 readonly-worker.go:16] step error=Get https://api-int.ci-ln-2k3dg0b-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/namespaces/graceful-testx5g6z: unexpected EOF
E0609 21:03:39.162305       1 calls.go:41] getAllConfigMaps error=Get https://api-int.ci-ln-2k3dg0b-f76d1.origin-ci-int-gce.dev.openshift.com:6443/api/v1/configmaps: unexpected EOF
```

![gcp-4.4.7-read-only-test](gcp-4.4.7-read-only-test.png)


### GCP 4.5 

#### master-1 node
```
E0609 21:34:21.045789       1 readonly-worker.go:16] step error=Get https://api-int.tkashem.gcp.devcluster.openshift.com:6443/api/v1/namespaces/graceful-testtvbsq: write tcp 10.0.0.6:51234->10.0.0.2:6443: write: broken pipe
E0609 21:34:21.050687       1 readonly-worker.go:16] step error=Get https://api-int.tkashem.gcp.devcluster.openshift.com:6443/api/v1/namespaces/graceful-testtvbsq: write tcp 10.0.0.6:51234->10.0.0.2:6443: write: broken pipe
E0609 21:34:21.053097       1 readonly-worker.go:16] step error=Get https://api-int.tkashem.gcp.devcluster.openshift.com:6443/api/v1/namespaces/graceful-testtvbsq: write tcp 10.0.0.6:51234->10.0.0.2:6443: write: broken pipe
E0609 21:34:21.053188       1 readonly-worker.go:16] step error=Get https://api-int.tkashem.gcp.devcluster.openshift.com:6443/api/v1/namespaces/graceful-testtvbsq: write tcp 10.0.0.6:51234->10.0.0.2:6443: write: broken pipe
```

![gcp-4.5-read-only-test](gcp-4.5-read-only-test.png)

#### master-0 node
```
E0609 21:34:21.036881       1 readonly-worker.go:16] step error=Get https://api-int.tkashem.gcp.devcluster.openshift.com:6443/api/v1/namespaces/graceful-testbclh6: unexpected EOF
E0609 21:34:21.036898       1 readonly-worker.go:16] step error=Get https://api-int.tkashem.gcp.devcluster.openshift.com:6443/api/v1/namespaces/graceful-testbclh6: unexpected EOF
E0609 21:34:21.036891       1 readonly-worker.go:16] step error=Get https://api-int.tkashem.gcp.devcluster.openshift.com:6443/api/v1/namespaces/graceful-testbclh6: unexpected EOF
E0609 21:34:21.036921       1 readonly-worker.go:16] step error=Get https://api-int.tkashem.gcp.devcluster.openshift.com:6443/api/v1/namespaces/graceful-testbclh6: unexpected EOF
E0609 21:34:21.036922       1 readonly-worker.go:16] step error=Get https://api-int.tkashem.gcp.devcluster.openshift.com:6443/api/v1/namespaces/graceful-testbclh6: unexpected EOF
I0609 21:34:21.036931       1 streamwatcher.go:114] Unexpected EOF during watch stream event decoding: unexpected EOF
I0609 21:34:21.037053       1 reflector.go:495] k8s.io/client-go/informers/factory.go:135: Watch close - *v1.Event total 36 items received
```

![gcp-4.5-read-only-test-master-0](gcp-4.5-read-only-test-master-0.png)
