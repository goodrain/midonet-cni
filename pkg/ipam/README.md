# key/value说明

## 分布式锁相关
* `/midonet-cni/router_ip` 操作router ip的锁
* `/midonet-cni/:tenantID/pod_ip` 操作租户pod ip的锁
* `/midonet-cni/tenant/:tenantID/create` 创建tenant并初始化的锁
* `/midonet-cni/tenant/:tenantID/bridge/create` 创建新bridge的锁

## 数据相关

* `/midonet-cni/tenant/:tenantID/info` 租户信息
* `/midonet-cni/tenant/:tenantID/router` 租户router信息
* `/midonet-cni/tenant/:tenantID/bridge/:bridgeID` 租户bridge信息
* `/midonet-cni/tenant/:tenantID/bridge/usage` 正在使用的bridge

* `/midonet-cni/ip/router/available` router ip可用池。value为网段，下级node key为可用IP
* `/midonet-cni/ip/pod/:tenantID/available` 租户pod ip可用池，value为网段，下级node key为可用ip
* `/midonet-cni/ip/pod/:tenantID/:containerID/`租户pod ip信息，value为 ip信息。 ->需要删除，还原到可用池（ip段未改变）

* `/midonet-cni/bingding/:tenantID/:containerID` 容器网卡绑定状态 ->需要删除
* `/midonet-cni/result/:tenantID/:containerID` cni结果  ->需要删除
