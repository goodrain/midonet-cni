package main

import (
	"context"

	"github.com/Sirupsen/logrus"
	"github.com/barnettzqg/midonet-cni/pkg/etcd"
	"github.com/barnettzqg/midonet-cni/pkg/types"
	"github.com/coreos/etcd/client"
)

func main() {
	etcdClient, err := etcd.CreateETCDClient(types.ETCDConf{
		URLs: []string{"http://139.224.234.115:2379"},
	})
	if err != nil {
		logrus.Error("Create etcd client error,", err.Error())
	}
	kapi := client.NewKeysAPI(etcdClient)
	res, err := kapi.Get(context.Background(), "/midonet-cni/tenant/xxx", nil)
	logrus.Info(res)
	if cerr, ok := err.(client.Error); ok && cerr.Code == client.ErrorCodeKeyNotFound {
		logrus.Info("KEY NOT FOUND")
	}
}
