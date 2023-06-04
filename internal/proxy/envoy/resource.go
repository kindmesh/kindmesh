package envoy

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/durationpb"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	listener "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	router "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	"github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/kindmesh/kindmesh/internal/spec"
)

const (
	xdsClusterName = "kind_xds_cluster"
)

func makeCluster(svc *spec.L7Service, req *spec.RouterRequest) *cluster.Cluster {
	name := svc.MetaData.Namespace + "_" + svc.MetaData.Name
	// TODO: check has subset
	// selector
	ipSet := map[string]bool{}
	isFrist := true
	for k, v := range svc.Spec.Selector {
		isFrist = true
		ipList, ok := req.Label2Pod[fmt.Sprintf("%s:%s", k, v)]
		if !ok {
			ipSet = map[string]bool{}
			break
		}
		// first label
		if isFrist {
			for ip := range ipList {
				// double check ip exists
				if _, ok := req.Pod2NS[ip]; !ok {
					continue
				}
				ipSet[ip] = true
			}
			continue
		}
		// not first label: try intersection
		for ip := range ipList {
			if !ipSet[ip] {
				delete(ipSet, ip)
			}
		}
	}
	endpoints := []*endpointInfo{}
	for ip := range ipSet {
		if ip == "" {
			fmt.Println("ip is empty?", svc.MetaData.Name, ipSet)
			continue
		}
		endpoints = append(endpoints, &endpointInfo{ip: ip, port: svc.Spec.TargetPort})
	}
	cls := makeEndpoint(name, endpoints)
	return &cluster.Cluster{
		Name:           name,
		ConnectTimeout: durationpb.New(5 * time.Second),
		LbPolicy:       cluster.Cluster_ROUND_ROBIN, // TODO configable
		LoadAssignment: cls,
	}
}

type endpointInfo struct {
	ip   string
	port uint32
}

func makeEndpoint(name string, eps []*endpointInfo) *endpoint.ClusterLoadAssignment {
	endpoints := []*endpoint.LbEndpoint{}
	for _, ep := range eps {
		ee := &endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: &core.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Protocol: core.SocketAddress_TCP,
								Address:  ep.ip,
								PortSpecifier: &core.SocketAddress_PortValue{
									PortValue: ep.port,
								},
							},
						},
					},
				},
			},
		}
		endpoints = append(endpoints, ee)
	}
	return &endpoint.ClusterLoadAssignment{
		ClusterName: name,
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			LbEndpoints: endpoints,
		}},
	}
}

func makeRoute(ns string, req *spec.RouterRequest) (*route.RouteConfiguration, error) {
	virtualHosts := []*route.VirtualHost{}
	for _, svc := range req.ServiceList {
		domain := svc.MetaData.Name + "." + svc.MetaData.Namespace
		name := svc.MetaData.Namespace + "_" + svc.MetaData.Name
		// current service + all service.ns + all service.ns.cluster.domain
		domains := []string{domain, domain + "." + req.ClusterDomain}
		if ns == svc.MetaData.Namespace {
			domains = append(domains, svc.MetaData.Name)
		}
		routers := []*route.Route{}
		if len(routers) == 0 {
			route := &route.Route{
				Match: &route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{Cluster: name},
					},
				},
			}
			routers = append(routers, route)
		}
		hh := &route.VirtualHost{
			Name:    name,
			Domains: domains,
			Routes:  routers,
		}
		virtualHosts = append(virtualHosts, hh)
	}
	return &route.RouteConfiguration{
		Name:         ns,
		VirtualHosts: virtualHosts,
	}, nil
}

func makeRouteV2(routeName string, clusterName string) *route.RouteConfiguration {
	return &route.RouteConfiguration{
		Name: routeName,
		VirtualHosts: []*route.VirtualHost{{
			Name:    "local_service",
			Domains: []string{"*"},
			Routes: []*route.Route{{
				Match: &route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: clusterName,
						},
					},
				},
			}},
		}},
	}
}

func makeHTTPListener(namespace, gwIP string) (*listener.Listener, error) {
	routerConfig, _ := anypb.New(&router.Router{})
	// HTTP filter configuration
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource:    makeConfigSource(),
				RouteConfigName: namespace,
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name:       wellknown.Router,
			ConfigType: &hcm.HttpFilter_TypedConfig{TypedConfig: routerConfig},
		}},
	}
	pbst, err := anypb.New(manager)
	if err != nil {
		return nil, err
	}

	return &listener.Listener{
		Name: namespace,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  gwIP,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: 80,
					},
				},
			},
		},
		FilterChains: []*listener.FilterChain{{
			Filters: []*listener.Filter{{
				Name: wellknown.HTTPConnectionManager,
				ConfigType: &listener.Filter_TypedConfig{
					TypedConfig: pbst,
				},
			}},
		}},
	}, nil
}

func makeConfigSource() *core.ConfigSource {
	source := &core.ConfigSource{}
	source.ResourceApiVersion = resource.DefaultAPIVersion
	source.ConfigSourceSpecifier = &core.ConfigSource_ApiConfigSource{
		ApiConfigSource: &core.ApiConfigSource{
			TransportApiVersion:       resource.DefaultAPIVersion,
			ApiType:                   core.ApiConfigSource_GRPC,
			SetNodeOnFirstMessageOnly: true,
			GrpcServices: []*core.GrpcService{{
				TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
					EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: xdsClusterName},
				},
			}},
		},
	}
	return source
}

var version int

func GenerateSnapshot(req *spec.RouterRequest) error {
	bs, _ := json.Marshal(req)
	log.Printf("will serve snapshot %+v\n", string(bs))
	version++
	lds := []types.Resource{}
	for ns, gwIP := range req.NS2GwIP {
		ld, err := makeHTTPListener(ns, gwIP)
		if err != nil {
			return err
		}
		lds = append(lds, ld)
	}

	rds := []types.Resource{}
	for ns := range req.NS2GwIP {
		rd, err := makeRoute(ns, req)
		if err != nil {
			return err // TODO: skip
		}
		rds = append(rds, rd)
		// rds = append(rds, spec.RDSInfo{Name: ns, VirtualHosts: vhosts})
	}

	cds := []types.Resource{}
	for _, svc := range req.ServiceList {
		cds = append(cds, makeCluster(svc, req))
	}

	snap, err := cache.NewSnapshot(strconv.Itoa(version),
		map[resource.Type][]types.Resource{
			resource.ListenerType: lds,
			resource.RouteType:    rds,
			resource.ClusterType:  cds,
		},
	)
	if err != nil {
		return err
	}

	// Create the snapshot that we'll serve to Envoy
	if err := snap.Consistent(); err != nil {
		return err
	}

	// Add the snapshot to the cache
	if err := snapCache.SetSnapshot(context.Background(), "nodeID", snap); err != nil {
		return err
	}
	return nil
}
