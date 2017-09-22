package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"github.com/Sirupsen/logrus"
	midonetclient "github.com/barnettzqg/golang-midonetclient/midonet"
	midonettypes "github.com/barnettzqg/golang-midonetclient/types"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types/020"
	"github.com/coreos/etcd/client"
	"github.com/goodrain/midonet-cni/pkg/etcd"
	"github.com/goodrain/midonet-cni/pkg/ipam"
	"github.com/goodrain/midonet-cni/pkg/midonet"
	conf "github.com/goodrain/midonet-cni/pkg/types"
	"github.com/goodrain/midonet-cni/pkg/vethctrl"
)

// CmdAddK8s k8s cni
// 增强的幂等特性
func CmdAddK8s(args *skel.CmdArgs, options *conf.Options, hostname string) (types.Result, error) {
	k8sArgs := conf.K8sArgs{}
	err := types.LoadArgs(args.Args, &k8sArgs)
	if err != nil {
		return nil, err
	}
	log := logrus.WithFields(logrus.Fields{"pod_name": k8sArgs.K8S_POD_NAME, "namespace": k8sArgs.K8S_POD_NAMESPACE, "action": "Add"})
	options.Log = log
	//create etcd client
	netContainerID := string(k8sArgs.K8S_POD_INFRA_CONTAINER_ID)
	c, err := etcd.CreateETCDClient(options.ETCDConf)
	if err != nil {
		log.Error("create etcd client error.", err.Error())
		return nil, err
	}
	//return last result if exist
	kapi := client.NewKeysAPI(c)
	res, err := kapi.Get(context.Background(), "/midonet-cni/result/"+netContainerID, nil)
	if err == nil {
		var r types020.Result
		err := json.Unmarshal([]byte(res.Node.Value), &r)
		if err == nil {
			log.Debugf("container(%s) result is exit.return old data.", netContainerID)
			return &r, nil
		}
	}
	//构建本机端口映射

	var bindingIP net.IPNet
	var bindingGetway net.IP

	midonetManager, err := midonet.NewManager(*options)
	if err != nil {
		return nil, err
	}
	//获取并验证租户是否存在
	tenant, err := midonetManager.GetTenant(string(k8sArgs.K8S_POD_NAMESPACE))
	if err != nil {
		return nil, err
	}
	ipamManager, err := ipam.CreateEtcdIpam(*options)
	if err != nil {
		logrus.Error("Create ipam error where get new ip for create pod")
		return nil, err
	}
	//分配新IP
	ipinfo, err := ipamManager.GetNewIP(tenant, netContainerID)
	if err != nil {
		//IP分配完成，创建新的网桥
		if err.Error() == "CreateNewBridge" {
			err = midonetManager.CreateNewBridge(tenant)
			if err != nil {
				return nil, err
			}
			ipinfo, err = ipamManager.GetNewIP(tenant, netContainerID)
		} else {
			return nil, err
		}
	}
	if err != nil {
		return nil, fmt.Errorf("The IP allocation error: %s", err.Error())
	}
	//绑定网卡
	err = midonetManager.Bingding("vif"+args.ContainerID[:12], tenant.ID, netContainerID)
	if err != nil {
		log.Error("Bingding veth to midonet bridge port error. will try", err.Error())
		err = midonetManager.Bingding("vif"+args.ContainerID[:12], tenant.ID, netContainerID)
		if err != nil {
			return nil, err
		}
	}
	bindingGetway = ipinfo.Getway
	bindingIP = ipinfo.IPNet
	result := &types020.Result{
		CNIVersion: "0.2.0",
		IP4: &types020.IPConfig{
			Gateway: bindingGetway,
			IP:      bindingIP,
		},
	}
	//创建veth对
	ctrl := vethctrl.GetVethCtrl(options.VethCtrlType, log)
	err = ctrl.DoNetworking(args, options, result)
	if err != nil {
		log.Error("Create and binding ip error,", err.Error())
		return nil, err
	}
	reData, err := json.Marshal(result)
	if err == nil {
		_, cerr := kapi.Set(context.Background(), "/midonet-cni/result/"+netContainerID, string(reData), nil)
		if cerr != nil {
			log.Warn("save result error.")
		}
	}
	return result, err
}

