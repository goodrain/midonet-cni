package midonet

import (
	"net"

	"golang.org/x/net/context"

	"fmt"

	"encoding/json"

	"github.com/Sirupsen/logrus"
	midonetclient "github.com/barnettzqg/golang-midonetclient/midonet"
	midonettypes "github.com/barnettzqg/golang-midonetclient/types"
	"github.com/coreos/etcd/client"
	etcdclient "github.com/coreos/etcd/client"
	"github.com/goodrain/midonet-cni/pkg/etcd"
	"github.com/goodrain/midonet-cni/pkg/ipam"
	"github.com/goodrain/midonet-cni/pkg/types"
	"github.com/goodrain/midonet-cni/pkg/util"
)

//Manager midonet manager
type Manager struct {
	conf       types.Options
	client     *midonetclient.Client
	etcdClient etcdclient.Client
	log        *logrus.Entry
}

//NewManager  创建管理器
func NewManager(conf types.Options) (*Manager, error) {
	if conf.Log == nil {
		conf.Log = logrus.WithField("defaultLog", true)
	}
	c, err := midonetclient.NewClient(&conf.MidoNetAPIConf)
	if err != nil {
		conf.Log.Error("Create midonet client error.", err.Error())
		return nil, err
	}
	etcdClient, err := etcd.CreateETCDClient(conf.ETCDConf)
	if err != nil {
		conf.Log.Error("Create etcd client error,", err.Error())
		return nil, err
	}

	return &Manager{
		conf:       conf,
		client:     c,
		etcdClient: etcdClient,
		log:        conf.Log,
	}, nil
}

//GetTenant 获取租户
func (m *Manager) GetTenant(tenantID string) (*midonettypes.Tenant, error) {
	var tenant midonettypes.Tenant
	kapi := etcdclient.NewKeysAPI(m.etcdClient)
	res, err := kapi.Get(context.Background(), "/midonet-cni/tenant/"+tenantID+"/info", nil)
	if err != nil {
		if cerr, ok := err.(client.Error); ok {
			if cerr.Code == client.ErrorCodeKeyNotFound {
				tenant.ID = tenantID
				tenant.Name = tenantID
				tenant.Enabled = true
				tenant.Description = ""
				err := m.InitTenant(tenant)
				if err != nil {
					return nil, err
				}
				return &tenant, nil
			}
		} else {
			return nil, err
		}
	}
	tenantData := res.Node.Value
	err = json.Unmarshal([]byte(tenantData), &tenant)
	if err != nil {
		return nil, err
	}
	return &tenant, nil
}

