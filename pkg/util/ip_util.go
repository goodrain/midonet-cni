package util

import "net"

//IP2Long IP 转位
func IP2Long(ip net.IP) uint {
	return (uint(ip[12]) << 24) + (uint(ip[13]) << 16) + (uint(ip[14]) << 8) + uint(ip[15])
}

//Long2IP 位转 IP
func Long2IP(long uint) net.IP {
	return net.IPv4(byte(long>>24), byte(long>>16), byte(long>>8), byte(long))
}

//IsPrivate 是否为局域网IP
func IsPrivate(ip net.IP) bool {
	switch {
	case ip[0] == 10: // 10.0.0.0/8: 10.0.0.0 - 10.255.255.255
		return true
	case ip[0] == 172 && (ip[1] >= 16 && ip[1] < 32): // 172.16.0.0/12: 172.16.0.0 - 172.31.255.255
		return true
	case ip[0] == 192 && ip[1] == 168: // 192.168.0.0/16: 192.168.0.0 - 192.168.255.255
		return true
	}

	return false
}