//CmdDelK8s 删除
func CmdDelK8s(args *skel.CmdArgs, options *conf.Options, hostname string) (types.Result, error) {
	if options.IPAM.Type != "etcd" {
		return nil, fmt.Errorf("ipam type only support etcd")
	}
	k8sArgs := conf.K8sArgs{}
	err := types.LoadArgs(args.Args, &k8sArgs)
	if err != nil {
		return nil, err
	}
	log := logrus.WithFields(logrus.Fields{"pod_name": k8sArgs.K8S_POD_NAME, "namespace": k8sArgs.K8S_POD_NAMESPACE, "action": "Delete"})
	options.Log = log
	netContainerID := string(k8sArgs.K8S_POD_INFRA_CONTAINER_ID)
	tenantID := string(k8sArgs.K8S_POD_NAMESPACE)
	c, err := etcd.CreateETCDClient(options.ETCDConf)
	if err != nil {
		logrus.Error("create etcd client error.", err.Error())
		return nil, err
	}
	kapi := client.NewKeysAPI(c)

	res, err := kapi.Delete(context.Background(), "/midonet-cni/result/"+netContainerID, nil)
	if err != nil && !client.IsKeyNotFound(err) {
		log.Error("Delete cni result error.", err.Error())
	}
	res, err = kapi.Get(context.Background(), fmt.Sprintf("/midonet-cni/bingding/%s/%s", tenantID, netContainerID), nil)
	if err == nil {
		bindingInfoStr := res.Node.Value
		bindingInfo := midonettypes.HostInterfacePort{}
		err := json.Unmarshal([]byte(bindingInfoStr), &bindingInfo)
		if err == nil {
			client, err := midonetclient.NewClient(&options.MidoNetAPIConf)
			if err == nil {
				for i := 2; i > 0; i-- {
					err := client.DeleteBinding(&bindingInfo)
					if err == nil {
						break
					} else {
						log.Error("Delete binding error.", err.Error())
					}
				}
			}
		}
		_, err = kapi.Delete(context.Background(), fmt.Sprintf("/midonet-cni/bingding/%s/%s", tenantID, netContainerID), nil)
		if err != nil && !client.IsKeyNotFound(err) {
			log.Error("Delete bingding status error.", err.Error())
		}
	}

	res, err = kapi.Get(context.Background(), fmt.Sprintf("/midonet-cni/ip/pod/%s/%s", tenantID, netContainerID), nil)
	if err == nil {
		i, err := ipam.CreateEtcdIpam(*options)
		if err == nil {
			err := i.ReleaseIP(tenantID, res.Node.Value)
			if err != nil {
				log.Error("Release ip when delete pod error,", err.Error())
			}
		}
		res, err = kapi.Delete(context.Background(), fmt.Sprintf("/midonet-cni/ip/pod/%s/%s", tenantID, netContainerID), nil)
		if err != nil && !client.IsKeyNotFound(err) {
			log.Error("Delete used ip info error.", err.Error())
		}
	}

	res, err = kapi.Get(context.Background(), fmt.Sprintf("/midonet-cni/bingding/%s", tenantID), &client.GetOptions{})
	if err == nil {
		if res.Node.Nodes == nil || res.Node.Nodes.Len() == 0 {
			manager, err := midonet.NewManager(*options)
			if err != nil {
				logrus.Error("Create midonet manager error when delete tenant.", err.Error())
			} else {
				err := manager.DeleteTenant(tenantID)
				if err != nil {
					logrus.Error("Delete tenant error.", err.Error())
				}
			}
		}
	}
	return &types020.Result{}, nil
}