//InitTenant 初始化midonet tenant
func (m *Manager) InitTenant(tenant midonettypes.Tenant) error {
	log := m.log
	log.Infof("Start init tenant,tenant_name:%s,tenant_id:%s", tenant.Name, tenant.ID)
	l := etcd.New("/midonet-cni/tenant/"+tenant.ID+"/create", 20, m.etcdClient)
	if l == nil {
		return fmt.Errorf("etcdsync.NewMutex failed")
	}
	err := l.Lock()
	if err != nil {
		return fmt.Errorf("etcdsync.Lock failed")
	}
	defer func() {
		err = l.Unlock()
		if err != nil {
			log.Errorf("etcdsync.Unlock failed where init tenant %s", tenant.ID)
		} else {
			log.Debug("etcdsync.Unlock OK")
		}
	}()
	if tenant.ID == "" {
		return fmt.Errorf("tenant id can not be empty where midonet init tenant")
	}
	kapi := etcdclient.NewKeysAPI(m.etcdClient)
	_, err = kapi.Get(context.Background(), "/midonet-cni/tenant/"+tenant.ID+"/info", nil)
	if err != nil {
		if cerr, ok := err.(etcdclient.Error); ok && cerr.Code == etcdclient.ErrorCodeKeyNotFound {
			log.Info("Begin init tenant to midonet")
			err := m.client.CreateTenant(&tenant)
			if err != nil {
				log.Error("Create tennat error where init midonet tenant ", err.Error())
				return err
			}
			//先初始化租户网络
			err = m.initNetwork(tenant)
			if err != nil {
				log.Error("init tenant network error.", err.Error())
				//删除创建的租户
				delerr := m.client.DeleteTenant(tenant.ID)
				if delerr != nil {
					log.Error("delete tenant error when init network have a error.", err.Error())
				}
				return err
			}
			//存储信息
			tenantData, err := json.Marshal(tenant)
			if err != nil {
				log.Error("Marshal tenant info error.", err.Error())
				return err
			}
			_, err = kapi.Set(context.Background(), "/midonet-cni/tenant/"+tenant.ID+"/info", string(tenantData), nil)
			if err != nil {
				if cerr, ok := err.(client.Error); ok {
					if cerr.Code == client.ErrorCodeNodeExist {
						return m.InitTenant(tenant) //重新获取。可能是其他进程已经创建
					}
					return cerr
				}
				return etcd.HandleError(err)
			}
			return nil
		}
		return etcd.HandleError(err)
	}
	return nil
}
func (m *Manager) createRouterPort(ip string, routerID *midonettypes.UUID) (*midonettypes.RouterPort, error) {
	log := m.log
	remoteIP, remoteIPNet, err := net.ParseCIDR(ip)
	if err != nil {
		log.Error("Parse the router port ip error.", err.Error())
		return nil, err
	}
	networkLength, _ := remoteIPNet.Mask.Size()
	remotePort := &midonettypes.RouterPort{
		NetworkAddress: remoteIPNet.IP.String(),
		PortAddress:    remoteIP.String(),
		NetworkLength:  networkLength,
	}
	remotePort.DeviceID = routerID
	err = m.client.CreateRouterPort(remotePort)
	if err != nil {
		log.Error("Create  router port  error", err.Error())
		return nil, err
	}
	log.Info("Create a router port ", remotePort)
	return remotePort, nil
}

//CreateDefaultRoute 创建租户router的路由规则
func (m *Manager) CreateDefaultRoute(localPort, remotePort *midonettypes.RouterPort, routerID, providerRouterID *midonettypes.UUID) error {
	log := m.log
	outRoute := &midonettypes.Route{
		ID:               midonettypes.CreateUUID(),
		NextHopGateway:   remotePort.PortAddress,
		SrcNetworkAddr:   "0.0.0.0",
		SrcNetworkLength: 0,
		DstNetworkAddr:   "0.0.0.0",
		DstNetworkLength: 0,
		Type:             "Normal",
		NextHopPort:      localPort.ID,
		RouterID:         routerID,
	}
	inRoute := &midonettypes.Route{
		ID:               midonettypes.CreateUUID(),
		SrcNetworkAddr:   "0.0.0.0",
		SrcNetworkLength: 0,
		DstNetworkAddr:   localPort.PortAddress,
		DstNetworkLength: 32, //此处是否使用32？ 此处必须使用32
		Type:             "Normal",
		NextHopPort:      remotePort.ID,
		RouterID:         providerRouterID,
	}
	err := m.client.CreateRoute(outRoute)
	if err != nil {
		log.Error("create out route for tenant router error.", err.Error())
		return err
	}
	log.Info("Create a out route ", outRoute.ID)
	err = m.client.CreateRoute(inRoute)
	if err != nil {
		log.Error("create in route for provider router error.", err.Error())
		delerr := m.client.DeleteRoute(routerID, outRoute.ID)
		if delerr != nil {
			log.Warn("delete out route error wehn create in route error")
		}
		return err
	}
	log.Info("Create a in route ", inRoute.ID)
	return nil
}

