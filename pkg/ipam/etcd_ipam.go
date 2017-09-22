package ipam

import (
	"net"
	"strings"

	"golang.org/x/net/context"

	"encoding/json"
	"fmt"

	"github.com/Sirupsen/logrus"
	midonettypes "github.com/barnettzqg/golang-midonetclient/types"
	"github.com/coreos/etcd/client"
	"github.com/goodrain/midonet-cni/pkg/etcd"
	"github.com/goodrain/midonet-cni/pkg/types"
	"github.com/goodrain/midonet-cni/pkg/util"
)

//EtcdIpam 基于etcd的IP管理
type EtcdIpam struct {
	conf   types.Options
	client client.Client
	log    *logrus.Entry
}

//CreateEtcdIpam 创建客户端
func CreateEtcdIpam(conf types.Options) (*EtcdIpam, error) {
	if conf.Log == nil {
		conf.Log = logrus.WithField("defaultLog", true)
	}
	c, err := etcd.CreateETCDClient(conf.ETCDConf)
	if err != nil {
		conf.Log.Error("create etcd client error where create etcd ipam,", err.Error())
		return nil, err
	}
	ipam := &EtcdIpam{
		conf:   conf,
		client: c,
		log:    conf.Log,
	}
	return ipam, nil
}

func (i *EtcdIpam) createRouterAvailable() ([]string, error) {
	return i.refreshRouterAvailable(i.conf.MidoNetRouterCIDR)
}

func (i *EtcdIpam) refreshRouterAvailable(iprange string) (ips []string, err error) {

	_, err = client.NewKeysAPI(i.client).Set(context.Background(), "/midonet-cni/ip/router/available/iprange", iprange, &client.SetOptions{})
	if err != nil {
		if cerr, ok := err.(client.Error); ok {
			if cerr.Code == client.ErrorCodeNodeExist {
				client.NewKeysAPI(i.client).Update(context.Background(), "/midonet-cni/ip/router/available/iprange", iprange)
			}
		} else {
			return nil, err
		}
	}
	r, err := util.NewRange(iprange)
	if err != nil {
		return nil, err
	}
	//放弃前两个ip
	for i := 2; i > 0; i-- {
		if !r.Next() {
			return nil, fmt.Errorf("Available Router IP shortage")
		}
	}
	for i := 2; i > 0; i-- {
		ips = append(ips, r.StringSuffix())
		if !r.Next() {
			if len(ips) == 2 {
				return
			}
			return nil, fmt.Errorf("Available Router IP shortage")
		}
	}
	for {
		v := *r
		if !r.Next() {
			break
		}
		//将ip段可用IP写入etcd
		client.NewKeysAPI(i.client).Set(context.Background(), "/midonet-cni/ip/router/available/"+v.String(), v.StringSuffix(), nil)
	}

	return ips, nil
}

//GetRouterIP 获取routerips。成对获取 例如返回：[172.16.0.2/24 172.16.0.3/24]
func (i *EtcdIpam) GetRouterIP() ([]string, error) {
	log := i.log
	var ips []string
	m := etcd.New("/midonet-cni/router_ip", 20, i.client)
	if m == nil {
		return nil, fmt.Errorf("etcdsync.NewMutex failed")
	}
	err := m.Lock()
	if err != nil {
		return nil, fmt.Errorf("etcdsync.Lock failed")
	}
	defer func() {
		err = m.Unlock()
		if err != nil {
			log.Error("etcdsync.Unlock failed")
		} else {
			log.Debug("etcdsync.Unlock OK GetRouter")
		}
	}()

	log.Info("Start create new router ip")

	kapi := client.NewKeysAPI(i.client)
	res, err := kapi.Get(context.Background(), "/midonet-cni/ip/router/available", &client.GetOptions{Recursive: true})
	if err != nil {
		if cerr, ok := err.(client.Error); ok {
			if cerr.Code == client.ErrorCodeKeyNotFound {
				logrus.Info("Initialize the filling the router available IP pool.")
				return i.createRouterAvailable()
			}
		} else {
			return nil, err
		}
	}
	if res.Node == nil || !res.Node.Dir {
		log.Warn("Initialize the filling the router available IP pool because available node")
		return i.createRouterAvailable()
	}
	if res.Node.Nodes.Len() < 3 { //可以node小于3，则放弃本网段，注入新网段IP
		res, err := kapi.Get(context.Background(), "/midonet-cni/ip/router/available/iprange", &client.GetOptions{})
		if err != nil {
			logrus.Error("Get old iprange info error from etcd.", err.Error())
			return nil, err
		}
		ipnet := res.Node.Value
		new, err := util.GetNextCIDR(ipnet)
		if err != nil {
			return nil, err
		}
		return i.refreshRouterAvailable(new)
	}
	for _, node := range res.Node.Nodes {
		if !strings.HasSuffix(node.Key, "/iprange") {
			_, err := kapi.Delete(context.Background(), node.Key, &client.DeleteOptions{})
			if err != nil {
				continue
			}
			ips = append(ips, node.Value)
			if len(ips) == 2 {
				break
			}
		}
	}
	return ips, nil
}

