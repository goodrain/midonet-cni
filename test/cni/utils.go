package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"

	config "github.com/barnettzqg/midonet-cni/pkg/types"
	"github.com/containernetworking/cni/pkg/types"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega/gexec"
)

// RunCNIPlugin sets ENV vars required then calls the CNI plugin
// specified in the config and returns the result and exitCode.
func RunCNIPlugin(netconf, command, args string) (types.Result, types.Error, int) {
	conf := config.Options{}
	if err := json.Unmarshal([]byte(netconf), &conf); err != nil {
		panic(fmt.Errorf("failed to load netconf: %v", err))
	}

	// Run the CNI plugin passing in the supplied netconf
	cmd := &exec.Cmd{
		Env:  []string{"CNI_COMMAND=" + command, "CNI_CONTAINERID=a", "CNI_NETNS=b", "CNI_IFNAME=c", "CNI_PATH=d", "CNI_ARGS=" + args},
		Path: "../midonet-cni",
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic("some error found," + err.Error())
	}

	io.WriteString(stdin, netconf)
	io.WriteString(stdin, "\n")
	stdin.Close()

	session, err := gexec.Start(cmd, ginkgo.GinkgoWriter, ginkgo.GinkgoWriter)
	if err != nil {
		panic("some error found," + err.Error())
	}
	session.Wait(5)
	exitCode := session.ExitCode()
	result := types.Result{}
	error := types.Error{}
	stdout := session.Out.Contents()
	if exitCode == 0 {
		if command == "ADD" {
			if err := json.Unmarshal(stdout, &result); err != nil {
				panic(fmt.Errorf("failed to load result: %s %v", stdout, err))
			}
		}
	} else {
		if err := json.Unmarshal(stdout, &error); err != nil {
			panic(fmt.Errorf("failed to load error: %s %v", stdout, err))
		}
	}

	return result, error, exitCode
}
