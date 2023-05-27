package main

import (
	// blank imports to make sure the plugin code is pulled in from vendor when building node-cache image

	"os"

	_ "github.com/coredns/coredns/plugin/bind"
	_ "github.com/coredns/coredns/plugin/bufsize"
	_ "github.com/coredns/coredns/plugin/cache"
	_ "github.com/coredns/coredns/plugin/debug"
	_ "github.com/coredns/coredns/plugin/dns64"
	_ "github.com/coredns/coredns/plugin/errors"
	_ "github.com/coredns/coredns/plugin/forward"
	_ "github.com/coredns/coredns/plugin/health"
	_ "github.com/coredns/coredns/plugin/hosts"
	_ "github.com/coredns/coredns/plugin/loadbalance"
	_ "github.com/coredns/coredns/plugin/log"
	_ "github.com/coredns/coredns/plugin/loop"
	_ "github.com/coredns/coredns/plugin/metrics"
	_ "github.com/coredns/coredns/plugin/pprof"
	_ "github.com/coredns/coredns/plugin/reload"
	_ "github.com/coredns/coredns/plugin/rewrite"
	_ "github.com/coredns/coredns/plugin/template"
	_ "github.com/coredns/coredns/plugin/trace"
	_ "github.com/coredns/coredns/plugin/whoami"
	_ "github.com/kindmesh/kindmesh/internal/dns/hijack"
	"github.com/kindmesh/kindmesh/internal/dns/server"
	"github.com/kindmesh/kindmesh/internal/spec"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/coremain"
)

func main() {
	os.Setenv("DNS_BIND_IP", spec.DNS_BIND_IP)

	dnsserver.Directives = append([]string{"hijack"}, dnsserver.Directives...)
	go func() {
		coremain.Run()
		panic("coredns exit")
	}()
	server.Serve(spec.DNS_BIND_IP + ":80")
}
