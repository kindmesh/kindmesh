package hijack

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

func init() {
	plugin.Register(name, setup)
}

func setup(c *caddy.Controller) error {
	c.Next()
	if c.NextArg() {
		return plugin.Error(name, c.ArgErr())
	}
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		hijacking := Hijacking{
			Next: next,
		}
		return hijacking
	})

	return nil
}
