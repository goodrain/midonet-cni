package midonet

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"bytes"

	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/barnettzqg/golang-midonetclient/types"
)

//Client midonet api client
type Client struct {
	mediaV1Map map[string]string
	mediaV5Map map[string]string
	apiConf    types.MidoNetAPIConf
	token      types.Token
	version    int
}

//NewClient 创建客户端
func NewClient(conf *types.MidoNetAPIConf) (*Client, error) {
	if conf == nil {
		return nil, fmt.Errorf("midonet api conf can not be nil")
	}
	c := *conf
	client := &Client{}
	if conf.Version != 0 {
		logrus.Infof("Midonet api version %d", conf.Version)
		client.version = conf.Version
	} else {
		client.version = 1
	}
	client.apiConf = c
	client.mediaV1Map = map[string]string{
		"router":   "application/vnd.org.midonet.Router-v2+json",
		"bridge":   "application/vnd.org.midonet.Bridge-v1+json",
		"port":     "application/vnd.org.midonet.Port-v2+json",
		"portlink": "application/vnd.org.midonet.PortLink-v1+json",
		"chain":    "application/vnd.org.midonet.Chain-v1+json",
		"rule":     "application/vnd.org.midonet.Rule-v2+json",
		"route":    "application/vnd.org.midonet.Route-v1+json",
		"binding":  "application/vnd.org.midonet.HostInterfacePort-v1+json",
	}
	client.mediaV5Map = map[string]string{
		"router":   "application/vnd.org.midonet.Router-v3+json",
		"bridge":   "application/vnd.org.midonet.Bridge-v4+json",
		"port":     "application/vnd.org.midonet.Port-v3+json",
		"portlink": "application/vnd.org.midonet.PortLink-v1+json",
		"chain":    "application/vnd.org.midonet.Chain-v1+json",
		"rule":     "application/vnd.org.midonet.Rule-v2+json",
		"route":    "application/vnd.org.midonet.Route-v1+json",
		"binding":  "application/vnd.org.midonet.HostInterfacePort-v1+json",
	}
	if err := client.login(); err != nil {
		return nil, err
	}

	return client, nil
}
func (c *Client) getMedia(key string) string {
	if c.version == 1 {
		if media, ok := c.mediaV1Map[key]; ok {
			return media
		}
	} else {
		if media, ok := c.mediaV5Map[key]; ok {
			return media
		}
	}
	return "application/json"
}

func (c *Client) setHeader(header http.Header, mediaType string) {
	ContentType := c.getMedia(mediaType)
	if ContentType == "" {
		ContentType = "application/json"
	}
	header.Add("Content-Type", ContentType)
	header.Add("X-Auth-Token", c.token.Key)
	header.Add("Connection", "keep-alive")
}

func (c *Client) getHeaderForGet(mediaType string) http.Header {
	ContentType := c.getMedia(mediaType)
	if ContentType == "" {
		ContentType = "application/json"
	}
	header := make(http.Header, 0)
	header.Add("Accept", ContentType)
	header.Add("X-Auth-Token", c.token.Key)
	header.Add("Connection", "keep-alive")
	return header
}

func (c *Client) getHTTPClient() *http.Client {
	httpclient := http.DefaultClient
	httpclient.Timeout = 5 * time.Second
	return httpclient
}

func (c *Client) resultErr(res *http.Response) error {
	defer res.Body.Close()
	rebody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}
	return fmt.Errorf(string(rebody))
}

//login 登陆
func (c *Client) login() error {
	request, err := http.NewRequest("POST", c.apiConf.URL+"/login", nil)
	if err != nil {
		logrus.Errorln("midonet client create login request error.", err.Error())
		return err
	}
	request.SetBasicAuth(c.apiConf.UserName, c.apiConf.PassWord)
	request.Header.Add("X-Auth-Project", c.apiConf.ProjectID)
	response, err := c.getHTTPClient().Do(request)
	if err != nil || (response != nil && response.StatusCode != 200) {
		logrus.Error("midonet client do login request error.", err)
		return err
	}
	res := types.Token{}
	defer response.Body.Close()
	err = json.NewDecoder(response.Body).Decode(&res)
	if err != nil {
		logrus.Errorln("midonet client parse login response error.", err)
		return err
	}
	if res.Key != "" {
		c.token = res
		return nil
	}
	return fmt.Errorf("Don't Get key After Login")
}

