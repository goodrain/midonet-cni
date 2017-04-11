package midonet

import (
	"fmt"
	"net/http"

	"encoding/json"

	"github.com/Sirupsen/logrus"
	"github.com/barnettzqg/golang-midonetclient/types"
)

//GetRouters 获取租户全部router
func (c *Client) GetRouters(tenantID string) []*types.Router {
	if tenantID == "" {
		return nil
	}
	request, err := http.NewRequest("GET", c.apiConf.URL+fmt.Sprintf("/routers?tenant_id=%s", tenantID), nil)
	if err != nil {
		logrus.Errorln("midonet client get routers request error.", err.Error())
		return nil
	}
	c.setHeader(request.Header, "router")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("get all router by tenant error.", err.Error())
		return nil
	}
	var routers []*types.Router
	if res.StatusCode/100 == 2 {
		defer res.Body.Close()
		err := json.NewDecoder(res.Body).Decode(&routers)
		if err != nil {
			logrus.Error("Get all router by tenant error.", err.Error())
		} else {
			return routers
		}
	}
	return nil
}

//GetBridges 获取租户全部bridge
func (c *Client) GetBridges(tenantID string) []*types.Bridge {
	if tenantID == "" {
		return nil
	}
	request, err := http.NewRequest("GET", c.apiConf.URL+fmt.Sprintf("/bridges?tenant_id=%s", tenantID), nil)
	if err != nil {
		logrus.Errorln("midonet client get bridges request error.", err.Error())
		return nil
	}
	c.setHeader(request.Header, "bridge")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("get all bridge by tenant error.", err.Error())
		return nil
	}
	var bridges []*types.Bridge
	if res.StatusCode/100 == 2 {
		defer res.Body.Close()
		err := json.NewDecoder(res.Body).Decode(&bridges)
		if err != nil {
			logrus.Error("Get all bridge by tenant error.", err.Error())
		} else {
			return bridges
		}
	}
	return nil
}

//GetChain 获取租户全部chain
func (c *Client) GetChain(tenantID string) []*types.Chain {
	if tenantID == "" {
		return nil
	}
	request, err := http.NewRequest("GET", c.apiConf.URL+fmt.Sprintf("/chains?tenant_id=%s", tenantID), nil)
	if err != nil {
		logrus.Errorln("midonet client get chains request error.", err.Error())
		return nil
	}
	c.setHeader(request.Header, "chain")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("get all chain by tenant error.", err.Error())
		return nil
	}
	var chains []*types.Chain
	if res.StatusCode/100 == 2 {
		defer res.Body.Close()
		err := json.NewDecoder(res.Body).Decode(&chains)
		if err != nil {
			logrus.Error("Get all chains by tenant error.", err.Error())
		} else {
			return chains
		}
	}
	return nil
}

//GetPortByRouter 获取router的关联port
func (c *Client) GetPortByRouter(routerID string) []*types.RouterPort {
	if routerID == "" {
		return nil
	}
	request, err := http.NewRequest("GET", c.apiConf.URL+fmt.Sprintf("/routers/%s/ports", routerID), nil)
	if err != nil {
		logrus.Errorln("midonet client get ports request error.", err.Error())
		return nil
	}
	c.setHeader(request.Header, "port")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("get all ports by tenant error.", err.Error())
		return nil
	}
	var ports []*types.RouterPort
	if res.StatusCode/100 == 2 {
		defer res.Body.Close()
		err := json.NewDecoder(res.Body).Decode(&ports)
		if err != nil {
			logrus.Error("Get all chains by tenant error.", err.Error())
		} else {
			return ports
		}
	}
	return nil
}

