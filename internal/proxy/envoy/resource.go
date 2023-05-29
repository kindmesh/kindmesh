package envoy

import (
	"context"
	"encoding/json"
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

func makeCluster(cds spec.CDSInfo) *cluster.Cluster {
	return &cluster.Cluster{
		Name:           cds.Name,
		ConnectTimeout: durationpb.New(5 * time.Second),
		LbPolicy:       cluster.Cluster_ROUND_ROBIN, // TODO configable
		LoadAssignment: makeEndpoint(cds),
	}
}

func makeEndpoint(cds spec.CDSInfo) *endpoint.ClusterLoadAssignment {
	endpoints := []*endpoint.LbEndpoint{}
	for _, ep := range cds.Endpoints {
		ee := &endpoint.LbEndpoint{
			HostIdentifier: &endpoint.LbEndpoint_Endpoint{
				Endpoint: &endpoint.Endpoint{
					Address: &core.Address{
						Address: &core.Address_SocketAddress{
							SocketAddress: &core.SocketAddress{
								Protocol: core.SocketAddress_TCP,
								Address:  ep.IP,
								PortSpecifier: &core.SocketAddress_PortValue{
									PortValue: ep.Port,
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
		ClusterName: cds.Name,
		Endpoints: []*endpoint.LocalityLbEndpoints{{
			LbEndpoints: endpoints,
		}},
	}
}

func makeRoute(rds spec.RDSInfo) (*route.RouteConfiguration, error) {
	virtualHosts := []*route.VirtualHost{}
	for _, info := range rds.VirtualHosts {
		routers := []*route.Route{}
		for _, routeJSON := range info.Routers {
			router := &route.Route{}
			if err := json.Unmarshal(routeJSON, router); err != nil {
				return nil, err
			}
			routers = append(routers, router)
		}
		if len(info.Routers) == 0 {
			route := &route.Route{
				Match: &route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{Cluster: info.Cluster},
					},
				},
			}
			routers = append(routers, route)
		}
		hh := &route.VirtualHost{
			Name:    info.Name,
			Domains: info.Domains,
			Routes:  routers,
		}
		virtualHosts = append(virtualHosts, hh)
	}
	return &route.RouteConfiguration{
		Name:         rds.Name,
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

func makeHTTPListener(lds spec.LDSInfo) (*listener.Listener, error) {
	routerConfig, _ := anypb.New(&router.Router{})
	// HTTP filter configuration
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.HttpConnectionManager_AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource:    makeConfigSource(),
				RouteConfigName: lds.Name,
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
		Name: lds.Name,
		Address: &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.SocketAddress_TCP,
					Address:  lds.IP,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: lds.Port,
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

func GenerateSnapshot(req *spec.RouterRequst) error {
	version++
	lds := []types.Resource{}
	for _, info := range req.LDS {
		ld, err := makeHTTPListener(info)
		if err != nil {
			return err
		}
		lds = append(lds, ld)
	}
	rds := []types.Resource{}
	for _, info := range req.RDS {
		rd, err := makeRoute(info)
		if err != nil {
			return err
		}
		rds = append(rds, rd)
	}
	cds := []types.Resource{}

	for _, info := range req.CDS {
		cd := makeCluster(info)
		cds = append(cds, cd)
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

	log.Printf("will serve snapshot %+v\n", snap)

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