//CreateBridge 创建租户网桥
func (c *Client) CreateBridge(bridge *types.Bridge) error {
	if bridge.ID == nil {
		bridge.ID = types.CreateUUID()
	}
	postData, err := json.Marshal(bridge)
	if err != nil {
		logrus.Error("Marshal bridge data error,", err.Error())
		return err
	}
	logrus.Info("create bridge:", string(postData))
	request, err := http.NewRequest("POST", c.apiConf.URL+"/bridges", bytes.NewReader(postData))
	if err != nil {
		logrus.Errorln("midonet client create post bridge request error.", err.Error())
		return err
	}
	c.setHeader(request.Header, "bridge")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("Create bridge error.", err.Error())
		return err
	}
	if res.StatusCode/100 == 2 {
		return nil
	}
	return c.resultErr(res)
}

//DeleteBridges 删除网桥
func (c *Client) DeleteBridges(bridgeID *types.UUID) error {
	if bridgeID == nil {
		return fmt.Errorf("bridge id can not be empty where delete bridge")
	}
	request, err := http.NewRequest("DELETE", c.apiConf.URL+"/bridges/"+bridgeID.String(), nil)
	if err != nil {
		logrus.Errorln("midonet client create delete bridge request error.", err.Error())
		return err
	}
	c.setHeader(request.Header, "bridge")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("Delete bridge error.", err.Error())
		return err
	}
	if res.StatusCode/100 == 2 {
		logrus.Infof("Delete bridge (%s) success", bridgeID)
		return nil
	}
	return c.resultErr(res)
}

//CreateBridgePort 创建申请租户网桥端口
func (c *Client) CreateBridgePort(bridgePort *types.BridgePort) error {
	if bridgePort.ID == nil {
		bridgePort.ID = types.CreateUUID()
	}
	if bridgePort.Type != "Bridge" {
		bridgePort.Type = "Bridge"
	}
	postData, err := json.Marshal(bridgePort)
	if err != nil {
		logrus.Error("Marshal bridge data error,", err.Error())
		return err
	}
	request, err := http.NewRequest("POST", c.apiConf.URL+fmt.Sprintf("/bridges/%s/ports", bridgePort.DeviceID), bytes.NewReader(postData))
	if err != nil {
		logrus.Errorln("midonet client create post bridge port request error.", err.Error())
		return err
	}
	c.setHeader(request.Header, "port")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("Create bridge port error.", err.Error())
		return err
	}
	if res.StatusCode/100 == 2 {
		return nil
	}
	return c.resultErr(res)
}

//CreateRouterPort 创建申请路由端口
func (c *Client) CreateRouterPort(routerPort *types.RouterPort) error {
	if routerPort.ID == nil {
		routerPort.ID = types.CreateUUID()
	}
	if routerPort.Type != "Router" {
		routerPort.Type = "Router"
	}
	postData, err := json.Marshal(routerPort)
	if err != nil {
		logrus.Error("Marshal router port data error,", err.Error())
		return err
	}
	request, err := http.NewRequest("POST", c.apiConf.URL+fmt.Sprintf("/routers/%s/ports", routerPort.DeviceID), bytes.NewReader(postData))
	if err != nil {
		logrus.Errorln("midonet client create post router port  request error.", err.Error())
		return err
	}
	c.setHeader(request.Header, "port")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("Create router port error.", err.Error())
		return err
	}
	if res.StatusCode/100 == 2 {
		return nil
	}
	return c.resultErr(res)
}

//DeletePort 删除端口
func (c *Client) DeletePort(portID *types.UUID) error {
	if portID == nil {
		return fmt.Errorf("port id can not be empty where delete port")
	}
	request, err := http.NewRequest("DELETE", c.apiConf.URL+"/ports/"+portID.String(), nil)
	if err != nil {
		logrus.Errorln("midonet client create delete port request error.", err.Error())
		return err
	}
	c.setHeader(request.Header, "port")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("Delete port error.", err.Error())
		return err
	}
	if res.StatusCode/100 == 2 {
		logrus.Infof("Delete port (%s) success", portID)
		return nil
	}
	return c.resultErr(res)
}

//CreatePortLink 创建端口连接
func (c *Client) CreatePortLink(link *types.PortLink) error {
	if link.PortID == nil {
		return errors.New("create port link port id can not be empty")
	}
	if link.PeerID == nil {
		return errors.New("create port link peer id can not be empty")
	}
	portID := link.PortID
	postData := []byte(fmt.Sprintf(`{"peerId":"%s"}`, link.PeerID.String()))
	request, err := http.NewRequest("POST", c.apiConf.URL+fmt.Sprintf("/ports/%s/link", portID.String()), bytes.NewReader(postData))
	if err != nil {
		logrus.Errorln("midonet client create post  port link  request error.", err.Error())
		return err
	}
	c.setHeader(request.Header, "portlink")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("Create port link error.", err.Error())
		return err
	}
	if res.StatusCode/100 == 2 {
		return nil
	}
	return c.resultErr(res)
}

