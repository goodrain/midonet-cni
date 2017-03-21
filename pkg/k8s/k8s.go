package k8s

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"time"

	"k8s.io/client-go/1.4/kubernetes"
	"k8s.io/client-go/1.4/tools/clientcmd"

	"github.com/Sirupsen/logrus"
	midonetclient "github.com/barnettzqg/golang-midonetclient/midonet"
	midonettypes "github.com/barnettzqg/golang-midonetclient/types"
	"github.com/barnettzqg/midonet-cni/pkg/etcd"
	"github.com/barnettzqg/midonet-cni/pkg/ipam"
	"github.com/barnettzqg/midonet-cni/pkg/midonet"
	conf "github.com/barnettzqg/midonet-cni/pkg/types"
	"github.com/barnettzqg/midonet-cni/pkg/vethctrl"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/coreos/etcd/client"
)

// CmdAddK8s k8s cni
// 增强的幂等特性
func CmdAddK8s(args *skel.CmdArgs, options *conf.Options, hostname string) (result *types.Result, err error) {
	k8sArgs := conf.K8sArgs{}
	err = types.LoadArgs(args.Args, &k8sArgs)
	if err != nil {
		return result, err
	}
	log := logrus.WithFields(logrus.Fields{"pod_name": k8sArgs.K8S_POD_NAME, "namespace": k8sArgs.K8S_POD_NAMESPACE, "action": "Add"})
	options.Log = log
	netContainerID := string(k8sArgs.K8S_POD_INFRA_CONTAINER_ID)
	c, err := etcd.CreateETCDClient(options.ETCDConf)
	if err != nil {
		log.Error("create etcd client error.", err.Error())
		return nil, err
	}
	kapi := client.NewKeysAPI(c)
	res, err := kapi.Get(context.Background(), "/midonet-cni/result/"+netContainerID, nil)
	if err == nil {
		var r types.Result
		err := json.Unmarshal([]byte(res.Node.Value), &r)
		if err == nil {
			log.Debugf("container(%s) result is exit.return old data.", netContainerID)
			return &r, nil
		}
	}
	var bindingIP net.IPNet
	var bindingGetway net.IP
	var ipnet *net.IPNet
	if options.IPAM.Type == "reginapi" {
		client, err := newK8sClient(options)
		if err != nil {
			return result, err
		}
		labels := make(map[string]string)
		annot := make(map[string]string)
		logrus.WithField("client", client).Debug("Created Kubernetes client")
		labels, annot, err = getK8sLabelsAnnotations(client, k8sArgs)
		if err != nil {
			return result, err
		}
		logrus.WithField("labels", labels).Debug("Fetched K8s labels")
		logrus.WithField("annotations", annot).Debug("Fetched K8s annotations")
		region := &ipam.RegionAPI{
			ReginNetAPI: options.IPAM.ReginNetAPI,
			Token:       options.IPAM.ReginToken,
			HTTPTimeOut: time.Minute * 1,
		}
		var version = ""
		if v, ok := labels["version"]; ok {
			version = v
		}
		info := conf.ReginNewIP{
			HostID:        options.MidoNetHostUUID,
			CtnID:         args.ContainerID,
			ReplicaID:     strings.Split(string(k8sArgs.K8S_POD_NAME), "-")[0],
			DeployVersion: version,
			PodName:       string(k8sArgs.K8S_POD_NAME),
		}
		ip, err := region.GetNewIP(info, string(k8sArgs.K8S_POD_NAMESPACE))
		if err != nil {
			return result, err
		}
		ips := strings.Split(ip, "@")
		if ip == "" || len(ips) != 2 {
			log.Errorf("pod_namespace= %s not apply ip rc_id= %s", k8sArgs.K8S_POD_NAMESPACE, strings.Split(string(k8sArgs.K8S_POD_NAME), "-")[0])
			return result, fmt.Errorf("get ip from region api error")
		}
		var nip net.IP
		nip, ipnet, err = net.ParseCIDR(ips[0])
		bindingGetway = net.ParseIP(ips[1])
		if err != nil {
			log.Error("the ip that it from region api get parse error, ", err.Error())
			return result, err
		}
		bindingIP = net.IPNet{
			IP:   nip,
			Mask: ipnet.Mask,
		}
	} else if options.IPAM.Type == "etcd" {
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

		ipinfo, err := ipamManager.GetNewIP(tenant, netContainerID)
		if err != nil {
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
	} else {
		return result, fmt.Errorf("Undefined IPAM")
	}
	result = &types.Result{
		IP4: &types.IPConfig{
			Gateway: bindingGetway,
			IP:      bindingIP,
		},
	}
	//创建veth对
	ctrl := vethctrl.GetVethCtrl(options.VethCtrlType, log)
	err = ctrl.DoNetworking(args, options, result)
	if err != nil {
		log.Error("Create and binding ip error,", err.Error())
		return result, err
	}
	reData, err := json.Marshal(result)
	if err == nil {
		_, err := kapi.Set(context.Background(), "/midonet-cni/result/"+netContainerID, string(reData), nil)
		if err != nil {
			log.Warn("save result error.")
		}
	}
	return result, nil
}

//CmdDelK8s 删除
func CmdDelK8s(args *skel.CmdArgs, options *conf.Options, hostname string) (result *types.Result, err error) {
	if options.IPAM.Type != "etcd" {
		return &types.Result{}, nil
	}
	k8sArgs := conf.K8sArgs{}
	err = types.LoadArgs(args.Args, &k8sArgs)
	if err != nil {
		return result, err
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
	if err != nil {
		log.Error("Delete cni result error.", err.Error())
	}
	res, err = kapi.Get(context.Background(), fmt.Sprintf("/midonet-cni/bingding/%s/%s", tenantID, netContainerID), nil)
	if err == nil {
		bindingInfoStr := res.Node.Value
		bindingInfo := midonettypes.HostInterfacePort{}
		err := json.Unmarshal([]byte(bindingInfoStr), &bindingInfo)
		if err == nil {
			client, err := midonetclient.NewClient(&options.MidoNetAPIConf)
			if err != nil {
				client.DeleteBinding(&bindingInfo)
			}
		}
	}
	res, err = kapi.Delete(context.Background(), fmt.Sprintf("/midonet-cni/bingding/%s/%s", tenantID, netContainerID), nil)
	if err != nil {
		log.Error("Delete bingding status error.", err.Error())
	}

	res, err = kapi.Get(context.Background(), fmt.Sprintf("/midonet-cni/ip/pod/%s/%s", tenantID, netContainerID), nil)
	if err == nil {
		if err == nil {
			i, err := ipam.CreateEtcdIpam(*options)
			if err == nil {
				i.ReleaseIP(tenantID, res.Node.Value)
			}
		}
	}
	res, err = kapi.Delete(context.Background(), fmt.Sprintf("/midonet-cni/ip/pod/%s/%s", tenantID, netContainerID), nil)
	if err != nil {
		log.Error("Delete used ip info error.", err.Error())
	}
	return &types.Result{}, nil
}

func newK8sClient(conf *conf.Options) (*kubernetes.Clientset, error) {
	// Some config can be passed in a kubeconfig file
	kubeconfig := conf.Kubernetes.Kubeconfig

	// Config can be overridden by config passed in explicitly in the network config.
	configOverrides := &clientcmd.ConfigOverrides{}

	// If an API root is given, make sure we're using using the name / port rather than
	// the full URL. Earlier versions of the config required the full `/api/v1/` extension,
	// so split that off to ensure compatibility.
	conf.Policy.K8sAPIRoot = strings.Split(conf.Policy.K8sAPIRoot, "/api/")[0]

	var overridesMap = []struct {
		variable *string
		value    string
	}{
		{&configOverrides.ClusterInfo.Server, conf.Policy.K8sAPIRoot},
		{&configOverrides.AuthInfo.ClientCertificate, conf.Policy.K8sClientCertificate},
		{&configOverrides.AuthInfo.ClientKey, conf.Policy.K8sClientKey},
		{&configOverrides.ClusterInfo.CertificateAuthority, conf.Policy.K8sCertificateAuthority},
		{&configOverrides.AuthInfo.Token, conf.Policy.K8sAuthToken},
	}

	// Using the override map above, populate any non-empty values.
	for _, override := range overridesMap {
		if override.value != "" {
			*override.variable = override.value
		}
	}

	// Also allow the K8sAPIRoot to appear under the "kubernetes" block in the network config.
	if conf.Kubernetes.K8sAPIRoot != "" {
		configOverrides.ClusterInfo.Server = conf.Kubernetes.K8sAPIRoot
	}

	// Use the kubernetes client code to load the kubeconfig file and combine it with the overrides.
	config, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
		configOverrides).ClientConfig()
	if err != nil {
		return nil, err
	}

	logrus.Debugf("Kubernetes config %v", config)

	// Create the clientset
	return kubernetes.NewForConfig(config)

}

func getK8sLabelsAnnotations(client *kubernetes.Clientset, k8sargs conf.K8sArgs) (map[string]string, map[string]string, error) {
	pod, err := client.Pods(fmt.Sprintf("%s", k8sargs.K8S_POD_NAMESPACE)).
		Get(fmt.Sprintf("%s", k8sargs.K8S_POD_NAME))
	if err != nil {
		logrus.Error("Get pods from apiserver error," + err.Error())
		return nil, nil, err
	}

	labels := pod.Labels
	if labels == nil {
		labels = make(map[string]string)
	}

	return labels, pod.Annotations, nil
}
