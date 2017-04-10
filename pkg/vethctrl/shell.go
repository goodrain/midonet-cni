package vethctrl

import (
	"os/exec"

	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	conf "github.com/goodrain/midonet-cni/pkg/types"
)

//ShellCtrl shell实现
type ShellCtrl struct{}

//DoNetworking 创建veth并绑定ip
func (s *ShellCtrl) DoNetworking(args *skel.CmdArgs, conf *conf.Options, result *types.Result) error {
	var err error
	err = s.ExecCreateBr(args.ContainerID)
	if err != nil {
		return err
	}
	err = s.ExecBindingIP(fmt.Sprintf("%s@%s", result.IP4.IP.String(), result.IP4.Gateway.String()), args.ContainerID)
	if err != nil {
		return err
	}
	return nil
}

// ExecCreateBr create local bridge
func (s *ShellCtrl) ExecCreateBr(containerID string) error {

	var cmd = "/usr/bin/vethctl create " + containerID
	logrus.Info("ExecCreateBr=== %s", cmd)
	_, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		logrus.Error("ExecCreateBr is failure")
		_, err = exec.Command("sh", "-c", cmd).Output()
		if err != nil {
			logrus.Error("ExecCreateBr is failure two,return false")
			return err
		}
	}
	logrus.Info("ExecCreateBr is success")
	return nil
}

// ExecBindingIP binding local container ip
func (s *ShellCtrl) ExecBindingIP(ip, containerID string) error {

	var cmd = "/usr/bin/vethctl setnetns " + ip + " " + containerID
	logrus.Info("ExecBindingIp=== %s", cmd)
	_, err := exec.Command("sh", "-c", cmd).Output()
	if err != nil {
		logrus.Errorf("ExecBindingIp is failure")
		_, err := exec.Command("sh", "-c", cmd).Output()
		if err != nil {
			logrus.Error("ExecBindingIP is failure two,return false")
			return err
		}
	}
	logrus.Info("ExecBindingIp is success")
	return nil
}