//DeletePortLink 删除端口连接
func (c *Client) DeletePortLink(portID *types.UUID) error {
	if portID == nil {
		return errors.New("delete port link port id can not be empty")
	}
	request, err := http.NewRequest("DELETE", c.apiConf.URL+fmt.Sprintf("/ports/%s/link", portID.String()), nil)
	if err != nil {
		logrus.Errorln("midonet client delete  port link  request error.", err.Error())
		return err
	}
	c.setHeader(request.Header, "portlink")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("delete port link error.", err.Error())
		return err
	}
	if res.StatusCode/100 == 2 {
		return nil
	}
	return c.resultErr(res)
}

//CreateRouter 创建路由器
func (c *Client) CreateRouter(router *types.Router) error {
	if router.ID == nil {
		router.ID = types.CreateUUID()
	}
	postData, err := json.Marshal(router)
	if err != nil {
		logrus.Error("Marshal router data error,", err.Error())
		return err
	}
	request, err := http.NewRequest("POST", c.apiConf.URL+"/routers", bytes.NewReader(postData))
	if err != nil {
		logrus.Errorln("midonet client create post router request error.", err.Error())
		return err
	}
	c.setHeader(request.Header, "router")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("Create router error.", err.Error())
		return err
	}
	logrus.Info("Create router", res)
	if res.StatusCode/100 == 2 {
		return nil
	}
	return c.resultErr(res)
}

//DeleteRouter 删除router
func (c *Client) DeleteRouter(routerID *types.UUID) error {
	if routerID == nil {
		return fmt.Errorf("router id can not be empty where delete chain")
	}
	request, err := http.NewRequest("DELETE", c.apiConf.URL+fmt.Sprintf("/routers/%s", routerID), nil)
	if err != nil {
		logrus.Errorln("midonet client create delete router request error.", err.Error())
		return err
	}
	c.setHeader(request.Header, "router")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("Delete router error.", err.Error())
		return err
	}
	if res.StatusCode/100 == 2 {
		return nil
	}
	return c.resultErr(res)
}

//CreateRoute 创建route
func (c *Client) CreateRoute(route *types.Route) error {
	if route.RouterID == nil {
		return fmt.Errorf("router id can not be empty where create route")
	}
	if route.ID == nil {
		route.ID = types.CreateUUID()
	}
	postData, err := json.Marshal(route)
	if err != nil {
		logrus.Error("Marshal route data error,", err.Error())
		return err
	}
	request, err := http.NewRequest("POST", c.apiConf.URL+fmt.Sprintf("/routers/%s/routes", route.RouterID), bytes.NewReader(postData))
	if err != nil {
		logrus.Errorln("midonet client create route request error.", err.Error())
		return err
	}
	c.setHeader(request.Header, "route")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("Create route error.", err.Error())
		return err
	}
	if res.StatusCode/100 == 2 {
		return nil
	}
	return c.resultErr(res)
}

//DeleteRoute 删除route
func (c *Client) DeleteRoute(routerID, routeID *types.UUID) error {
	if routeID == nil {
		return fmt.Errorf("route id can not be empty when delete route")
	}
	if routerID == nil {
		return fmt.Errorf("router id can not be empty when delete route")
	}
	var path string
	if c.version == 5 {
		path = c.apiConf.URL + fmt.Sprintf("/routes/%s", routeID)
	} else {
		path = c.apiConf.URL + fmt.Sprintf("/routers/%s/routes/%s", routerID, routeID)
	}
	request, err := http.NewRequest("DELETE", path, nil)
	if err != nil {
		logrus.Errorln("midonet client delete  route request error.", err.Error())
		return err
	}
	c.setHeader(request.Header, "route")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("delete route error.", err.Error())
		return err
	}
	if res.StatusCode/100 == 2 {
		return nil
	}
	return c.resultErr(res)
}

//CreateChain 创建chain
func (c *Client) CreateChain(chain *types.Chain) error {
	if chain.ID == nil {
		chain.ID = types.CreateUUID()
	}
	if chain.TenantID == "" {
		return fmt.Errorf("tenant id can not be empty where create chain")
	}
	postData, err := json.Marshal(chain)
	if err != nil {
		logrus.Error("Marshal chain data error,", err.Error())
		return err
	}
	request, err := http.NewRequest("POST", c.apiConf.URL+"/chains", bytes.NewReader(postData))
	if err != nil {
		logrus.Errorln("midonet client create post chain request error.", err.Error())
		return err
	}
	c.setHeader(request.Header, "chain")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("Create chain error.", err.Error())
		return err
	}
	if res.StatusCode/100 == 2 {
		return nil
	}
	return c.resultErr(res)
}

