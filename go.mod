module github.com/ayaseen/health-check-runner

go 1.23.0

toolchain go1.23.8

require (
	github.com/alexmullins/zip v0.0.0-20180717182244-4affb64b04d0
	github.com/fatih/color v1.18.0
	github.com/openshift/api v0.0.0-20250411135543-10a8fa583797
	github.com/openshift/client-go v0.0.0-20250402181141-b3bad3b645f2
	github.com/prometheus/client_golang v1.22.0
	github.com/prometheus/common v0.63.0
	github.com/schollz/progressbar/v3 v3.18.0
	github.com/spf13/cobra v1.9.1
	github.com/spf13/viper v1.20.1
	github.com/vmware/govmomi v0.49.0
	k8s.io/api v0.32.3
	k8s.io/apimachinery v0.32.3
	k8s.io/client-go v1.5.2
	sigs.k8s.io/yaml v1.4.0
)

require (
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/emicklei/go-restful/v3 v3.11.0 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.20.2 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/google/gnostic-models v0.6.8 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/imdario/mergo v0.3.6 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/jpillora/backoff v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/matttproud/golang_protobuf_extensions v1.0.4 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/mwitkow/go-conntrack v0.0.0-20190716064945-2f068394615f // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/sagikazarmark/locafero v0.9.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.14.0 // indirect
	github.com/spf13/cast v1.7.1 // indirect
	github.com/spf13/pflag v1.0.6 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/atomic v1.11.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.37.0 // indirect
	golang.org/x/net v0.39.0 // indirect
	golang.org/x/oauth2 v0.25.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/term v0.31.0 // indirect
	golang.org/x/text v0.24.0 // indirect
	golang.org/x/time v0.8.0 // indirect
	google.golang.org/protobuf v1.36.5 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	k8s.io/kube-openapi v0.0.0-20241105132330-32ad38e42d3f // indirect
	k8s.io/utils v0.0.0-20250321185631-1f6e0b77f77e // indirect
	sigs.k8s.io/json v0.0.0-20241014173422-cfa47c3a1cc8 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.7.0 // indirect
)

replace (
	k8s.io/api => k8s.io/api v0.28.0
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.28.0
	k8s.io/apimachinery => k8s.io/apimachinery v0.28.0
	k8s.io/apiserver => k8s.io/apiserver v0.28.0
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.28.0
	k8s.io/client-go => k8s.io/client-go v0.28.0
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.28.0
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.28.0
	k8s.io/code-generator => k8s.io/code-generator v0.28.0
	k8s.io/component-base => k8s.io/component-base v0.28.0
	k8s.io/component-helpers => k8s.io/component-helpers v0.28.0
	k8s.io/controller-manager => k8s.io/controller-manager v0.28.0
	k8s.io/cri-api => k8s.io/cri-api v0.28.0
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.28.0
	k8s.io/dynamic-resource-allocation => k8s.io/dynamic-resource-allocation v0.28.0
	k8s.io/kms => k8s.io/kms v0.28.0
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.28.0
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.28.0
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.28.0
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.28.0
	k8s.io/kubectl => k8s.io/kubectl v0.28.0
	k8s.io/kubelet => k8s.io/kubelet v0.28.0
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.28.0
	k8s.io/metrics => k8s.io/metrics v0.28.0
	k8s.io/mount-utils => k8s.io/mount-utils v0.28.0
	k8s.io/pod-security-admission => k8s.io/pod-security-admission v0.28.0
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.28.0
)

replace k8s.io/sample-cli-plugin => k8s.io/sample-cli-plugin v0.28.0

replace k8s.io/sample-controller => k8s.io/sample-controller v0.28.0

replace k8s.io/endpointslice => k8s.io/endpointslice v0.28.0
