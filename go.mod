module github.com/skydive-project/skydive-flow-exporter

require (
	github.com/GehirnInc/crypt v0.0.0-20190301055215-6c0105aabd46 // indirect
	github.com/IBM/ibm-cos-sdk-go v1.3.0
	github.com/cnf/structhash v0.0.0-20180104161610-62a607eb0224 // indirect
	github.com/coreos/etcd v3.3.16+incompatible // indirect
	github.com/fatih/structs v1.1.0 // indirect
	github.com/gocarina/gocsv v0.0.0-20190927101021-3ecffd272576
	github.com/gophercloud/gophercloud v0.4.0 // indirect
	github.com/gorilla/mux v1.7.3 // indirect
	github.com/hashicorp/go-version v1.2.0 // indirect
	github.com/olivere/elastic v6.2.24+incompatible // indirect
	github.com/pierrec/xxHash v0.1.5 // indirect
	github.com/pmylund/go-cache v2.1.0+incompatible
	github.com/skydive-project/skydive v0.26.0
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.4.0
	github.com/vishvananda/netns v0.0.0-20190625233234-7109fa855b0f // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.1.0 // indirect
	golang.org/x/net v0.0.0-20191009170851-d66e71096ffb // indirect
)

// This section is copied as-is from skydive v0.26.0 go.mod
replace (
	github.com/digitalocean/go-libvirt => github.com/lebauce/go-libvirt v0.0.0-20190717144624-7799d804f7e4
	github.com/iovisor/gobpf => github.com/lebauce/gobpf v0.0.0-20190909090614-f9e9df81702a
	github.com/newtools/ebpf => github.com/nplanel/ebpf v0.0.0-20190918123742-99947faabce5
	github.com/prometheus/client_golang => github.com/prometheus/client_golang v0.9.3
	github.com/skydive-project/skydive/scripts/gendecoder => ./scripts/gendecoder
	github.com/spf13/viper v1.4.0 => github.com/lebauce/viper v0.0.0-20190903114911-3b7a98e30843
	github.com/vishvananda/netlink v1.0.0 => github.com/lebauce/netlink v0.0.0-20190122103356-fa328be7c8d2
	golang.org/x/sys => golang.org/x/sys v0.0.0-20190412213103-97732733099d
	golang.org/x/tools => golang.org/x/tools v0.0.0-20190925230517-ea99b82c7b93
	k8s.io/api => k8s.io/api v0.0.0-20191016110408-35e52d86657a
	k8s.io/apiextensions-apiserver => k8s.io/apiextensions-apiserver v0.0.0-20191016113550-5357c4baaf65
	k8s.io/apimachinery => k8s.io/apimachinery v0.0.0-20191004115801-a2eda9f80ab8
	k8s.io/apiserver => k8s.io/apiserver v0.0.0-20191016112112-5190913f932d
	k8s.io/cli-runtime => k8s.io/cli-runtime v0.0.0-20191016114015-74ad18325ed5
	k8s.io/client-go => k8s.io/client-go v0.0.0-20191016111102-bec269661e48
	k8s.io/cloud-provider => k8s.io/cloud-provider v0.0.0-20191016115326-20453efc2458
	k8s.io/cluster-bootstrap => k8s.io/cluster-bootstrap v0.0.0-20191016115129-c07a134afb42
	k8s.io/code-generator => k8s.io/code-generator v0.0.0-20191004115455-8e001e5d1894
	k8s.io/component-base => k8s.io/component-base v0.0.0-20191016111319-039242c015a9
	k8s.io/cri-api => k8s.io/cri-api v0.0.0-20190828162817-608eb1dad4ac
	k8s.io/csi-translation-lib => k8s.io/csi-translation-lib v0.0.0-20191016115521-756ffa5af0bd
	k8s.io/kube-aggregator => k8s.io/kube-aggregator v0.0.0-20191016112429-9587704a8ad4
	k8s.io/kube-controller-manager => k8s.io/kube-controller-manager v0.0.0-20191016114939-2b2b218dc1df
	k8s.io/kube-proxy => k8s.io/kube-proxy v0.0.0-20191016114407-2e83b6f20229
	k8s.io/kube-scheduler => k8s.io/kube-scheduler v0.0.0-20191016114748-65049c67a58b
	k8s.io/kubectl => k8s.io/kubectl v0.0.0-20191016120415-2ed914427d51
	k8s.io/kubelet => k8s.io/kubelet v0.0.0-20191016114556-7841ed97f1b2
	k8s.io/legacy-cloud-providers => k8s.io/legacy-cloud-providers v0.0.0-20191016115753-cf0698c3a16b
	k8s.io/metrics => k8s.io/metrics v0.0.0-20191016113814-3b1a734dba6e
	k8s.io/sample-apiserver => k8s.io/sample-apiserver v0.0.0-20191016112829-06bb3c9d77c9
)