//GetPeerPortByRouter 获取router的peerport
func (c *Client) GetPeerPortByRouter(routerID string) []*types.RouterPort {
	if routerID == "" {
		return nil
	}
	request, err := http.NewRequest("GET", c.apiConf.URL+fmt.Sprintf("/routers/%s/peer_ports", routerID), nil)
	if err != nil {
		logrus.Errorln("midonet client get ports request error.", err.Error())
		return nil
	}
	c.setHeader(request.Header, "port")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("get all ports by tenant error.", err.Error())
		return nil
	}
	var ports []*types.RouterPort
	if res.StatusCode/100 == 2 {
		defer res.Body.Close()
		err := json.NewDecoder(res.Body).Decode(&ports)
		if err != nil {
			logrus.Error("Get all chains by tenant error.", err.Error())
		} else {
			return ports
		}
	}
	return nil
}

//GetPortByBridge 获取router的关联port
func (c *Client) GetPortByBridge(routerID string) []*types.BridgePort {
	if routerID == "" {
		return nil
	}
	request, err := http.NewRequest("GET", c.apiConf.URL+fmt.Sprintf("/bridges/%s/ports", routerID), nil)
	if err != nil {
		logrus.Errorln("midonet client get ports request error.", err.Error())
		return nil
	}
	c.setHeader(request.Header, "port")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("get all ports by tenant error.", err.Error())
		return nil
	}
	var ports []*types.BridgePort
	if res.StatusCode/100 == 2 {
		defer res.Body.Close()
		err := json.NewDecoder(res.Body).Decode(&ports)
		if err != nil {
			logrus.Error("Get all chains by tenant error.", err.Error())
		} else {
			return ports
		}
	}
	return nil
}

//GetRoutes 获取router的关联route
func (c *Client) GetRoutes(routerID string) []*types.Route {
	if routerID == "" {
		return nil
	}
	request, err := http.NewRequest("GET", c.apiConf.URL+fmt.Sprintf("/routers/%s/routes", routerID), nil)
	if err != nil {
		logrus.Errorln("midonet client get routes request error.", err.Error())
		return nil
	}
	c.setHeader(request.Header, "route")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("get all routes by router error.", err.Error())
		return nil
	}
	var routes []*types.Route
	if res.StatusCode/100 == 2 {
		defer res.Body.Close()
		err := json.NewDecoder(res.Body).Decode(&routes)
		if err != nil {
			logrus.Error("Get all routes by router error.", err.Error())
		} else {
			return routes
		}
	}
	return nil
}

//GetRuleByChain 获取规则通过chain
func (c *Client) GetRuleByChain(chainID string) []*types.Rule {
	if chainID == "" {
		return nil
	}
	request, err := http.NewRequest("GET", c.apiConf.URL+fmt.Sprintf("/chains/%s/rules", chainID), nil)
	if err != nil {
		logrus.Errorln("midonet client get rules request error.", err.Error())
		return nil
	}
	c.setHeader(request.Header, "rule")
	res, err := c.getHTTPClient().Do(request)
	if err != nil {
		logrus.Error("get all rules by chain error.", err.Error())
		return nil
	}
	var rules []*types.Rule
	if res.StatusCode/100 == 2 {
		defer res.Body.Close()
		err := json.NewDecoder(res.Body).Decode(&rules)
		if err != nil {
			logrus.Error("Get all rules by chain error.", err.Error())
		} else {
			return rules
		}
	}
	return nil
}

//GetRouterIPsByTenant 获取租户router使用的ip
func (c *Client) GetRouterIPsByTenant(tenantID string) []string {
	var address []string
	routers := c.GetRouters(tenantID)
	if routers != nil && len(routers) > 0 {
		for _, router := range routers {
			ports := c.GetPortByRouter(router.ID.String())
			peerPorts := c.GetPeerPortByRouter(router.ID.String())
			if ports != nil && peerPorts != nil {
				allPort := append(ports, peerPorts...)
				if len(allPort) > 0 {
					for _, port := range allPort {
						if port.Type == "InteriorRouter" {
							address = append(address, fmt.Sprintf("%s/%d", port.PortAddress, port.NetworkLength))
						}
					}
				}
			}

		}
	}
	return address
}
