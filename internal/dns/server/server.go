package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/kindmesh/kindmesh/internal/dns/state"
)

type hijackState struct {
	Pod2NS     map[string]string
	Pod2GwIP   map[string]string
	AllDomains []string
}

func Serve(addr string) {

	/*
		curl -d '{"Pod2NS": {"127.0.0.1": "default"}, "Pod2GwIP": {"127.0.0.1": "169.254.10.1"}, "AllDomains": ["abc.default.svc.cluster.local."]}' 127.0.0.1:19001/set-dns-hijack
	*/
	http.HandleFunc("/set-dns-hijack", func(w http.ResponseWriter, r *http.Request) {
		var p hijackState
		err := json.NewDecoder(r.Body).Decode(&p)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		allDomains := map[string]bool{}
		for _, v := range p.AllDomains {
			allDomains[v] = true
		}
		state.SetHijackIp(p.Pod2NS, p.Pod2GwIP, allDomains)
		fmt.Fprintf(w, "update success")
	})

	http.ListenAndServe(addr, nil)
}
