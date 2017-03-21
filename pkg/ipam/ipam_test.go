package ipam

import (
	"testing"

	midonettypes "github.com/barnettzqg/golang-midonetclient/types"
	"github.com/barnettzqg/midonet-cni/pkg/types"
)

func TestGetRouterIP(t *testing.T) {
	t.Log("TestGetRouterIP")
	ipam, err := CreateEtcdIpam(types.Options{
		MidoNetRouterCIDR: "172.16.0.0/30",
		ETCDConf: types.ETCDConf{
			URLs: []string{"http://server2:2379"},
		},
	})
	if err != nil {
		t.Error(err.Error())
	}
	t.Log(ipam.GetRouterIP())
}

func TestGetNewIP(t *testing.T) {
	t.Log("TestGetNewIP")
	ipam, err := CreateEtcdIpam(types.Options{
		MidoNetRouterCIDR: "172.16.0.0/30",
		ETCDConf: types.ETCDConf{
			URLs: []string{"http://server1:2379"},
		},
	})
	if err != nil {
		t.Error(err.Error())
	}
	t.Log(ipam.GetNewIP(&midonettypes.Tenant{
		ID:   "testNamespace9",
		Name: "testNamespace9",
	}, ""))
}
