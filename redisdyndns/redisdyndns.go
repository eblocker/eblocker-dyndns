package redisdyndns

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/coredns/coredns/plugin"
	"github.com/coredns/coredns/request"
	"github.com/miekg/dns"
	"github.com/redis/go-redis/v9"
)

const ttl = 60

type RedisDynDns struct {
	Next    plugin.Handler
	Domains []string
	Client  redis.Client
}

// newRedisDynDns creates a new plugin handler that uses Redis at localhost:6379
func NewRedisDynDns(next plugin.Handler, domains []string) *RedisDynDns {
	rd := RedisDynDns{
		Next:    next,
		Domains: domains,
		Client: *redis.NewClient(&redis.Options{
			Addr:     "localhost:6379",
			Password: "",
			DB:       0,
		}),
	}
	return &rd
}

// Name implements plugin.Handler.
func (r RedisDynDns) Name() string {
	return "redisdyndns"
}

// ServeDNS implements plugin.Handler.
func (rd *RedisDynDns) ServeDNS(ctx context.Context, writer dns.ResponseWriter, req *dns.Msg) (int, error) {
	state := request.Request{W: writer, Req: req}
	resp := new(dns.Msg)
	resp.Authoritative = true
	domain := strings.TrimSuffix(state.Name(), ".")
	if rd.matches(domain) {
		if key := getKey(state.QType(), domain); key != "" {
			ip, err := rd.getIp(ctx, key)
			if err != nil {
				// Redis failed
				return dns.RcodeServerFailure, err
			}
			if ip != nil {
				addAnswer(ip, &state, resp)
			} else {
				// ip not found in Redis
				resp.SetRcode(req, dns.RcodeNameError)
			}
		} else {
			// type not supported
			resp.SetRcode(req, dns.RcodeNameError)
		}
	} else {
		// domain suffix does not match configured domains
		resp.SetRcode(req, dns.RcodeNameError)
	}

	writer.WriteMsg(resp)
	// Return success as the rcode to signal we have written to the client.
	return dns.RcodeSuccess, nil
}

// addAnswer adds an IPv4 or IPv6 address to the response
func addAnswer(ip net.IP, state *request.Request, resp *dns.Msg) {
	resp.SetReply(state.Req)
	if state.QType() == dns.TypeAAAA {
		aaaa := new(dns.AAAA)
		aaaa.AAAA = ip
		aaaa.Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeAAAA, Class: state.QClass(), Ttl: ttl}
		resp.Answer = []dns.RR{aaaa}
	} else {
		a := new(dns.A)
		a.A = ip
		a.Hdr = dns.RR_Header{Name: state.QName(), Rrtype: dns.TypeA, Class: state.QClass(), Ttl: ttl}
		resp.Answer = []dns.RR{a}
	}
}

// getIp reads an IP address from Redis
func (rd *RedisDynDns) getIp(ctx context.Context, key string) (net.IP, error) {
	result, err := rd.Client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // domain not found
		} else {
			return nil, fmt.Errorf("error getting '%s' from redis: %w", key, err)
		}
	}
	return net.ParseIP(result), nil
}

// getKey returns the key to search for in Redis for a given domain.
// Only queries of type A and AAAA are supported.
func getKey(qtype uint16, domain string) string {
	switch qtype {
	case dns.TypeA:
		return domain
	case dns.TypeAAAA:
		return domain + "/AAAA"
	default:
		return ""
	}
}

// matches checks whether the given domain is a subdomain of any of the configured domains
func (rd *RedisDynDns) matches(domain string) bool {
	for _, suffix := range rd.Domains {
		if strings.HasSuffix(domain, suffix) {
			return true
		}
	}
	return false
}
