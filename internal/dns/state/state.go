package state

import (
	"net"
	"strings"
	"sync/atomic"
)

var state = atomic.Value{}

type hijackState struct {
	pod2NS        map[string]string
	pod2GwIP      map[string]net.IP
	allDomains    map[string]bool
	clusterDomain string
}

func init() {
	state.Store(&hijackState{pod2NS: map[string]string{}, allDomains: map[string]bool{}, pod2GwIP: map[string]net.IP{}})
}

// GetHijackIP returns dns ip by domain and clientIP
func GetHijackIP(domain, clientIP string) net.IP {
	m := state.Load().(*hijackState)
	if m.allDomains[domain] {
		return m.pod2GwIP[clientIP]
	}
	if m.allDomains[domain+m.clusterDomain] {
		return m.pod2GwIP[clientIP]
	}
	ns, ok := m.pod2NS[clientIP]
	if !ok {
		return nil
	}
	if m.allDomains[domain+ns+"."+m.clusterDomain] {
		return m.pod2GwIP[clientIP]
	}
	return nil
}

// SetHijackIp set hjiack config
func SetHijackIp(pod2NS, pod2GwIP map[string]string, serviceList []string, clusterDomain string) {
	clusterDomain = strings.TrimPrefix(clusterDomain, ".")
	if !strings.HasSuffix(clusterDomain, ".") {
		clusterDomain = clusterDomain + "."
	}

	allDomains := map[string]bool{}
	for _, v := range serviceList {
		if !strings.HasSuffix(v, ".") {
			v = v + "."
		}
		allDomains[v+clusterDomain] = true
	}
	newPod2GwIP := map[string]net.IP{}
	for k, v := range pod2GwIP {
		newPod2GwIP[k] = net.ParseIP(v)
	}
	state.Store(&hijackState{pod2NS: pod2NS, allDomains: allDomains, pod2GwIP: newPod2GwIP, clusterDomain: clusterDomain})
}