//CreateDefaultRule 创建租户router route rule
func (m *Manager) CreateDefaultRule(localPort, remotePort *midonettypes.RouterPort, router *midonettypes.Router) error {
	log := m.log
	ruleOne := &midonettypes.Rule{
		MatchForwardFlow: false,
		MatchReturnFlow:  true,
		FragmentPolicy:   "any",
		NoVlan:           false,
		CondInvert:       false,
		Position:         1,
		Type:             "accept",
		ChainID:          router.OutboundFilterID,
	}
	ruleTwo := &midonettypes.Rule{
		FragmentPolicy: "unfragmented",
		NoVlan:         false,
		CondInvert:     false,
		Position:       2,
		Type:           "snat",
		FlowAction:     "accept",
		InvOutPorts:    false,
		OutPorts:       []*midonettypes.UUID{localPort.ID},
		NatTargets:     []midonettypes.NatTarget{midonettypes.NatTarget{AddressFrom: localPort.PortAddress, AddressTo: localPort.PortAddress, PortTo: 65535, PortFrom: 1024}},
		ChainID:        router.OutboundFilterID,
	}
	err := m.client.CreateRule(ruleOne)
	if err != nil {
		log.Error("create rule one for OutboundFilter error.", err.Error())
		return err
	}
	err = m.client.CreateRule(ruleTwo)
	if err != nil {
		log.Error("create rule two for OutboundFilter error.", err.Error())
		delerr := m.client.DeleteRule(ruleOne.ID)
		if delerr != nil {
			log.Warn("delete rule error wehn create rule two error")
		}
		return err
	}
	ruleThree := &midonettypes.Rule{
		FlowAction:     "continue",
		InPorts:        []*midonettypes.UUID{localPort.ID},
		FragmentPolicy: "any",
		CondInvert:     false,
		Position:       1,
		InvInPorts:     false,
		InvNwProto:     false,
		Type:           "rev_snat",
		ChainID:        router.InboundFilterID,
	}
	err = m.client.CreateRule(ruleThree)
	if err != nil {
		log.Error("create rule three for InboundFilter error.", err.Error())
		delerr := m.client.DeleteRule(ruleOne.ID)
		if delerr != nil {
			log.Warn("delete rule error when create rule three error")
		}
		delerr = m.client.DeleteRule(ruleTwo.ID)
		if delerr != nil {
			log.Warn("delete rule error when create rule three error")
		}
		return err
	}
	kapi := client.NewKeysAPI(m.etcdClient)
	ruleData, err := json.Marshal([]*midonettypes.Rule{ruleOne, ruleTwo, ruleThree})
	if err == nil {
		kapi.Set(context.Background(), "/midonet-cni/tenant/"+router.TenantID+"/router-rule", string(ruleData), nil)
	}
	return nil
}
func (m *Manager) initNetwork(tenant midonettypes.Tenant) error {
	log := m.log
	log.Infof("Start init tenant network,tenant_name:%s,tenant_id:%s", tenant.Name, tenant.ID)
	//创建chains
	inChain := &midonettypes.Chain{
		ID:       midonettypes.CreateUUID(),
		Name:     fmt.Sprintf("%s upstream_in", tenant.Name),
		TenantID: tenant.ID,
	}
	outChain := &midonettypes.Chain{
		ID:       midonettypes.CreateUUID(),
		Name:     fmt.Sprintf("%s upstream_out", tenant.Name),
		TenantID: tenant.ID,
	}
	err := m.client.CreateChain(inChain)
	if err != nil {
		log.Error("Create in chain for tenant router error.", err.Error())
		return err
	}
	log.Info("Create a in chain ", inChain.ID)
	err = m.client.CreateChain(outChain)
	if err != nil {
		log.Error("Create out  chain for tenant router error.", err.Error())
		delerr := m.client.DeleteChain(inChain.ID)
		if delerr != nil {
			log.Warning("Delete in chain error when create out chain error.", err.Error())
		}
		return err
	}
	log.Info("Create a out chain ", outChain.ID)
	//创建router
	router := &midonettypes.Router{
		Name:             fmt.Sprintf("%s default router", tenant.Name),
		TenantID:         tenant.ID,
		InboundFilterID:  inChain.ID,
		OutboundFilterID: outChain.ID,
	}
	err = m.client.CreateRouter(router)
	if err != nil {
		delerr := m.client.DeleteChain(inChain.ID)
		if delerr != nil {
			log.Warning("Delete in chain error when create router error.", err.Error())
		}
		delerr = m.client.DeleteChain(outChain.ID)
		if delerr != nil {
			log.Warning("Delete out chain error when create router error.", err.Error())
		}
		return err
	}
	log.Info("Create a default router ", router.ID)
	//获取router 与 provider router 链接的ip
	ipam, err := ipam.CreateEtcdIpam(m.conf)
	if err != nil {
		return err
	}
	ips, err := ipam.GetRouterIP()
	if err != nil {
		log.Error("Get router port ips for tenant error.", err.Error())
		return err
	}
	if len(ips) != 2 {
		log.Error("Get router port ips for tenant length not is 2.")
		return fmt.Errorf("IPs for tenant router port inadequate")
	}
	//创建remote_router_port
	ProviderRouterID, err := midonettypes.String2UUID(m.conf.MidoNetAPIConf.ProviderRouterID)
	if err != nil {
		log.Error("Parse the Provider Router ID Error", err.Error())
		return err
	}
	remotePort, err := m.createRouterPort(ips[0], ProviderRouterID)
	if err != nil {
		log.Error("Create Provider router port  error", err.Error())
		delerr := m.client.DeleteChain(inChain.ID)
		if delerr != nil {
			log.Warning("Delete in chain error when create remotePort error.", err.Error())
		}
		delerr = m.client.DeleteChain(outChain.ID)
		if delerr != nil {
			log.Warning("Delete out chain error when create remotePort error.", err.Error())
		}
		delerr = m.client.DeleteRouter(router.ID)
		if delerr != nil {
			log.Warning("Delete router error when create remotePort error.", err.Error())
		}
		return err
	}
	//创建 local_royter_port
	localPort, err := m.createRouterPort(ips[1], router.ID)
	if err != nil {
		log.Error("Create tenant router  port error", err.Error())
		delerr := m.client.DeleteChain(inChain.ID)
		if delerr != nil {
			log.Warning("Delete in chain error when create localPort error.", err.Error())
		}
		delerr = m.client.DeleteChain(outChain.ID)
		if delerr != nil {
			log.Warning("Delete out chain error when create localPort error.", err.Error())
		}
		delerr = m.client.DeleteRouter(router.ID)
		if delerr != nil {
			log.Warning("Delete router error when create localPort error.", err.Error())
		}
		delerr = m.client.DeletePort(remotePort.ID)
		if delerr != nil {
			log.Warning("Delete remote port error when create localPort error.", err.Error())
		}

		return err
	}
	portlink := &midonettypes.PortLink{
		PortID: localPort.ID,
		PeerID: remotePort.ID,
	}
	//创建port 链接
	err = m.client.CreatePortLink(portlink)
	if err != nil {
		log.Error("Create  port link error", err.Error())
		delerr := m.client.DeletePort(remotePort.ID)
		if delerr != nil {
			log.Warning("Delete remote port error when create port link error.", err.Error())
		}
		delerr = m.client.DeletePort(localPort.ID)
		if delerr != nil {
			log.Warning("Delete remote port error when create port link error.", err.Error())
		}
		delerr = m.client.DeleteChain(inChain.ID)
		if delerr != nil {
			log.Warning("Delete in chain error when create port link error.", err.Error())
		}
		delerr = m.client.DeleteChain(outChain.ID)
		if delerr != nil {
			log.Warning("Delete out chain error when create port link error.", err.Error())
		}
		delerr = m.client.DeleteRouter(router.ID)
		if delerr != nil {
			log.Warning("Delete router error when create port link error.", err.Error())
		}

		return err
	}
	//创建默认的路由规则
	err = m.CreateDefaultRoute(localPort, remotePort, router.ID, ProviderRouterID)
	if err != nil {
		log.Error("Create default route error", err.Error())
		delerr := m.client.DeletePortLink(localPort.ID)
		if delerr != nil {
			log.Warning("Delete port link error when create default route error.", err.Error())
		}
		delerr = m.client.DeletePort(remotePort.ID)
		if delerr != nil {
			log.Warning("Delete remote port error when create default route error.", err.Error())
		}
		delerr = m.client.DeletePort(localPort.ID)
		if delerr != nil {
			log.Warning("Delete remote port error when create default route error.", err.Error())
		}
		delerr = m.client.DeleteChain(inChain.ID)
		if delerr != nil {
			log.Warning("Delete in chain error when create default route error.", err.Error())
		}
		delerr = m.client.DeleteChain(outChain.ID)
		if delerr != nil {
			log.Warning("Delete out chain error when create default route error.", err.Error())
		}
		delerr = m.client.DeleteRouter(router.ID)
		if delerr != nil {
			log.Warning("Delete router error when create default route error.", err.Error())
		}

		return err
	}
	err = m.CreateDefaultRule(localPort, remotePort, router)
	if err != nil {
		log.Error("Create default rule error", err.Error())
		delerr := m.client.DeletePortLink(localPort.ID)
		if delerr != nil {
			log.Warning("Delete port link error when create default route error.", err.Error())
		}
		delerr = m.client.DeletePort(remotePort.ID)
		if delerr != nil {
			log.Warning("Delete remote port error when create default route error.", err.Error())
		}
		delerr = m.client.DeletePort(localPort.ID)
		if delerr != nil {
			log.Warning("Delete remote port error when create default route error.", err.Error())
		}
		delerr = m.client.DeleteChain(inChain.ID)
		if delerr != nil {
			log.Warning("Delete in chain error when create default route error.", err.Error())
		}
		delerr = m.client.DeleteChain(outChain.ID)
		if delerr != nil {
			log.Warning("Delete out chain error when create default route error.", err.Error())
		}
		delerr = m.client.DeleteRouter(router.ID)
		if delerr != nil {
			log.Warning("Delete router error when create default route error.", err.Error())
		}
		return err
	}
	// 创建第一个bridge
	defaultBridge := &midonettypes.Bridge{
		Name:     fmt.Sprintf("%s bridge0", tenant.Name),
		TenantID: tenant.ID,
		ID:       midonettypes.CreateUUID(),
	}
	err = m.client.CreateBridge(defaultBridge)
	if err != nil {
		log.Error("Create default bridge error", err.Error())
		delerr := m.client.DeletePortLink(localPort.ID)
		if delerr != nil {
			log.Warning("Delete port link error when create default route error.", err.Error())
		}
		delerr = m.client.DeletePort(remotePort.ID)
		if delerr != nil {
			log.Warning("Delete remote port error when create default route error.", err.Error())
		}
		delerr = m.client.DeletePort(localPort.ID)
		if delerr != nil {
			log.Warning("Delete remote port error when create default route error.", err.Error())
		}
		delerr = m.client.DeleteChain(inChain.ID)
		if delerr != nil {
			log.Warning("Delete in chain error when create default route error.", err.Error())
		}
		delerr = m.client.DeleteChain(outChain.ID)
		if delerr != nil {
			log.Warning("Delete out chain error when create default route error.", err.Error())
		}
		delerr = m.client.DeleteRouter(router.ID)
		if delerr != nil {
			log.Warning("Delete router error when create default route error.", err.Error())
		}
		return err
	}
	log.Info("Create a default bridge ", defaultBridge.ID)
	// 创建默认bridge对应的租户可用ip池
	err = ipam.CreateIPsForTenantBridge(tenant, m.conf.MidoNetBridgeCIDR)
	if err != nil {
		log.Error("Create default bridge error", err.Error())
		delerr := m.client.DeleteBridges(defaultBridge.ID)
		if delerr != nil {
			log.Warning("Delete bridge error when CreateIPsForTenantBridge error.", err.Error())
		}
		delerr = m.client.DeletePortLink(localPort.ID)
		if delerr != nil {
			log.Warning("Delete port link error when CreateIPsForTenantBridge error.", err.Error())
		}
		delerr = m.client.DeletePort(remotePort.ID)
		if delerr != nil {
			log.Warning("Delete remote port error when CreateIPsForTenantBridge error.", err.Error())
		}
		delerr = m.client.DeletePort(localPort.ID)
		if delerr != nil {
			log.Warning("Delete remote port error when CreateIPsForTenantBridge error.", err.Error())
		}
		delerr = m.client.DeleteChain(inChain.ID)
		if delerr != nil {
			log.Warning("Delete in chain error when CreateIPsForTenantBridge error.", err.Error())
		}
		delerr = m.client.DeleteChain(outChain.ID)
		if delerr != nil {
			log.Warning("Delete out chain error when CreateIPsForTenantBridge error.", err.Error())
		}
		delerr = m.client.DeleteRouter(router.ID)
		if delerr != nil {
			log.Warning("Delete router error when CreateIPsForTenantBridge error.", err.Error())
		}
		return err
	}
	err = m.CreateBridgeLinkRouter(defaultBridge.ID, router.ID, m.conf.MidoNetBridgeCIDR)
	if err != nil {
		log.Error("Create bridge link router error.", err.Error())
		delerr := m.client.DeletePortLink(localPort.ID)
		if delerr != nil {
			log.Warning("Delete port link error when CreateBridgeLinkRouter error.", err.Error())
		}
		delerr = m.client.DeletePort(remotePort.ID)
		if delerr != nil {
			log.Warning("Delete remote port error when CreateBridgeLinkRouter error.", err.Error())
		}
		delerr = m.client.DeletePort(localPort.ID)
		if delerr != nil {
			log.Warning("Delete remote port error when CreateBridgeLinkRouter error.", err.Error())
		}
		delerr = m.client.DeleteChain(inChain.ID)
		if delerr != nil {
			log.Warning("Delete in chain error when CreateBridgeLinkRouter error.", err.Error())
		}
		delerr = m.client.DeleteChain(outChain.ID)
		if delerr != nil {
			log.Warning("Delete out chain error when CreateBridgeLinkRouter error.", err.Error())
		}
		delerr = m.client.DeleteRouter(router.ID)
		if delerr != nil {
			log.Warning("Delete router error when CreateBridgeLinkRouter error.", err.Error())
		}
		return err
	}
	//存储相关信息，router信息，bridge信息，port信息，route信息
	kapi := client.NewKeysAPI(m.etcdClient)
	routerData, err := json.Marshal(router)
	if err == nil {
		kapi.Set(context.Background(), "/midonet-cni/tenant/"+tenant.ID+"/router", string(routerData), nil)
	}
	_, err = kapi.Set(context.Background(), "/midonet-cni/pod/router/"+router.ID.String(), ips[1]+":"+ips[0], nil)
	if err != nil {
		log.Warning("set router ip info to etcd error.")
	}
	portLinkData, err := json.Marshal(portlink)
	if err == nil {
		kapi.Set(context.Background(), "/midonet-cni/tenant/"+tenant.ID+"/router-link", string(portLinkData), nil)
	}
	bridgeData, err := json.Marshal(defaultBridge)
	if err == nil {
		kapi.Set(context.Background(), "/midonet-cni/tenant/"+tenant.ID+"/bridge/"+defaultBridge.ID.String(), string(bridgeData), nil)
	}
	_, err = kapi.Set(context.Background(), "/midonet-cni/tenant/"+tenant.ID+"/bridge/usage", defaultBridge.ID.String(), nil)
	if err != nil {
		log.Error("Save the used bridge to etcd error.", err.Error())
		delerr := m.client.DeletePortLink(localPort.ID)
		if delerr != nil {
			log.Warning("Delete port link error when Save the used bridge to etcd.", err.Error())
		}
		delerr = m.client.DeletePort(remotePort.ID)
		if delerr != nil {
			log.Warning("Delete remote port error when Save the used bridge to etcd", err.Error())
		}
		delerr = m.client.DeletePort(localPort.ID)
		if delerr != nil {
			log.Warning("Delete remote port error when Save the used bridge to etcd", err.Error())
		}
		delerr = m.client.DeleteChain(inChain.ID)
		if delerr != nil {
			log.Warning("Delete in chain error when Save the used bridge to etcd", err.Error())
		}
		delerr = m.client.DeleteChain(outChain.ID)
		if delerr != nil {
			log.Warning("Delete out chain error when Save the used bridge to etcd", err.Error())
		}
		delerr = m.client.DeleteRouter(router.ID)
		if delerr != nil {
			log.Warning("Delete router error when Save the used bridge to etcd", err.Error())
		}
		return err
	}

	return nil
}

