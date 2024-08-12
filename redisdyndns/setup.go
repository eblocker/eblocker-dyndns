package redisdyndns

import (
	"strings"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

func init() { plugin.Register("redisdyndns", setup) }

func setup(c *caddy.Controller) error {
	c.Next() // 'redisdyndns'
	domains := c.RemainingArgs()
	if len(domains) == 0 {
		return plugin.Error("redisdyndns", c.ArgErr())
	}
	for i, domain := range domains {
		// normalize to format: .sub.domain
		domains[i] = "." + strings.TrimPrefix(strings.TrimSuffix(domain, "."), ".")
	}
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return NewRedisDynDns(next, domains)
	})

	return nil
}
