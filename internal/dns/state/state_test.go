package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHijack(t *testing.T) {
	clusterDomain := ".svc.cluster.local"
	pod := "172.1.2.3"
	ns := "default"
	domain := "abc."
	serviceName := "abc.default"
	gwIP := "169.254.1.1"

	SetHijackIp(map[string]string{pod: ns}, map[string]string{pod: gwIP}, []string{serviceName}, clusterDomain)

	ip := GetHijackIP(domain, pod)
	assert.Equal(t, gwIP, ip.String())

	ip = GetHijackIP(domain+ns+".", pod)
	assert.Equal(t, gwIP, ip.String())
	ip = GetHijackIP(domain+ns+clusterDomain+".", pod)
	assert.Equal(t, gwIP, ip.String())

	ip = GetHijackIP(domain, "127.0.0.1")
	assert.Nil(t, ip)

	ip = GetHijackIP(domain+".abc.", pod)
	assert.Nil(t, ip)
}