//CreateBridgeLinkRouter 创建bridge与租户router的连接端口
// iprange 传入bridge 网段 例如：192.168.1.0/24
func (m *Manager) CreateBridgeLinkRouter(bridgeID, routerID *midonettypes.UUID, iprange string) error {
	log := m.log
	log.Infof("Start create bridge(%s) link router iprange:%s", bridgeID, iprange)
	_, ipn, err := net.ParseCIDR(iprange)
	if err != nil {
		return err
	}
	getway := util.Long2IP(util.IP2Long(net.ParseIP(ipn.IP.String())) + 254)
	networkLength, _ := ipn.Mask.Size()
	routerPort, err := m.createRouterPort(fmt.Sprintf("%s/%d", getway, networkLength), routerID)
	if err != nil {
		return err
	}
	bridgePort := &midonettypes.BridgePort{}
	bridgePort.DeviceID = bridgeID
	err = m.client.CreateBridgePort(bridgePort)
	if err != nil {
		delerr := m.client.DeletePort(routerPort.ID)
		if delerr != nil {
			log.Warning("Delete port error when create bridge port error.", err.Error())
		}
		return err
	}
	portlink := &midonettypes.PortLink{
		PortID: routerPort.ID,
		PeerID: bridgePort.ID,
	}
	//创建port 链接
	err = m.client.CreatePortLink(portlink)
	if err != nil {
		log.Error("Create router and bridge port link error", err.Error())
		delerr := m.client.DeletePort(routerPort.ID)
		if delerr != nil {
			log.Warning("Delete port error when create bridge port link error.", err.Error())
		}
		delerr = m.client.DeletePort(bridgePort.ID)
		if delerr != nil {
			log.Warning("Delete port error when create bridge port link error.", err.Error())
		}
		return err
	}

	routerRoute := &midonettypes.Route{
		RouterID:         routerID,
		SrcNetworkAddr:   "0.0.0.0",
		SrcNetworkLength: 0,
		DstNetworkAddr:   ipn.IP.String(),
		DstNetworkLength: networkLength,
		Type:             "Normal",
		NextHopPort:      routerPort.ID,
	}
	err = m.client.CreateRoute(routerRoute)
	if err != nil {
		log.Error("Create route for router link new bridge error.", err.Error())
		delerr := m.client.DeletePortLink(routerPort.ID)
		if delerr != nil {
			log.Warning("Delete port link error when create route error.", err.Error())
		}
		delerr = m.client.DeletePort(routerPort.ID)
		if delerr != nil {
			log.Warning("Delete port error when create route error.", err.Error())
		}
		delerr = m.client.DeletePort(bridgePort.ID)
		if delerr != nil {
			log.Warning("Delete port error when create route error.", err.Error())
		}
		return err
	}
	return nil
}

