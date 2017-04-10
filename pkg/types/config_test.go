package types

import (
	"context"
	"testing"

	"github.com/coreos/etcd/client"
)

func TestDefault(t *testing.T) {
	c, _ := createETCDClient(ETCDConf{
		URLs: []string{"http://192.168.56.101:2379"},
	})
	value := `{"url":"http://127.0.0.1:8080/midonet-api","user_name":"admin","password":"","project_id":"admin","provider_router_id":"******","version":1,"keystone_conf":{"url":"http://127.0.0.1:35357/v2.0","token":"*****"}}`
	client.NewKeysAPI(c).Set(context.Background(), "/midonet-cni/config/midonet-api", value, nil)
	kubeValue := `{"k8s_api_root": "http://127.0.0.1:8080", "kubeconfig": "", "node_name": "tree01"}`
	client.NewKeysAPI(c).Set(context.Background(), "/midonet-cni/config/kubernetes", kubeValue, nil)
	option := &Options{
		ETCDConf: ETCDConf{
			URLs: []string{"http://192.168.56.101:2379"},
		},
	}
	err := option.Default()
	if err != nil {
		t.Fatal(err)
	}
	t.Log(option.MidoNetHostUUID)
	t.Log(option.MidoNetAPIConf)
	t.Log(option.Kubernetes)
}
