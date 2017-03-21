package types

import (
	"errors"
	"strings"

	"github.com/twinj/uuid"
)

//MidoNetAPIConf 配置
type MidoNetAPIConf struct {
	URL              string       `json:"url"`
	UserName         string       `json:"user_name"`
	PassWord         string       `json:"password"`
	ProjectID        string       `json:"project_id"`
	ProviderRouterID string       `json:"provider_router_id"`
	Version          int          `json:"version"`
	KeystoneConf     KeystoneConf `json:"keystone_conf"`
}

//KeystoneConf keystone_conf
type KeystoneConf struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

//UUID midonet id
type UUID struct {
	UuID uuid.Uuid
}

//CreateUUID 创建uuid
func CreateUUID() *UUID {
	uid := uuid.NewV4()
	return &UUID{UuID: uid}
}

func (u *UUID) String() string {
	return u.UuID.String()
}

//String2UUID string 转 uuid
func String2UUID(data string) (*UUID, error) {
	u, err := uuid.Parse(data)
	if err != nil {
		return nil, err
	}
	return &UUID{UuID: u}, err
}

//UnmarshalJSON 解析json
func (u *UUID) UnmarshalJSON(b []byte) (err error) {
	s := strings.Trim(string(b), "\"")
	u.UuID, err = uuid.Parse(s)
	if u.UuID == nil {
		return errors.New("Could not parse UUID")
	}
	return err
}

// MarshalJSON 解析成json串
func (u *UUID) MarshalJSON() (data []byte, err error) {
	return []byte("\"" + u.UuID.String() + "\""), nil
}

//Tenant 租户
type Tenant struct {
	Name        string `json:"name"`
	Enabled     bool   `json:"enabled"`
	ID          string `json:"id"`
	Description string `json:"description"`
}

//Bridge 网桥
type Bridge struct {
	Name                 string  `json:"name"`
	TenantID             string  `json:"tenantId"`
	ID                   *UUID   `json:"id,omitempty"`
	MacPortTemplate      string  `json:"macPortTemplate,omitempty"`
	VlanMacPortTemplate  string  `json:"vlanMacPortTemplate,omitempty"`
	VlanMacTableTemplate string  `json:"vlanMacTableTemplate,omitempty"`
	AdminStateUp         bool    `json:"adminStateUp,omitempty"`
	InboundFilterID      *UUID   `json:"inboundFilterId,omitempty"`
	InboundMirrorIDs     []*UUID `json:"inboundMirrorIds,omitempty"`
	OutboundFilterID     *UUID   `json:"outboundFilterId,omitempty"`
	OutboundMirrorIDs    []*UUID `json:"outboundMirrorIds,omitempty"`
	VxlanPortIds         []*UUID `json:"vxlanPortIds,omitempty"`
}

//Chain 链路
type Chain struct {
	ID       *UUID  `json:"id,omitempty"`
	Name     string `json:"name"`
	TenantID string `json:"tenantId"`
}

//Route 路由
type Route struct {
	ID               *UUID  `json:"id,omitempty"`
	DstNetworkAddr   string `json:"dstNetworkAddr"`
	DstNetworkLength int    `json:"dstNetworkLength"`
	NextHopGateway   string `json:"nextHopGateway,omitempty"`
	NextHopPort      *UUID  `json:"nextHopPort"`
	SrcNetworkAddr   string `json:"srcNetworkAddr"`
	SrcNetworkLength int    `json:"srcNetworkLength"`
	Type             string `json:"type"`
	Weight           int    `json:"weight"`
	Learned          bool   `json:"learned,omitempty"`
	RouterID         *UUID  `json:"routerId"`
}

//Router 路由器
type Router struct {
	ID                *UUID   `json:"id,omitempty"`
	AdminStateUp      bool    `json:"adminStateUp,omitempty"`
	InboundFilterID   *UUID   `json:"inboundFilterId,omitempty"`
	InboundMirrorIDs  []*UUID `json:"inboundMirrorIds,omitempty"`
	OutboundFilterID  *UUID   `json:"outboundFilterId,omitempty"`
	OutboundMirrorIDs []*UUID `json:"outboundMirrorIds,omitempty"`
	AsNumber          int     `json:"asNumber,omitempty"`
	LoadBalancerID    *UUID   `json:"loadBalancerId,omitempty"`
	Name              string  `json:"name"`
	TenantID          string  `json:"tenantId"`
}

