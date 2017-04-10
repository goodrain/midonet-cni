package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"

	"github.com/Sirupsen/logrus"
	"github.com/containernetworking/cni/pkg/skel"
	cni "github.com/containernetworking/cni/pkg/types"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/goodrain/midonet-cni/pkg/k8s"
	"github.com/goodrain/midonet-cni/pkg/types"
)

var hostname string
var kubernetes = "kubernetes"

func init() {
	// This ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()

	hostname, _ = os.Hostname()
}

func cmdAdd(args *skel.CmdArgs) error {
	options := &types.Options{}
	if err := json.Unmarshal(args.StdinData, options); err != nil {
		return fmt.Errorf("Failed to parse config: %v", err)
	}
	if err := options.Default(); err != nil {
		return err
	}
	options.SetLog()
	log := logrus.WithFields(logrus.Fields{"container_id": args.ContainerID})
	log.Debug("Configuring pod networking.")
	if options.CNIType == kubernetes {
		result, err := k8s.CmdAddK8s(args, options, hostname)
		if err != nil {
			if err := os.Setenv("CNI_COMMAND", "DEL"); err != nil {
				// Failed to set CNI_COMMAND to DEL.
				log.Warning("Failed to set CNI_COMMAND=DEL")
			} else {
				if err := cmdDel(args); err != nil {
					// Failed to cmdDel for failed ADD
					log.Warning("Failed to cmdDel for failed ADD")
				}
			}
			return err
		}
		log.Debug("Configuring pod network success")
		return result.Print()
	}
	return fmt.Errorf("midonet-cni not support your type:%s", options.Type)
}

func cmdDel(args *skel.CmdArgs) error {
	options := &types.Options{}
	if err := json.Unmarshal(args.StdinData, options); err != nil {
		return fmt.Errorf("Failed to parse config: %v", err)
	}
	options.Default()
	options.SetLog()
	log := logrus.WithFields(logrus.Fields{"container_id": args.ContainerID})
	log.Debug("Del Configuring pod networking.")
	if options.CNIType == kubernetes {
		result, err := k8s.CmdDelK8s(args, options, hostname)
		if err != nil {
			return err
		}
		return result.Print()
	}
	return fmt.Errorf("midonet-cni not support your type:%s", options.Type)
}

// VERSION is filled out during the build process (using git describe output)
var VERSION string

func main() {
	// Display the version on "-v", otherwise just delegate to the skel code.
	// Use a new flag set so as not to conflict with existing libraries which use "flag"
	flagSet := flag.NewFlagSet("Midonet", flag.ExitOnError)

	v := flagSet.Bool("v", false, "Display version")
	err := flagSet.Parse(os.Args[1:])
	if err != nil {
		(&cni.Error{
			Code: 100,
			Msg:  err.Error(),
		}).Print()
		os.Exit(1)
	}
	if *v {
		fmt.Println(VERSION)
		os.Exit(0)
	}
	skel.PluginMain(cmdAdd, cmdDel, version.Legacy)
}
