/*
 * Copyright 2024 eBlocker Open Source UG (haftungsbeschraenkt)
 *
 * Licensed under the EUPL, Version 1.2 or - as soon they will be
 * approved by the European Commission - subsequent versions of the EUPL
 * (the "License"); You may not use this work except in compliance with
 * the License. You may obtain a copy of the License at:
 *
 *   https://joinup.ec.europa.eu/page/eupl-text-11-12
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" basis,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
 * implied. See the License for the specific language governing
 * permissions and limitations under the License.
 */
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

// Database is a read-only key-value store
type Database interface {
	// Query looks up a string value given a query string
	Query(ctx context.Context, query string) (string, error)
}

type RedisDatabase struct {
	Client redis.Client
}

type RedisDynDns struct {
	Next     plugin.Handler
	Domains  []string
	Database Database
}

// newRedisDynDns creates a new plugin handler that uses Redis at localhost:6379
func NewRedisDynDns(next plugin.Handler, domains []string) *RedisDynDns {
	rd := RedisDynDns{
		Next:    next,
		Domains: domains,
		Database: &RedisDatabase{
			Client: *redis.NewClient(&redis.Options{
				Addr:     "localhost:6379",
				Password: "",
				DB:       0,
			}),
		},
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
	result, err := rd.Database.Query(ctx, key)
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

// Query implements interface Database
func (rdb *RedisDatabase) Query(ctx context.Context, query string) (string, error) {
	return rdb.Client.Get(ctx, query).Result()
}
