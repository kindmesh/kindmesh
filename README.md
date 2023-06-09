# KindMesh

KindMesh的目标是为Kubernetes提供低延迟、高可用、具有丰富流量治理功能的网络通信能力

# Feature

- 将CoreDNS部署在每个Node上，集群内DNS在本地直接返回，集群外的DNS支持本地缓存，实现极致的DNS性能
- 控制面使用L7Service(CRD)定义转发规则和流量配置，支持丰富的流量治理功能
- 数据面将Envoy部署在每个Node上，实现低延迟的服务网络通信

## Architecture

![alt text](doc/arch1.png "Title")

## Pre Requirements

- 安装 Kubernetes，本地测试可使用[Kind](https://kind.sigs.k8s.io/)来安装。


- 安装 CRD
```
kubectl apply -f resource/l7service_crd.yaml
```
- 部署DaemonSet
```
kubectl apply -f resource/daemonset.yaml
```
- 配置DNS

修改kubelet的 --cluster-dns 参数为169.254.99.1，即可不用在POD中配置，比如可以这样配置Kind的集群：
```
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
- role: control-plane
- role: worker
  kubeadmConfigPatches:
  - |
    kind: JoinConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        cluster-dns: 169.254.99.1
- role: worker
  kubeadmConfigPatches:
  - |
    kind: JoinConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        cluster-dns: 169.254.99.1
```

或在POD中配置DNS为169.254.99.1，如
```
apiVersion: v1
kind: Pod
metadata:
  namespace: default
  name: dns-example
spec:
  containers:
    - name: test
      image: nginx
  dnsPolicy: "None"
  dnsConfig:
    nameservers:
      - 169.254.99.1 # 固定的本地地址
```


## Example

1. 部署示例Deployment
```
kubectl apply -f resource/example/bookinfo/deployments.yaml
```

3. 配置L7 Service

```
kubectl apply -f resource/example/bookinfo/l7services.yaml
```

```
apiVersion: v1
kind: L7Service
metadata:
  name: ratings
spec:
  selector:
    app: ratings
  containerPort: 8080
```
以上示例在l7services.yaml定义了raings域名，即可以在集群内通过域名 raings或ratings.(namespace)，或ratings.(namespace).svc.cluster.local来访问对应deployment中的容器。
