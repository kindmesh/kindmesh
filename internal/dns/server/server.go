package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/kindmesh/kindmesh/internal/dns/state"
	"github.com/kindmesh/kindmesh/internal/spec"
)

func Serve(addr string) {
	/*
		curl -d '{"Pod2NS": {"169.254.99.1": "default"}, "NS2GwIP": {"default": "169.254.100.100"}, "ClusterDomain": "svc.cluster.local.", "ServiceList": ["abc.default."]}' 169.254.99.1/set-dns-hijack
	*/
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		var p spec.DNSRequest
		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		state.SetHijackIp(p.Pod2NS, p.NS2GwIP, p.ServiceList, p.ClusterDomain)
		fmt.Fprintf(w, "update success")
	})

	for {
		err := http.ListenAndServe(addr, nil)
		log.Println("dns control server err", err)
		time.Sleep(time.Second * 10)
	}
}
