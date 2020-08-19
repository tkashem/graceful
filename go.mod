module github.com/tkashem/graceful

go 1.13

require (
	github.com/imdario/mergo v0.3.8 // indirect

	github.com/openshift/api v0.0.0-20200803131051-87466835fcc0
	github.com/openshift/client-go v0.0.0-20200729195840-c2b1adc6bed6
	github.com/prometheus/client_golang v1.7.1
	github.com/stretchr/testify v1.4.0
	golang.org/x/oauth2 v0.0.0-20200107190931-bf48bf16ab8d // indirect
	golang.org/x/time v0.0.0-20191024005414-555d28b269f0

	// pin to v0.19.0-rc.4
	k8s.io/api v0.19.0-rc.4
	k8s.io/apimachinery v0.19.0-rc.4
	k8s.io/apiserver v0.19.0-rc.4
	k8s.io/client-go v0.19.0-rc.4
	k8s.io/component-base v0.19.0-rc.4
	k8s.io/klog v1.0.0
)