//CreateIPsForTenantBridge 创建IP可用池为租户网桥
func (i *EtcdIpam) CreateIPsForTenantBridge(tenant midonettypes.Tenant, iprange string) error {
	_, err := client.NewKeysAPI(i.client).Set(context.Background(), "/midonet-cni/ip/pod/"+tenant.ID+"/available/iprange", iprange, &client.SetOptions{})
	if err != nil {
		if cerr, ok := err.(client.Error); ok {
			if cerr.Code == client.ErrorCodeNodeExist {
				client.NewKeysAPI(i.client).Update(context.Background(), "/midonet-cni/ip/pod/"+tenant.ID+"/available/iprange", iprange)
			}
		} else {
			return err
		}
	}
	if util.RangeLength(iprange) < 4 {
		return fmt.Errorf("This ip range (%s) not have available ip ", iprange)
	}
	r, err := util.NewRange(iprange)
	if err != nil {
		return err
	}
	//放弃前两个ip
	for i := 2; i > 0; i-- {
		if !r.Next() {
			return fmt.Errorf("Available Bridge IP shortage")
		}
	}
	for { //将ip段可用IP写入etcd,最后一个ip忽略
		v := *r
		if !r.Next() {
			break
		}
		//去除.254 IP,此IP是bridge得IP地址。不能占用，否则网络到不了ROUTER
		if strings.HasSuffix(v.String(), ".254") {
			continue
		}
		client.NewKeysAPI(i.client).Set(context.Background(), "/midonet-cni/ip/pod/"+tenant.ID+"/available/"+v.String(), v.StringSuffix(), nil)
	}

	return nil
}

//GetNewIP 获取bridge ip
//幂等操作，参数一样，返回ip信息一样
func (i *EtcdIpam) GetNewIP(tenant *midonettypes.Tenant, containerID string) (ipinfo types.IPInfo, err error) {
	log := i.log
	kapi := client.NewKeysAPI(i.client)

	res, err := kapi.Get(context.Background(), fmt.Sprintf("/midonet-cni/ip/pod/%s/%s", tenant.ID, containerID), nil)
	if err == nil {
		err := json.Unmarshal([]byte(res.Node.Value), &ipinfo)
		if err == nil {
			log.Debugf("container(%s) ip is exit.return old data.", containerID)
			return ipinfo, nil
		}
	}

	res, err = kapi.Get(context.Background(), "/midonet-cni/ip/pod/"+tenant.ID+"/available", &client.GetOptions{Recursive: true})
	if err != nil {
		if cerr, ok := err.(client.Error); ok {
			return ipinfo, fmt.Errorf(cerr.Message)
		}
		return ipinfo, err
	}
	if res.Node == nil || !res.Node.Dir {
		return ipinfo, fmt.Errorf("Initialize the filling the bridge available IP pool because available node")
	}
	if res.Node.Nodes.Len() < 2 { //可以node小于3，则放弃本网段，创建新bridge
		return ipinfo, fmt.Errorf("CreateNewBridge")
	}
	var ipData string
	for _, node := range res.Node.Nodes {
		if !strings.HasSuffix(node.Key, "/iprange") {
			_, err := kapi.Delete(context.Background(), node.Key, &client.DeleteOptions{})
			if err != nil {
				continue
			}
			//忽略错误数据.254结尾的IP
			if strings.HasSuffix(node.Key, ".254") {
				continue
			}
			ipData = node.Value
			break
		}
	}
	ip, ipn, err := net.ParseCIDR(ipData)
	if err != nil {
		log.Error("the ip that it get from etcd parse error, ", err.Error())
		return ipinfo, err
	}
	ipnet := net.IPNet{
		IP:   ip,
		Mask: ipn.Mask,
	}
	getway := util.Long2IP(util.IP2Long(net.ParseIP(ipn.IP.String())) + 254)
	ipinfo.ContainerID = containerID
	ipinfo.Getway = getway
	ipinfo.IPNet = ipnet

	res, err = kapi.Set(context.Background(), fmt.Sprintf("/midonet-cni/ip/pod/%s/%s", tenant.ID, containerID), ipData, nil)
	if err != nil {
		log.Warnf("save container(%s) ip info error,%s", containerID, err.Error())
	}

	log.Infof("Etcd ipam return ip info IPNET:%s,GETAWY:%s", ipnet, getway)
	return ipinfo, nil
}