//DeleteChain 删除chain
func (c *Client) DeleteChain(chainID *types.UUID) error {
	if chainID == nil {
		return fmt.Errorf("chain id can not be empty where delete chain")
	}
	request, err := http.NewRequest("DELETE", c.apiConf.URL+fmt.Sprintf("/chains/%s", chainID), nil)
	if err != nil {
		logrus.Errorln("midonet client create delete chain request error.", err.Error())
		return err
	}
	c.setHeader(request.Header, "chain")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("Delete chain error.", err.Error())
		return err
	}
	if res.StatusCode/100 == 2 {
		return nil
	}
	return c.resultErr(res)
}

//CreateRule 创建rule
func (c *Client) CreateRule(rule *types.Rule) error {
	if rule.ChainID == nil {
		return fmt.Errorf("chain id can not be empty where create rule")
	}
	if rule.ID == nil {
		rule.ID = types.CreateUUID()
	}
	postData, err := json.Marshal(rule)
	if err != nil {
		logrus.Error("Marshal rule data error,", err.Error())
		return err
	}
	request, err := http.NewRequest("POST", c.apiConf.URL+fmt.Sprintf("/chains/%s/rules", rule.ChainID), bytes.NewReader(postData))
	if err != nil {
		logrus.Errorln("midonet client create post rule request error.", err.Error())
		return err
	}
	c.setHeader(request.Header, "rule")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("Create rule error.", err.Error())
		return err
	}
	if res.StatusCode/100 == 2 {
		return nil
	}
	return c.resultErr(res)
}

// DeleteRule 删除rule
func (c *Client) DeleteRule(ruleID *types.UUID) error {
	if ruleID == nil {
		return fmt.Errorf("rule id can not e empty when delete rule")
	}

	request, err := http.NewRequest("DELETE", c.apiConf.URL+fmt.Sprintf("/rules/%s", ruleID), nil)
	if err != nil {
		logrus.Errorln("midonet client delete rule request error.", err.Error())
		return err
	}
	c.setHeader(request.Header, "rule")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("delete rule error.", err.Error())
		return err
	}
	if res.StatusCode/100 == 2 {
		return nil
	}
	return c.resultErr(res)
}

//BindingInterface 绑定网卡到租户端口
func (c *Client) BindingInterface(bindingInfo *types.HostInterfacePort) error {
	if bindingInfo.HostID == nil {
		return fmt.Errorf("host id can not be empty where binding interface")
	}
	if bindingInfo.PortID == nil {
		return fmt.Errorf("port id can not be empty where binding interface")
	}
	if bindingInfo.InterfaceName == "" {
		return fmt.Errorf("interface name can not be empty where binding interface")
	}
	postData, err := json.Marshal(bindingInfo)
	if err != nil {
		logrus.Error("Marshal bindingInfo data error,", err.Error())
		return err
	}
	request, err := http.NewRequest("POST", c.apiConf.URL+fmt.Sprintf("/hosts/%s/ports", bindingInfo.HostID), bytes.NewReader(postData))
	if err != nil {
		logrus.Errorln("midonet client create post binding interface request error.", err.Error())
		return err
	}
	c.setHeader(request.Header, "binding")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("Create binding error.", err.Error())
		return err
	}
	if res.StatusCode/100 == 2 {
		return nil
	}
	return c.resultErr(res)
}

//DeleteBinding 删除绑定
func (c *Client) DeleteBinding(bindingInfo *types.HostInterfacePort) error {
	if bindingInfo.HostID == nil {
		return fmt.Errorf("host id can not be empty where delete binding interface")
	}
	if bindingInfo.PortID == nil {
		return fmt.Errorf("port id can not be empty where delete binding interface")
	}
	if bindingInfo.InterfaceName == "" {
		return fmt.Errorf("interface name can not be empty where delete binding interface")
	}

	request, err := http.NewRequest("DELETE", c.apiConf.URL+fmt.Sprintf("/hosts/%s/ports/%s", bindingInfo.HostID, bindingInfo.PortID), nil)
	if err != nil {
		logrus.Errorln("midonet client create delete binding interface request error.", err.Error())
		return err
	}
	c.setHeader(request.Header, "binding")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("Delete binding error.", err.Error())
		return err
	}
	if res.StatusCode/100 == 2 {
		return nil
	}
	return c.resultErr(res)
}

