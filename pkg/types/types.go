package types

import (
	"encoding"
	"fmt"
	"net"
	"reflect"
	"strings"

	"github.com/containernetworking/cni/pkg/types"
)

// K8sArgs is the valid CNI_ARGS used for Kubernetes
type K8sArgs struct {
	types.CommonArgs
	IP                         net.IP
	K8S_POD_NAME               types.UnmarshallableString
	K8S_POD_NAMESPACE          types.UnmarshallableString
	K8S_POD_INFRA_CONTAINER_ID types.UnmarshallableString
}

// GetKeyField is a helper function to receive Values
// Values that represent a pointer to a struct
func GetKeyField(keyString string, v reflect.Value) reflect.Value {
	return v.Elem().FieldByName(keyString)
}

// LoadArgs parses args from a string in the form "K=V;K2=V2;..."
func LoadArgs(args string, container interface{}) error {
	if args == "" {
		return nil
	}

	containerValue := reflect.ValueOf(container)

	pairs := strings.Split(args, ";")
	unknownArgs := []string{}
	for _, pair := range pairs {
		kv := strings.Split(pair, "=")
		if len(kv) != 2 {
			return fmt.Errorf("ARGS: invalid pair %q", pair)
		}
		keyString := kv[0]
		valueString := kv[1]
		keyField := GetKeyField(keyString, containerValue)
		if !keyField.IsValid() {
			unknownArgs = append(unknownArgs, pair)
			continue
		}

		u := keyField.Addr().Interface().(encoding.TextUnmarshaler)
		err := u.UnmarshalText([]byte(valueString))
		if err != nil {
			return fmt.Errorf("ARGS: error parsing value of pair %q: %v)", pair, err)
		}
	}

	isIgnoreUnknown := GetKeyField("IgnoreUnknown", containerValue).Bool()
	if len(unknownArgs) > 0 && !isIgnoreUnknown {
		return fmt.Errorf("ARGS: unknown args %q", unknownArgs)
	}
	return nil
}

//ReginNewIP 从region申请ip信息
type ReginNewIP struct {
	HostID        string `json:"host_id"`
	CtnID         string `json:"ctn_id"`
	ReplicaID     string `json:"replica_id"`
	DeployVersion string `json:"deploy_version"`
	PodName       string `json:"pod_name"`
}

//IPInfo ipam返回信息
type IPInfo struct {
	ContainerID string    `json:"containerID"`
	IPNet       net.IPNet `json:"ipnet"`
	Getway      net.IP    `json:"getway"`
}
