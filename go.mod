module github.com/xuanson2406/machine-controller-manager

go 1.16

require (
	github.com/Azure/azure-sdk-for-go v42.2.0+incompatible
	github.com/Azure/go-autorest/autorest v0.11.18
	github.com/Azure/go-autorest/autorest/adal v0.9.13
	github.com/Azure/go-autorest/autorest/to v0.3.0
	github.com/Masterminds/semver v1.5.0
	github.com/aliyun/alibaba-cloud-sdk-go v0.0.0-20180828111155-cad214d7d71f
	github.com/aws/aws-sdk-go v1.34.9
	github.com/cenkalti/backoff/v4 v4.1.0
	github.com/davecgh/go-spew v1.1.1
	github.com/go-git/go-git/v5 v5.4.2
	github.com/go-openapi/spec v0.19.5
	github.com/google/uuid v1.2.0
	github.com/gophercloud/gophercloud v0.7.0
	github.com/gophercloud/utils v0.0.0-20200204043447-9864b6f1f12f
	github.com/onsi/ginkgo v1.16.2
	github.com/onsi/gomega v1.11.0
	github.com/packethost/packngo v0.0.0-20181217122008-b3b45f1b4979
	github.com/prometheus/client_golang v1.12.1
	github.com/spf13/pflag v1.0.5
	github.com/xuanson2406/go-vcloud-director-fptcloud/v2 v2.1.5
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616
	golang.org/x/oauth2 v0.0.0-20210514164344-f6687ab2804c
	google.golang.org/api v0.44.0
	helm.sh/helm/v3 v3.7.1
	k8s.io/api v0.22.9
	k8s.io/apiextensions-apiserver v0.22.9
	k8s.io/apimachinery v0.22.9
	k8s.io/apiserver v0.22.9
	k8s.io/client-go v0.22.9
	k8s.io/cluster-bootstrap v0.22.9
	k8s.io/code-generator v0.22.9
	k8s.io/component-base v0.22.9
	k8s.io/klog/v2 v2.9.0
	k8s.io/kube-openapi v0.0.0-20211110012726-3cc51fd1e909 // keep this value in sync with k8s.io/apiserver
	k8s.io/utils v0.0.0-20211116205334-6203023598ed
)

require (
	github.com/gofrs/flock v0.8.1
	github.com/pkg/errors v0.9.1
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/Azure/go-autorest/autorest/validation v0.2.0 // indirect
	github.com/Masterminds/squirrel v1.5.2 // indirect
	github.com/emicklei/go-restful v2.9.6+incompatible // indirect
	golang.org/x/crypto v0.0.0-20211202192323-5770296d904e // indirect
	golang.org/x/tools v0.1.5 // indirect
	k8s.io/kubectl v0.22.4 // indirect
)

replace (
	github.com/rubenv/sql-migrate => github.com/rubenv/sql-migrate v0.0.0-20210614095031-55d5740dbbcc
	k8s.io/apimachinery => k8s.io/apimachinery v0.22.9
)
