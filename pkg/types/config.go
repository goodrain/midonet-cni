package types

import (
	"os"

	"path"

	"github.com/Sirupsen/logrus"
	"github.com/barnettzqg/golang-midonetclient/types"
	"github.com/barnettzqg/midonet-cni/pkg/util"
)

// Policy is a struct to hold policy config (which currently happens to also contain some K8s config)
type Policy struct {
	PolicyType              string `json:"type"`
	K8sAPIRoot              string `json:"k8s_api_root"`
	K8sAuthToken            string `json:"k8s_auth_token"`
	K8sClientCertificate    string `json:"k8s_client_certificate"`
	K8sClientKey            string `json:"k8s_client_key"`
	K8sCertificateAuthority string `json:"k8s_certificate_authority"`
}

// Kubernetes a K8s specific struct to hold config
type Kubernetes struct {
	K8sAPIRoot string `json:"k8s_api_root"`
	Kubeconfig string `json:"kubeconfig"`
	NodeName   string `json:"node_name"`
}

// Options cni配置
type Options struct {
	Name              string               `json:"name"`
	Type              string               `json:"type"`
	LogLevel          string               `json:"log_level"`
	LogPath           string               `json:"log_path"`
	MidoNetHostUUID   string               `json:"midonet_host_uuid"`
	MidoNetRouterCIDR string               `json:"midonet_router_cidr"`
	MidoNetBridgeCIDR string               `json:"midonet_bridge_cidr"`
	IPAM              IPAM                 `json:"ipam"`
	MTU               int                  `json:"mtu"`
	Policy            Policy               `json:"policy"`
	Kubernetes        Kubernetes           `json:"kubernetes"`
	VethCtrlType      string               `json:"veth_ctrl_type"`
	MidoNetAPIConf    types.MidoNetAPIConf `json:"midonet_api"`
	CNIType           string               `json:"cni_type"`
	ETCDConf          ETCDConf             `json:"etcd_conf"`
	Log               *logrus.Entry
}

//IPAM ip管理器配置
type IPAM struct {
	Type        string `json:"type"`
	IPV4        bool   `json:"ipv4"`
	IPV6        bool   `json:"ipv6"`
	ReginNetAPI string `json:"region_net_api"`
	ReginToken  string `json:"region_token"`
	Route       *Route `json:"route"`
}

//Route 路由规则
type Route struct {
	Net     string `json:"net"`
	NetMask string `json:"netmask"`
	GW      string `json:"gw"`
}

// SetLog 设置日志
func (c *Options) SetLog() error {
	if c.LogLevel != "" {
		if logLevel, err := logrus.ParseLevel(c.LogLevel); err != nil {
			logrus.Error("Unknown log level. Using default: INFO")
		} else {
			logrus.SetLevel(logLevel)
		}
	}
	if c.LogPath == "" {
		c.LogPath = "/var/log/midonet-cni"
	}
	_, err := os.Stat(c.LogPath)
	if os.IsNotExist(err) {
		os.Mkdir(c.LogPath, os.ModeDir)
	}
	logFile, err := os.OpenFile(path.Join(c.LogPath, "midonet-cni.log"), os.O_WRONLY|os.O_APPEND|os.O_CREATE, os.ModePerm)
	if err != nil {
		logrus.Warning("Open log file error. so log will be writed in stderr")
		logrus.SetOutput(os.Stderr)
	} else {
		logrus.SetOutput(logFile)
	}
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	return nil
}

//Default 赋值
func (c *Options) Default() {
	if c.MidoNetRouterCIDR == "" {
		c.MidoNetRouterCIDR = "172.16.0.0/24"
	} else {
		size := util.RangeLength(c.MidoNetRouterCIDR)
		if size < 5 {
			logrus.Error("Invalid MidoNetRouterCIDR ,will set it is 172.16.0.0/24")
			c.MidoNetRouterCIDR = "172.16.0.0/24"
		}
	}
	if c.MidoNetBridgeCIDR == "" {
		c.MidoNetBridgeCIDR = "192.168.0.0/24"
	} else {
		size := util.RangeLength(c.MidoNetBridgeCIDR)
		if size < 4 {
			logrus.Error("Invalid MidoNetBridgeCIDR ,will set it is 192.168.0.0/24")
			c.MidoNetBridgeCIDR = "192.168.0.0/24"
		}
	}
	if c.MTU == 0 {
		c.MTU = 1500
	}

}

// ETCDConf etcd配置
type ETCDConf struct {
	URLs        []string `json:"urls"`
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	PeerTimeOut string   `json:"timeout"`
}
