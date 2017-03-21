package main

import (
	"bytes"
	"os"
	"os/exec"

	"github.com/Sirupsen/logrus"
)

func main() {
	conf := `
	  {
        "name": "midonet-cni", 
        "type": "kubernetes", 
        "log_level": "debug", 
        "midonet_host_uuid": "", 
        "ipam": {
            "region_net_api": "", 
            "region_token": "", 
            "type": "reginapi"
        }, 
        "kubernetes": {
            "k8s_api_root": "http://127.0.0.1:8080", 
            "kubeconfig": "", 
            "node_name": ""
        }, 
        "policy": {
            "type": "k8s", 
            "k8s_api_root": "", 
            "k8s_auth_token": "", 
            "k8s_client_certificate": "", 
            "k8s_client_key": "", 
            "k8s_certificate_authority": ""
        }
     }

	`
	logrus.Info(conf)
	// RunCNIPlugin(conf, "ADD", "")
	cmd := &exec.Cmd{
		Env:  []string{"CNI_COMMAND=ADD", "CNI_CONTAINERID=furious_banach", "CNI_NETNS=b", "CNI_IFNAME=eth0", "CNI_PATH=d", "CNI_ARGS="},
		Path: "../midonet-cni",
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = bytes.NewReader([]byte(conf + " \n"))
	err := cmd.Start()
	logrus.Error(err.Error())
}
