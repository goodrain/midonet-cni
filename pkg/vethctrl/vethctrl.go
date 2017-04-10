package vethctrl

import (
	"fmt"
	"net"
	"os"

	"github.com/Sirupsen/logrus"
	"github.com/containernetworking/cni/pkg/ip"
	"github.com/containernetworking/cni/pkg/ns"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	conf "github.com/goodrain/midonet-cni/pkg/types"
	"github.com/vishvananda/netlink"
)

//VethCtrl veth网桥操作接口
type VethCtrl interface {
	DoNetworking(args *skel.CmdArgs, conf *conf.Options, result *types.Result) error
}

//GetVethCtrl 获取veth操作器
func GetVethCtrl(key string, log *logrus.Entry) VethCtrl {
	switch key {
	case "shell":
		return &ShellCtrl{}
	default:
		log.Debug("Use inner veth ctrl")
		return &InnerVethCtrl{
			log: log,
		}
	}

}

//InnerVethCtrl 网桥管理器
type InnerVethCtrl struct {
	log *logrus.Entry
}

//DoNetworking 创建veth对,绑定IP
func (v *InnerVethCtrl) DoNetworking(args *skel.CmdArgs, conf *conf.Options, result *types.Result) error {
	var err error
	// Select the first 11 characters of the containerID for the host veth.
	hostVethName := "vif" + args.ContainerID[:min(12, len(args.ContainerID))]
	contVethName := "eth0"
	v.log.Debugf("Set veth %s , %s", hostVethName, contVethName)
	err = ns.WithNetNSPath(args.Netns, func(hostNS ns.NetNS) error {
		//判断eth0网卡是否存在，如果已存在将其修改为eth1
		oldEth0, err := netlink.LinkByName(contVethName)
		if err == nil && oldEth0 != nil {
			v.log.Debug(contVethName + " interface exist,will rename to eth1")
			err = netlink.LinkSetDown(oldEth0)
			if err != nil {
				v.log.Error("old interface eth0 set down error.", err.Error())
				return err
			}
			err = netlink.LinkSetName(oldEth0, "eth1")
			if err != nil {
				v.log.Error("old interface eth0 rename eth1 error.", err.Error())
				return err
			}
			newEth1, err := netlink.LinkByName("eth1")
			if err != nil {
				v.log.Errorf("failed to lookup %q: %v", "eth1", err)
				return err
			}
			err = netlink.LinkSetUp(newEth1)
			if err != nil {
				v.log.Errorf("failed to set up %q: %v", "eth1", err)
				return err
			}
		}
		//创建新的eth0
		veth := &netlink.Veth{
			LinkAttrs: netlink.LinkAttrs{
				Name:  contVethName,
				Flags: net.FlagUp,
				MTU:   conf.MTU,
			},
			PeerName: hostVethName,
		}
		if err := netlink.LinkAdd(veth); err != nil {
			v.log.Errorf("Error adding veth %+v: %s", veth, err)
			return err
		}

		hostVeth, err := netlink.LinkByName(hostVethName)
		if err != nil {
			err = fmt.Errorf("failed to lookup %q: %v", hostVethName, err)
			return err
		}
		// Explicitly set the veth to UP state, because netlink doesn't always do that on all the platforms with net.FlagUp.
		// veth won't get a link local address unless it's set to UP state.
		if err = netlink.LinkSetUp(hostVeth); err != nil {
			return fmt.Errorf("failed to set %q up: %v", hostVethName, err)
		}

		contVeth, err := netlink.LinkByName(contVethName)
		if err != nil {
			err = fmt.Errorf("failed to lookup %q: %v", contVethName, err)
			return err
		}
		// Before returning, create the routes inside the namespace, first for IPv4 then IPv6.
		if result.IP4 != nil {

			if err = netlink.AddrAdd(contVeth, &netlink.Addr{IPNet: &result.IP4.IP}); err != nil {
				return fmt.Errorf("failed to add IP addr to %q: %v", contVethName, err)
			}

			if conf.IPAM.Route != nil && conf.IPAM.Route.Net != "" && conf.IPAM.Route.NetMask != "" && conf.IPAM.Route.GW != "" {
				// 添加一个路由规则，从DstNet来的包从gw出去
				DstNet := net.ParseIP(conf.IPAM.Route.Net)
				mask := net.ParseIP(conf.IPAM.Route.NetMask)
				DstMask := net.IPv4Mask(mask[12], mask[13], mask[14], mask[15])
				gw := net.ParseIP(conf.IPAM.Route.GW)
				gwNet := &net.IPNet{IP: DstNet, Mask: DstMask}
				router := &netlink.Route{
					Dst: gwNet,
					Gw:  gw,
				}
				v.log.Debug("Add a route ", router)
				if err = netlink.RouteAdd(router); err != nil {
					return fmt.Errorf("failed to add route %v", err)
				}
			}
			_, defNet, _ := net.ParseCIDR("0.0.0.0/0")
			if err = netlink.RouteDel(&netlink.Route{
				Dst: defNet,
			}); err != nil {
				v.log.Warn("Delete default route error,", err.Error())
			}
			if err = ip.AddDefaultRoute(result.IP4.Gateway, contVeth); err != nil {
				return fmt.Errorf("failed to add default route %v", err)
			}

		}
		// Now that the everything has been successfully set up in the container, move the "host" end of the
		// veth into the host namespace.
		if err = netlink.LinkSetNsFd(hostVeth, int(hostNS.Fd())); err != nil {
			return fmt.Errorf("failed to move veth to host netns: %v", err)
		}
		return nil
	})
	if err != nil {
		v.log.Errorf("Error creating veth: %s", err)
		return err
	}

	// Moving a veth between namespaces always leaves it in the "DOWN" state. Set it back to "UP" now that we're
	// back in the host namespace.
	hostVeth, err := netlink.LinkByName(hostVethName)
	if err != nil {
		return fmt.Errorf("failed to lookup %q: %v", hostVethName, err)
	}

	if err = netlink.LinkSetUp(hostVeth); err != nil {
		return fmt.Errorf("failed to set %q up: %v", hostVethName, err)
	}

	// 做一个软连接便于调试
	err = os.Symlink(args.Netns, "/var/run/netns/"+args.ContainerID[:min(12, len(args.ContainerID))])
	if err != nil {
		v.log.Warnf("create link error. source file:%s,target file:%s", args.Netns, "/var/run/netns/"+args.ContainerID[:min(12, len(args.ContainerID))])
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
