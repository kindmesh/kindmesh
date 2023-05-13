package state

import (
	"fmt"
	"net"
	"sync/atomic"
)

var state = atomic.Value{}
var localDomain = "svc.cluster.local."

type hijackState struct {
	pod2NS     map[string]string
	allDomains map[string]bool
	pod2GwIP   map[string]net.IP
}

func init() {
	state.Store(&hijackState{pod2NS: map[string]string{}, allDomains: map[string]bool{}, pod2GwIP: map[string]net.IP{}})
}

func GetHijackIP(domain, clientIP string) net.IP {
	m := state.Load().(*hijackState)
	fmt.Println("get", domain, clientIP, m.pod2NS, m.allDomains, m.pod2GwIP)
	if m.allDomains[domain] {
		return m.pod2GwIP[clientIP]
	}
	if m.allDomains[domain+localDomain] {
		return m.pod2GwIP[clientIP]
	}
	ns, ok := m.pod2NS[clientIP]
	if !ok {
		return nil
	}
	if m.allDomains[domain+ns+"."+localDomain] {
		return m.pod2GwIP[clientIP]
	}
	return nil
}

func SetHijackIp(pod2NS, pod2GwIP map[string]string, allDomains map[string]bool) {
	newPod2GwIP := map[string]net.IP{}
	for k, v := range pod2GwIP {
		newPod2GwIP[k] = net.ParseIP(v)
	}
	state.Store(&hijackState{pod2NS: pod2NS, allDomains: allDomains, pod2GwIP: newPod2GwIP})
}