//Rule 规则
type Rule struct {
	ID                 *UUID       `json:"id,omitempty"`
	Type               string      `json:"type"`
	CondInvert         bool        `json:"condInvert,omitempty"`
	DlDst              string      `json:"dlDst,omitempty"`
	DlSrc              string      `json:"dlSrc,omitempty"`
	DlDstMask          string      `json:"dlDstMask,omitempty"`
	DlSrcMask          string      `json:"dlSrcMask,omitempty"`
	DlType             int         `json:"dlType,omitempty"`
	FlowAction         string      `json:"flowAction,omitempty"`
	FragmentPolicy     string      `json:"fragmentPolicy,omitempty"`
	InPorts            []*UUID     `json:"inPorts,omitempty"`
	IPAddrGroupDst     *UUID       `json:"ipAddrGroupDst,omitempty"`
	IPAddrGroupSrc     *UUID       `json:"ipAddrGroupSrc,omitempty"`
	InvDlDst           bool        `json:"invDlDst,omitempty"`
	InvDlSrc           bool        `json:"invDlSrc,omitempty"`
	InvDlType          bool        `json:"invDlType,omitempty"`
	InvInPorts         bool        `json:"invInPorts,omitempty"`
	InvIPAddrGroupDst  bool        `json:"invIpAddrGroupDst,omitempty"`
	InvIPAddrGroupSrc  bool        `json:"invIpAddrGroupSrc,omitempty"`
	InvNwDst           bool        `json:"invNwDst,omitempty"`
	InvNwProto         bool        `json:"invNwProto,omitempty"`
	InvNwSrc           bool        `json:"invNwSrc,omitempty"`
	InvNwTos           bool        `json:"invNwTos,omitempty"`
	InvOutPorts        bool        `json:"invOutPorts,omitempty"`
	InvPortGroup       bool        `json:"invPortGroup,omitempty"`
	InvTpDst           bool        `json:"invTpDst,omitempty"`
	InvTpSrc           bool        `json:"invTpSrc,omitempty"`
	InvTraversedDevice bool        `json:"invTraversedDevice,omitempty"`
	MatchForwardFlow   bool        `json:"matchForwardFlow,omitempty"`
	MatchReturnFlow    bool        `json:"matchReturnFlow,omitempty"`
	NoVlan             bool        `json:"noVlan,omitempty"`
	NwDstAddress       string      `json:"nwDstAddress,omitempty"`
	NwDstLength        int         `json:"nwDstLength,omitempty"`
	NwProto            int         `json:"nwProto,omitempty"`
	NwSrcAddress       string      `json:"nwSrcAddress,omitempty"`
	NwSrcLength        int         `json:"nwSrcLength,omitempty"`
	NwTos              int         `json:"nwTos,omitempty"`
	OutPorts           []*UUID     `json:"outPorts,omitempty"`
	Position           int         `json:"position,omitempty"`
	PortGroup          *UUID       `json:"portGroup,omitempty"`
	TpDst              *MinMax     `json:"tpDst,omitempty"`
	TpSrc              *MinMax     `json:"tpSrc,omitempty"`
	TraversedDevice    *UUID       `json:"traversedDevice,omitempty"`
	Vlan               bool        `json:"vlan,omitempty"`
	Action             string      `json:"action,omitempty"`
	ChainID            *UUID       `json:"chainId,omitempty"`
	NatTargets         []NatTarget `json:"natTargets"`
}

//NatTarget 配置
type NatTarget struct {
	AddressTo   string `json:"addressTo"`
	AddressFrom string `json:"addressFrom"`
	PortTo      int    `json:"portTo"`
	PortFrom    int    `json:"portFrom"`
}

//MinMax 范围
type MinMax struct {
	Start int `json:"start,omitempty"`
	End   int `json:"end,omitempty"`
}

//Port 通信端口
type Port struct {
	ID                *UUID   `json:"id,omitempty"`
	Type              string  `json:"type"`
	AdminStateUp      bool    `json:"adminStateUp,omitempty"`
	InboundFilterID   *UUID   `json:"inboundFilterId,omitempty"`
	InboundMirrorIDs  []*UUID `json:"inboundMirrorIds,omitempty"`
	OutboundFilterID  *UUID   `json:"outboundFilterId,omitempty"`
	OutboundMirrorIDs []*UUID `json:"outboundMirrorIds,omitempty"`
	InsertionIDs      []*UUID `json:"insertionIds,omitempty"`
	DeviceID          *UUID   `json:"deviceId,omitempty"`
	InterfaceName     string  `json:"interfaceName,omitempty"`
	HostID            *UUID   `json:"hostId,omitempty"`
	PeerID            *UUID   `json:"peerId,omitempty"`
	TunnelKey         int     `json:"tunnelKey,omitempty"`
	VifID             *UUID   `json:"vifId,omitempty"`
}

//BridgePort 网桥端口
type BridgePort struct {
	Port
	VlanID *UUID `json:"vlanId"`
}

//RouterPort 路由器端口
type RouterPort struct {
	Port
	NetworkAddress string `json:"networkAddress"`
	NetworkLength  int    `json:"networkLength"`
	PortAddress    string `json:"portAddress"`
	PortMac        string `json:"portMac,omitempty"`
	BgpStatus      string `json:"bgpStatus,omitempty"`
}

//VxlanPort 扩展端口
type VxlanPort struct {
	Port
	VtepID *UUID `json:"vtepId"`
}

//Token 身份令牌
type Token struct {
	Key     string `json:"key,omitempty"`
	Expires string `json:"expires,omitempty"`
}

//HostInterfacePort binding allows mapping a virtual network port to an interface (virtual or physical) of a physical host where Midolman is running.
type HostInterfacePort struct {
	HostID        *UUID  `json:"hostId"`
	InterfaceName string `json:"interfaceName"`
	PortID        *UUID  `json:"portId"`
}

//PortLink 端口连接
type PortLink struct {
	PortID *UUID `json:"portId,omitempty"`
	PeerID *UUID `json:"peerId"`
}
