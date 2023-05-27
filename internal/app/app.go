package app

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"

	"github.com/kindmesh/kindmesh/internal/proxy/envoy"
	"github.com/kindmesh/kindmesh/internal/spec"
	"github.com/kindmesh/kindmesh/internal/watch/k8s"
	"github.com/kindmesh/kindmesh/internal/watch/netdevice"
	"github.com/kindmesh/kindmesh/internal/watch/processor"
	_ "k8s.io/dns/pkg/netif"
)

func Run() {
	// watch crd/pods
	netdevice.EnsureDevice("bridge0")
	netdevice.AddAddr(spec.DNS_BIND_IP)
	netdevice.AddAddr(spec.ENVOY_CONTROL_IP)

	processor.Emitor = func(dns *spec.DNSRequest, router *spec.RouterRequst) {
		buf, _ := json.Marshal(dns)
		body := bytes.NewBuffer(buf)
		resp, err := http.Post("http://"+spec.DNS_BIND_IP, "application/json", body)
		if err != nil {
			log.Println("set dns error", err)
		} else {
			log.Println("set dns succ", *dns)
			resp.Body.Close()
		}
		// log.Printf("ds %v %v\n", *dns, *router)
	}

	k8s.Watch()
	// start envoy control api
	envoy.ServeAPI()
}