//CreateTenant 创建租户
func (c *Client) CreateTenant(tenant *types.Tenant) error {
	if tenant.ID == "" {
		return fmt.Errorf("Tenant id can not be empty where create tenant")
	}
	postStruct := map[string]*types.Tenant{
		"tenant": tenant,
	}
	postData, err := json.Marshal(postStruct)
	if err != nil {
		logrus.Error("Marshal tenant data error,", err.Error())
		return err
	}
	logrus.Info("Create tenant:", string(postData))
	request, err := http.NewRequest("POST", c.apiConf.KeystoneConf.URL+"/tenants", bytes.NewReader(postData))
	if err != nil {
		logrus.Errorln("midonet client create post bridge request error.", err.Error())
		return err
	}
	request.Header.Add("X-Auth-Token", c.apiConf.KeystoneConf.Token)
	request.Header.Add("Connection", "keep-alive")
	request.Header.Add("content-type", "application/json")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("Create tenant error.", err.Error())
		return err
	}
	logrus.Info("Create tenant", res)
	if res.StatusCode/100 == 2 {
		return nil
	}
	return c.resultErr(res)
}

//DeleteTenant 删除租户
func (c *Client) DeleteTenant(tenantID string, all bool) error {
	if all {
		err := c.deleteAll(tenantID)
		if err != nil {
			logrus.Error("delete tenant all info error.", err.Error())
		}
	}
	if tenantID == "" {
		return fmt.Errorf("Tenant id can not be empty where delete tenant")
	}
	request, err := http.NewRequest("DELETE", c.apiConf.KeystoneConf.URL+"/tenants/"+tenantID, nil)
	if err != nil {
		logrus.Errorln("midonet client create delete tenant request error.", err.Error())
		return err
	}
	request.Header.Add("X-Auth-Token", c.apiConf.KeystoneConf.Token)
	request.Header.Add("Connection", "keep-alive")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("Delete tenant error.", err.Error())
		return err
	}
	if res.StatusCode/100 == 2 {
		return nil
	}
	return c.resultErr(res)
}

func (c *Client) deleteAll(tenantID string) error {
	var wait sync.WaitGroup
	wait.Add(3)
	//删除router相关
	go func() {
		routers := c.GetRouters(tenantID)
		if routers != nil && len(routers) > 0 {
			for _, router := range routers {
				ports := c.GetPortByRouter(router.ID.String())
				if ports != nil && len(ports) > 0 {
					for _, port := range ports {
						c.DeletePort(port.ID)
						c.DeletePort(port.PeerID)
						c.DeletePortLink(port.ID)
					}
				}
				routes := c.GetRoutes(router.ID.String())
				if routes != nil && len(routes) > 0 {
					for _, route := range routes {
						c.DeleteRoute(router.ID, route.ID)
					}
				}
				err := c.DeleteRouter(router.ID)
				if err != nil {
					logrus.Errorf("Delete router (%s) by tenant (%s) error.%s", router.ID, tenantID, err.Error())
				}
			}
		}
		wait.Done()
	}()
	//删除bridge相关
	go func() {
		bridges := c.GetBridges(tenantID)
		if bridges != nil && len(bridges) > 0 {
			for _, bridge := range bridges {
				ports := c.GetPortByBridge(bridge.ID.String())
				if ports != nil && len(ports) > 0 {
					for _, port := range ports {
						c.DeletePort(port.ID)
						c.DeletePort(port.PeerID)
						c.DeletePortLink(port.ID)
					}
				}
				err := c.DeleteBridges(bridge.ID)
				if err != nil {
					logrus.Errorf("Delete bridge (%s) by tenant (%s) error.%s", bridge.ID, tenantID, err.Error())
				}
			}
		}
		wait.Done()
	}()
	//删除chain相关
	go func() {
		chains := c.GetChain(tenantID)
		if chains != nil && len(chains) > 0 {
			for _, chain := range chains {
				rules := c.GetRuleByChain(chain.ID.String())
				if rules != nil && len(rules) > 0 {
					for _, rule := range rules {
						c.DeleteRule(rule.ID)
					}
				}
				err := c.DeleteChain(chain.ID)
				if err != nil {
					logrus.Errorf("Delete chain (%s) by tenant (%s) error.%s", chain.ID, tenantID, err.Error())
				}
			}

		}
		wait.Done()
	}()
	wait.Wait()
	logrus.Info("Delete tenant all info success")
	return nil
}
