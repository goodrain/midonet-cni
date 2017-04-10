package types

import (
	"errors"
	"os"
	"time"

	"golang.org/x/net/context"

	"path"

	"io/ioutil"

	"strings"

	"encoding/json"

	"github.com/Sirupsen/logrus"
	"github.com/barnettzqg/golang-midonetclient/types"
	"github.com/coreos/etcd/client"
	"github.com/goodrain/midonet-cni/pkg/util"
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
	Type        string   `json:"type"`
	IPV4        bool     `json:"ipv4"`
	IPV6        bool     `json:"ipv6"`
	ReginNetAPI string   `json:"region_net_api"`
	ReginToken  string   `json:"region_token"`
	Route       []*Route `json:"route"`
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
func (c *Options) Default() error {
	if len(c.ETCDConf.URLs) < 1 {
		return errors.New("Please config etcd ")
	}
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
		c.MTU = 1454
	}
	if c.MidoNetHostUUID == "" {
		_, err := os.Stat("/etc/midolman/host_uuid.properties")
		var data []byte
		if err == nil {
			data, err = ioutil.ReadFile("/etc/midolman/host_uuid.properties")
			if err != nil {
				logrus.Error("Read /etc/midolman/host_uuid.properties file error.", err.Error())
			} else {
				goto parse
			}
		}
		_, err = os.Stat("/etc/midonet_host_id.properties")
		if err != nil {
			logrus.Error("Don't find /etc/host_uuid.properties file .", err.Error())
			return err
		}
		data, err = ioutil.ReadFile("/etc/midonet_host_id.properties")
		if err != nil {
			logrus.Error("Read /etc/host_uuid.properties file error.", err.Error())
			return err
		}
	parse:
		datas := strings.Split(string(data), "=")
		if len(datas) == 2 {
			c.MidoNetHostUUID = datas[1]
		} else {
			logrus.Error("Parse the midonet host uuid error.")
			return err
		}
	}
	//get midonet api config from etcd
	if c.MidoNetAPIConf.URL == "" {
		etcdClient, err := createETCDClient(c.ETCDConf)
		if err != nil {
			return err
		}
		response, err := client.NewKeysAPI(etcdClient).Get(context.Background(), "/midonet-cni/config/midonet-api", &client.GetOptions{})
		if err != nil {
			return errors.New("Find midonet api config from etcd error." + err.Error())
		}
		value := response.Node.Value
		var con types.MidoNetAPIConf
		err = json.Unmarshal([]byte(value), &con)
		if err != nil {
			return errors.New("Find midonet api config from etcd error." + err.Error())
		}
		c.MidoNetAPIConf = con
	}
	if c.Kubernetes.K8sAPIRoot == "" {
		etcdClient, err := createETCDClient(c.ETCDConf)
		if err != nil {
			return err
		}
		response, err := client.NewKeysAPI(etcdClient).Get(context.Background(), "/midonet-cni/config/kubernetes", &client.GetOptions{})
		if err != nil {
			return errors.New("Find kubernetes api config from etcd error." + err.Error())
		}
		value := response.Node.Value
		var kube Kubernetes
		err = json.Unmarshal([]byte(value), &kube)
		if err != nil {
			return errors.New("Find kubernetes api config from etcd error." + err.Error())
		}
		c.Kubernetes = kube
	}
	//如果etcd中有定义route，获取它
	if c.IPAM.Route == nil || len(c.IPAM.Route) == 0 {
		etcdClient, err := createETCDClient(c.ETCDConf)
		if err != nil {
			return err
		}
		response, err := client.NewKeysAPI(etcdClient).Get(context.Background(), "/midonet-cni/config/route", &client.GetOptions{})
		if err != nil {
			return errors.New("Find midonet api config from etcd error." + err.Error())
		}
		value := response.Node.Value
		var routes []*Route
		err = json.Unmarshal([]byte(value), &routes)
		if err != nil {
			return errors.New("Find midonet api config from etcd error." + err.Error())
		}
		c.IPAM.Route = routes
	}
	return nil
}

// ETCDConf etcd配置
type ETCDConf struct {
	URLs        []string `json:"urls"`
	Username    string   `json:"username"`
	Password    string   `json:"password"`
	PeerTimeOut string   `json:"timeout"`
}

//createETCDClient 创建etcd客户端
func createETCDClient(conf ETCDConf) (client.Client, error) {
	var timeout time.Duration
	if conf.PeerTimeOut != "" {
		var err error
		timeout, err = time.ParseDuration(conf.PeerTimeOut)
		if err != nil {
			timeout = time.Second
		}
	}

	cfg := client.Config{
		Endpoints: conf.URLs,
		Transport: client.DefaultTransport,
		// set timeout per request to fail fast when the target endpoint is unavailable
		HeaderTimeoutPerRequest: timeout,
		Username:                conf.Username,
		Password:                conf.Password,
	}
	c, err := client.New(cfg)
	if err != nil {
		logrus.Error("Create etcd client error,", err.Error())
		return nil, err
	}
	return c, nil
}
