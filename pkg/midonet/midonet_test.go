package midonet

import (
	"testing"

	"github.com/Sirupsen/logrus"
	midonettypes "github.com/barnettzqg/golang-midonetclient/types"
	"github.com/goodrain/midonet-cni/pkg/types"
)

func TestGetTenant(t *testing.T) {
	midonetManager, err := NewManager(types.Options{
		MidoNetRouterCIDR: "172.16.0.0/24",
		MidoNetBridgeCIDR: "192.168.0.0/30",
		MidoNetAPIConf: midonettypes.MidoNetAPIConf{
			URL:              []string{"http://127.0.0.1:8080"},
			UserName:         "admin",
			PassWord:         "6bJslp7jBs",
			ProjectID:        "admin",
			ProviderRouterID: "a25f9dc3-4e62-459d-91b1-bbb68a8a46e5",
			Version:          1,
		},
		ETCDConf: types.ETCDConf{
			URLs: []string{"http://server1:2379"},
		},
		Log: logrus.WithField("tenant_name", "testNamespace11"),
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	tenant, err := midonetManager.GetTenant("testNamespace11")
	if err != nil {
		t.Fatal(err.Error())
	}
	t.Log(tenant)
}

func TestCreateNewBridge(t *testing.T) {
	t.SkipNow()
	midonetManager, err := NewManager(types.Options{
		MidoNetRouterCIDR: "172.16.0.0/24",
		MidoNetBridgeCIDR: "192.168.0.0/30",
		MidoNetAPIConf: midonettypes.MidoNetAPIConf{
			URL:              []string{"http://127.0.0.1:8080"},
			UserName:         "admin",
			PassWord:         "6bJslp7jBs",
			ProjectID:        "admin",
			ProviderRouterID: "a25f9dc3-4e62-459d-91b1-bbb68a8a46e5",
			Version:          1,
		},
		ETCDConf: types.ETCDConf{
			URLs: []string{"http://server1:2379"},
		},
	})
	if err != nil {
		t.Fatal(err.Error())
	}
	err = midonetManager.CreateNewBridge(&midonettypes.Tenant{
		ID:   "testNamespace9",
		Name: "testNamespace9",
	})
	if err != nil {
		t.Fatal(err.Error())
	}

}
