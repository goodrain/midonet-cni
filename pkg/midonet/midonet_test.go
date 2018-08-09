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

func TestUUID(t *testing.T) {
	bridgeID := "5eb95bd1-919b-4ac1-bf6a-81bbdcaa947e"
	//hostID := "4202d021-20af-8b83-a249-e7a9a7e6573f"
	oldxxx := " a134eab8-3d42-40f5-84a5-fcf2b7a44b31"
	_, err := midonettypes.String2UUID(bridgeID)
	if err != nil {
		t.Fatal(err)
	}
	// _, err = midonettypes.String2UUID(hostID)
	// if err != nil {
	// 	t.Fatal(err)
	// }
	_, err = midonettypes.String2UUID(oldxxx)
	if err != nil {
		t.Fatal(err)
	}
}