//Bingding 从租户bridge上创建端口,并绑定网卡
//幂等操作，containerID一样，多次调用无重复
func (m *Manager) Bingding(ifName, tenantID, containerID string) error {
	log := m.log
	kapi := client.NewKeysAPI(m.etcdClient)
	_, err := kapi.Get(context.Background(), fmt.Sprintf("/midonet-cni/bingding/%s/%s", tenantID, containerID), nil)
	if err == nil { //已经存在,不在绑定
		return nil
	}
	res, err := kapi.Get(context.Background(), "/midonet-cni/tenant/"+tenantID+"/bridge/usage", nil)
	if err != nil {
		log.Error("Get the used bridge to etcd error.", err.Error())
		return err
	}
	bridgeID, err := midonettypes.String2UUID(res.Node.Value)
	if err != nil {
		return err
	}
	hostID, err := midonettypes.String2UUID(m.conf.MidoNetHostUUID)
	if err != nil {
		return err
	}
	port := &midonettypes.BridgePort{}
	port.DeviceID = bridgeID

	err = m.client.CreateBridgePort(port)
	if err != nil {
		log.Error("create bridge port error.", err.Error())
		return err
	}
	hostport := &midonettypes.HostInterfacePort{
		HostID:        hostID,
		PortID:        port.ID,
		InterfaceName: ifName,
	}
	err = m.client.BindingInterface(hostport)
	if err != nil {
		log.Error("bingding interface to midonet bridge port error.", err.Error())
		return err
	}
	value, err := json.Marshal(hostport)
	if err != nil {
		m.client.DeleteBinding(hostport)
		log.Error("save the binding info error.", err.Error())
		return err
	}
	_, err = kapi.Set(context.Background(), fmt.Sprintf("/midonet-cni/bingding/%s/%s", tenantID, containerID), string(value), nil)
	if err != nil {
		m.client.DeleteBinding(hostport)
		log.Error("save the binding info error.", err.Error())
		return err
	}
	return nil
}

