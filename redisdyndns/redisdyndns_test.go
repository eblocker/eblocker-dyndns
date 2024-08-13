package redisdyndns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"testing"

	"github.com/coredns/coredns/plugin/pkg/dnstest"
	"github.com/coredns/coredns/plugin/test"

	"github.com/miekg/dns"
	"github.com/redis/go-redis/v9"
)

type MockResponse struct {
	result string
	err    error
}

type MockDatabase struct {
	response map[string]MockResponse
}

func (md *MockDatabase) Query(ctx context.Context, query string) (string, error) {
	mr := md.response[query]
	return mr.result, mr.err
}

// TestServeDns tests responses from ServeDNS.
func TestServeDns(t *testing.T) {
	ctx := context.TODO()

	rdd := RedisDynDns{
		Next: test.ErrorHandler(),
		Domains: []string{
			"home.eblocker.org",
		},
		Database: &MockDatabase{
			map[string]MockResponse{
				"foo.home.eblocker.org":      {"1.2.3.4", nil},
				"foo.home.eblocker.org/AAAA": {"fe80::1:2:3:4", nil},
				"badip.home.eblocker.org":    {"not an IP", nil},
				"empty.home.eblocker.org":    {"", nil},
				"notfound.home.eblocker.org": {"", redis.Nil},
			},
		},
	}
	var tests = []struct {
		domain         string
		qtype          uint16
		expectedRcode  int
		expectedResult net.IP
	}{
		{"example.org", dns.TypeA, dns.RcodeNameError, nil},
		{"foo.home.eblocker.org", dns.TypeA, dns.RcodeSuccess, net.ParseIP("1.2.3.4")},
		{"foo.home.eblocker.org", dns.TypeAAAA, dns.RcodeSuccess, net.ParseIP("fe80::1:2:3:4")},
		{"badip.home.eblocker.org", dns.TypeA, dns.RcodeNameError, nil},
		{"empty.home.eblocker.org", dns.TypeA, dns.RcodeNameError, nil},
		{"notfound.home.eblocker.org", dns.TypeA, dns.RcodeNameError, nil},
	}
	for _, tt := range tests {
		testname := fmt.Sprintf("%s, %d", tt.domain, tt.qtype)
		t.Run(testname, func(t *testing.T) {
			r := new(dns.Msg)
			r.SetQuestion(tt.domain, tt.qtype)
			rec := dnstest.NewRecorder(&test.ResponseWriter{})
			rdd.ServeDNS(ctx, rec, r)
			if rec.Rcode != tt.expectedRcode {
				t.Errorf("Expected code %d, but got %d", tt.expectedRcode, rec.Rcode)
			}
			checkAnswer(rec.Msg.Answer, tt.domain, tt.expectedResult, t)
		})
	}
}

// checkAnswer checks whether there is exactly one record with the expected IP address
// or zero records if there is no IP address expected
func checkAnswer(rr []dns.RR, domain string, expectedIP net.IP, t *testing.T) {
	if expectedIP == nil {
		if len(rr) != 0 {
			t.Errorf("Expected zero records in answer, but got: %d", len(rr))
		}
	} else {
		if len(rr) != 1 {
			t.Errorf("Expected exactly one record in answer, but got: %d", len(rr))
		}
		var gotIP net.IP
		switch rr[0].Header().Rrtype {
		case dns.TypeA:
			gotIP = rr[0].(*dns.A).A
		case dns.TypeAAAA:
			gotIP = rr[0].(*dns.AAAA).AAAA
		default:
			gotIP = nil
		}
		if !gotIP.Equal(expectedIP) {
			t.Errorf("Expected %v but got %v", expectedIP, gotIP)
		}
		if domain != rr[0].Header().Name {
			t.Errorf("Expected name %s in record, but got: %s", domain, rr[0].Header().Name)
		}
	}
}

// TestServeDnsDBError tests whether ServeDNS returns a server failure if a database error occurs.
// The error will be logged by CoreDNS.
func TestServeDnsDBError(t *testing.T) {
	ctx := context.TODO()
	rdd := RedisDynDns{
		Next: test.ErrorHandler(),
		Domains: []string{
			"home.eblocker.org",
		},
		Database: &MockDatabase{
			map[string]MockResponse{
				"foo.home.eblocker.org": {"", errors.New("Database error!")},
			},
		},
	}
	r := new(dns.Msg)
	r.SetQuestion("foo.home.eblocker.org", dns.TypeA)
	rec := dnstest.NewRecorder(&test.ResponseWriter{})
	code, err := rdd.ServeDNS(ctx, rec, r)
	if code != dns.RcodeServerFailure {
		t.Errorf("Expected return code %d of ServeDNS, but got %d", dns.RcodeServerFailure, code)
	}
	if err == nil {
		t.Errorf("Expected an error from ServeDNS")
	}
}
