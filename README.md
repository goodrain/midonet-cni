#  Midonet CNI Plugin For Kubernetes

此插件实现基于midonet网络的cni插件，支持多租户网络。

# 项目状态

* v1:   
 集合region api已有的功能实现cni.
* v2:    
 golang实现veth管理,保留shell版本。可配置使用。依赖region
* v3:   
 ip管理，midonet直接操作。不依赖region

# Feature

1. midonet 支持多租户特性。
2. 基于etcd的IP管理，租户级别全局一致性。
3. 容器网络管理，网络路由管理。
4. 增强的幂等特性

 

  
  