//ReleaseIP 释放ip
func (i *EtcdIpam) ReleaseIP(tenantID, oldIP string) error {
	log := i.log
	kapi := client.NewKeysAPI(i.client)
	res, err := kapi.Get(context.Background(), "/midonet-cni/ip/pod/"+tenantID+"/available/iprange", nil)
	if err != nil {
		log.Error("Get now iprange error when release ip.", err.Error())
		return err
	}
	iprange := res.Node.Value
	_, ipnet, err := net.ParseCIDR(iprange)
	if err != nil {
		log.Error("ParseCIDR iprange error when release ip.", err.Error())
		return err
	}
	ip, oldIPNet, err := net.ParseCIDR(oldIP)
	if err != nil {
		log.Error("ParseCIDR oldIP error when release ip.", err.Error())
		return err
	}
	if ipnet.IP.String() == oldIPNet.IP.String() && ipnet.Mask.String() == oldIPNet.Mask.String() {
		log.Infof("oldIP(%s) could be released.", oldIP)
		l := etcd.New("/midonet-cni/tenant/"+tenantID+"/bridge/create", 20, i.client)
		if l == nil {
			return fmt.Errorf("etcdsync.NewMutex failed when release old ip")
		}
		err := l.Lock()
		if err != nil {
			return fmt.Errorf("etcdsync.Lock failed when release old ip")
		}
		defer func() {
			err = l.Unlock()
			if err != nil {
				log.Errorf("etcdsync.Unlock failed when release old ip")
			} else {
				log.Debug("etcdsync.Unlock OK when release old ip")
			}
		}()
		kapi.Set(context.Background(), "/midonet-cni/ip/pod/"+tenantID+"/available/"+ip.String(), oldIP, nil)
	}
	return nil
}

//ReleaseRouterIP 释放router ip
func (i *EtcdIpam) ReleaseRouterIP(ips []string) error {
	log := i.log
	kapi := client.NewKeysAPI(i.client)
	res, err := kapi.Get(context.Background(), "/midonet-cni/ip/router/available/iprange", nil)
	if err != nil {
		log.Error("Get now iprange error when release ip.", err.Error())
		return err
	}
	iprange := res.Node.Value
	_, ipnet, err := net.ParseCIDR(iprange)
	if err != nil {
		log.Error("ParseCIDR iprange error when release ip.", err.Error())
		return err
	}
	for _, oldIP := range ips {
		ip, oldIPNet, err := net.ParseCIDR(oldIP)
		if err != nil {
			log.Error("ParseCIDR oldIP error when release ip.", err.Error())
			return err
		}
		if ipnet.IP.String() == oldIPNet.IP.String() && ipnet.Mask.String() == oldIPNet.Mask.String() {
			log.Infof("oldIP(%s) could be released.", oldIP)
			l := etcd.New("/midonet-cni/router_ip", 20, i.client)
			if l == nil {
				return fmt.Errorf("etcdsync.NewMutex failed when release old ip")
			}
			err := l.Lock()
			if err != nil {
				return fmt.Errorf("etcdsync.Lock failed when release old ip")
			}
			defer func() {
				err = l.Unlock()
				if err != nil {
					log.Errorf("etcdsync.Unlock failed when release old ip")
				} else {
					log.Debug("etcdsync.Unlock OK when release old ip")
				}
			}()
			kapi.Set(context.Background(), "/midonet-cni/ip/router/available/"+ip.String(), oldIP, nil)
		}
	}
	return nil
}
