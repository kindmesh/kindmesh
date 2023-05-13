package app

import (
	"github.com/kindmesh/kindmesh/internal/proxy/envoy"
	"github.com/kindmesh/kindmesh/internal/watch/k8s"
	// "k8s.io/dns/pkg/netif"
)

func Run() {
	// ip := net.ParseIP("169.254.10.1")
	// mgr := netif.NewNetifManager([]net.IP{ip})
	// if err := mgr.AddDummyDevice("kindmesh"); err != nil {
	// panic(err)
	// }
	// watch crd/pods
	k8s.Watch()
	// start envoy control api
	envoy.ServeAPI()
}
