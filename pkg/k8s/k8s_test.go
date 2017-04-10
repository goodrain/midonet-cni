package k8s

import (
	"testing"

	midonettypes "github.com/barnettzqg/golang-midonetclient/types"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/goodrain/midonet-cni/pkg/types"
)

func TestCmdAddK8s(t *testing.T) {
	option := &types.Options{
		MidoNetRouterCIDR: "172.16.0.0/24",
		MidoNetBridgeCIDR: "192.168.0.0/30",
		MidoNetAPIConf: midonettypes.MidoNetAPIConf{
			URL:              "http://127.0.0.1:8080/midonet-api",
			UserName:         "admin",
			PassWord:         "6bJslp7jBs",
			ProjectID:        "admin",
			ProviderRouterID: "a25f9dc3-4e62-459d-91b1-bbb68a8a46e5",
			KeystoneConf: midonettypes.KeystoneConf{
				URL:   "http://127.0.0.1:35357/v2.0",
				Token: "0897e7b78686feb934ff",
			},
			Version: 1,
		},
		ETCDConf: types.ETCDConf{
			URLs: []string{"http://127.0.0.1:2379"},
		},
		IPAM: types.IPAM{
			Type:  "etcd",
			Route: types.Route{},
		},
	}
	var containerID = "asdasdasdasdasad3"
	args := &skel.CmdArgs{
		ContainerID: containerID,
		Netns:       "/proc/xxxxx",
		IfName:      "eth0",
		Path:        "",
	}
	args.Args = "K8S_POD_INFRA_CONTAINER_ID=" + containerID + ";K8S_POD_NAMESPACE=namespace2;K8S_POD_NAME=podname-001"
	result, err := CmdAddK8s(args, option, "hostname")
	if err != nil {
		t.Fatal(err)
	}
	t.Log(result)
}
