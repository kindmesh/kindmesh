package state

import (
	"net"
	"strings"
	"sync/atomic"
)

var state = atomic.Value{}

type hijackState struct {
	pod2NS        map[string]string
	ns2GwIP       map[string]net.IP
	allDomains    map[string]bool
	clusterDomain string
}

func init() {
	state.Store(&hijackState{pod2NS: map[string]string{}, allDomains: map[string]bool{}, ns2GwIP: map[string]net.IP{}})
}

// GetHijackIP returns dns ip by domain and clientIP
func GetHijackIP(domain, clientIP string) net.IP {
	m := state.Load().(*hijackState)
	ns, ok := m.pod2NS[clientIP]
	if !ok {
		return nil
	}
	ip := m.ns2GwIP[ns]
	if ip == nil {
		return nil
	}
	if m.allDomains[domain] {
		return ip
	}
	if m.allDomains[domain+m.clusterDomain] {
		return ip
	}
	if m.allDomains[domain+ns+"."+m.clusterDomain] {
		return ip
	}
	return nil
}

// SetHijackIp set hjiack config
func SetHijackIp(pod2NS, ns2GwIP map[string]string, serviceList []string, clusterDomain string) {
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
	newNS2GwIP := map[string]net.IP{}
	for k, v := range ns2GwIP {
		newNS2GwIP[k] = net.ParseIP(v)
	}
	state.Store(&hijackState{pod2NS: pod2NS, allDomains: allDomains, ns2GwIP: newNS2GwIP, clusterDomain: clusterDomain})
}
