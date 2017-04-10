#  Midonet CNI Plugin For Kubernetes
由北京[好雨云](https://www.goodrain.com)开源ETCD版 midonet-cni插件。
此插件实现基于midonet网络的cni插件，支持多租户网络。

# Feature

1. 支持midonet多租户特性,使用kubernetes namespace。
2. 基于etcd的IP地址管理，租户级别全局一致性。
3. 容器网络管理，网络路由管理。
4. 插件增强的幂等特性

# 工作依赖
 * etcd
 * midonet
 * keystone
 
# 构建安装

```
go get github.com/goodrain/midonet-cni
cd $GOPATH/src/github.com/goodrain/midonet-cni
go build -o midonet-cni
cp midonet-cni /opt/cni/bin/midonet-cni
```
  
  
