package util

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

//List 列出全部ip
func List(iprange string) (ips []string, err error) {
	r, err := NewRange(iprange)
	if err != nil {
		return nil, err
	}

	for {
		ips = append(ips, r.String())
		if !r.Next() {
			break
		}
	}

	return
}

//RangeLength 获取网段所有ip数量
func RangeLength(iprange string) int {
	ips, err := List(iprange)
	if err != nil {
		return 0
	}
	return len(ips)
}

//GetNextCIDR 获取下一个网段  172.168.0.0/24 ->172.168.1.0/24
func GetNextCIDR(iprange string) (string, error) {
	ip, ipnet, err := net.ParseCIDR(iprange)
	if err != nil {
		return "", err
	} else if !ip.Equal(ipnet.IP) {
		return "", errors.New("Invalid cidr")
	}

	var iprangeMask int
	if slashPos := strings.LastIndex(iprange, "/"); slashPos == -1 {
		iprangeMask = 32
	} else {
		iprangeMask, err = strconv.Atoi(iprange[slashPos+1:])
		if err != nil {
			return "", err
		}
	}
	newIP := Long2IP(IP2Long(ip) + 1<<uint(32-iprangeMask))
	return newIP.String() + fmt.Sprintf("/%d", iprangeMask), nil
}

//NewRange 网段内ip范围
func NewRange(iprange string) (*Range, error) {
	if iprange == "" {
		return nil, fmt.Errorf("iprange can not be empty")
	}
	return NewRangeWithBlockSize(iprange, 32)
}

//NewRangeWithBlockSize 创建Range 通过设置总位数 ipv4 32 ipv6 64
func NewRangeWithBlockSize(iprange string, blockSize int) (*Range, error) {
	ip, ipnet, err := net.ParseCIDR(iprange)
	if err != nil {
		return nil, err
	} else if !ip.Equal(ipnet.IP) {
		return nil, errors.New("Invalid cidr")
	}

	var iprangeMask int
	if slashPos := strings.LastIndex(iprange, "/"); slashPos == -1 {
		iprangeMask = 32
	} else {
		iprangeMask, err = strconv.Atoi(iprange[slashPos+1:])
		if err != nil {
			return nil, err
		}
	}

	if iprangeMask > blockSize || blockSize > 32 {
		return nil, errors.New("Invalid block size")
	}

	return &Range{
		ip:         ip,
		iplong:     IP2Long(ip),
		ipnet:      ipnet,
		stepprefix: "/" + strconv.Itoa(blockSize),
		stepsuffix: "/" + strconv.Itoa(iprangeMask),
		lastiplong: IP2Long(ip) + (1 << uint(32-iprangeMask)) - 1,
		step:       1 << uint(32-blockSize),
	}, nil
}

type Range struct {
	ipnet      *net.IPNet
	ip         net.IP
	iplong     uint
	lastiplong uint
	stepprefix string
	stepsuffix string
	step       uint
}

func (r *Range) Next() bool {
	r.iplong += r.step
	r.ip = Long2IP(r.iplong)

	return r.iplong <= r.lastiplong
}

func (r *Range) String() string {
	return r.ip.String()
}

func (r *Range) StringPrefix() string {
	return r.ip.String() + r.stepprefix
}

func (r *Range) StringSuffix() string {
	return r.ip.String() + r.stepsuffix
}