//CreateNewBridge 创建新的bridge ,注入新的IP
func (m *Manager) CreateNewBridge(tenant *midonettypes.Tenant) error {
	log := m.log
	l := etcd.New("/midonet-cni/tenant/"+tenant.ID+"/bridge/create", 20, m.etcdClient)
	if l == nil {
		return fmt.Errorf("etcdsync.NewMutex failed where create new bridge")
	}
	err := l.Lock()
	if err != nil {
		return fmt.Errorf("etcdsync.Lock failed where create new bridge")
	}
	defer func() {
		err = l.Unlock()
		if err != nil {
			log.Errorf("etcdsync.Unlock failed where create new bridge")
		} else {
			log.Debug("etcdsync.Unlock OK where create new bridge")
		}
	}()
	kapi := client.NewKeysAPI(m.etcdClient)
	//获取到锁后先查询，防止重复写入
	res, err := kapi.Get(context.Background(), "/midonet-cni/ip/pod/"+tenant.ID+"/available", &client.GetOptions{Recursive: true})
	if err != nil {
		if cerr, ok := err.(client.Error); ok {
			return fmt.Errorf(cerr.Message)
		}
		return err
	}
	if res.Node == nil || !res.Node.Dir {
		return fmt.Errorf("Initialize the filling the bridge available IP pool because available node")
	}
	if res.Node.Nodes.Len() < 2 { //可以node小于3，则放弃本网段，创建新bridge
		newBridge := &midonettypes.Bridge{
			TenantID: tenant.ID,
			ID:       midonettypes.CreateUUID(),
		}
		newBridge.Name = fmt.Sprintf("%s bridge-%s", tenant.Name, newBridge.ID)

		err = m.client.CreateBridge(newBridge)
		if err != nil {
			log.Error("Create new bridge error", err.Error())
			return err
		}

		//获取bridge 网段
		res, err = kapi.Get(context.Background(), "/midonet-cni/ip/pod/"+tenant.ID+"/available/iprange", &client.GetOptions{})
		if err != nil {
			log.Error("Get old iprange info error from etcd.", err.Error())
			return err
		}
		oldIPnet := res.Node.Value
		new, err := util.GetNextCIDR(oldIPnet)
		if err != nil {
			return err
		}
		ipamManager, err := ipam.CreateEtcdIpam(m.conf)
		if err != nil {
			return err
		}
		err = ipamManager.CreateIPsForTenantBridge(*tenant, new)
		if err != nil {
			return err
		}
		//连接bridge和router
		res, err := kapi.Get(context.Background(), "/midonet-cni/tenant/"+tenant.ID+"/router", &client.GetOptions{})
		if err != nil {
			log.Error("Get old iprange info error from etcd.", err.Error())
			return err
		}
		var router midonettypes.Router
		err = json.Unmarshal([]byte(res.Node.Value), &router)
		if err != nil {
			log.Error("Unmarshal router info error.", err.Error())
			return err
		}
		err = m.CreateBridgeLinkRouter(newBridge.ID, router.ID, new)
		if err != nil {
			log.Error("create bridge link tenant router error", err.Error())
			return err
		}
		_, err = kapi.Set(context.Background(), "/midonet-cni/tenant/"+tenant.ID+"/bridge/usage", newBridge.ID.String(), nil)
		if err != nil {
			log.Error("Save the used bridge to etcd error.", err.Error())
			return err
		}
	}
	return nil
}
