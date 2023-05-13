package state

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHijack(t *testing.T) {
	pod := "172.1.2.3"
	ns := "default"
	domain := "abc."
	fullDomain := "abc.default." + localDomain
	gwIP := "169.254.1.1"

	SetHijackIp(map[string]string{pod: ns}, map[string]string{pod: gwIP}, map[string]bool{fullDomain: true})

	ip := GetHijackIP(domain, pod)
	assert.Equal(t, gwIP, ip.String())

	ip = GetHijackIP(domain+ns+".", pod)
	assert.Equal(t, gwIP, ip.String())
	ip = GetHijackIP(domain+ns+"."+localDomain, pod)
	assert.Equal(t, gwIP, ip.String())

	ip = GetHijackIP(domain, "127.0.0.1")
	assert.Nil(t, ip)

	ip = GetHijackIP(domain+ns+".abc", pod)
	assert.Nil(t, ip)
}
