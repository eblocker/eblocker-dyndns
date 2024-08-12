package redisdyndns

import (
	"testing"

	"github.com/coredns/caddy"
)

// TestSetup tests parsing of the CoreDNS configuration file
func TestSetup(t *testing.T) {
	c := caddy.NewTestController("dns", `redisdyndns`)
	if err := setup(c); err == nil {
		t.Errorf("Expected error because of missing domains")
	}

	c = caddy.NewTestController("dns", `redisdyndns home.eblocker.com`)
	if err := setup(c); err != nil {
		t.Errorf("Expected no errors, but got: %v", err)
	}

	c = caddy.NewTestController("dns", `redisdyndns home.eblocker.com home.test.eblocker.com`)
	if err := setup(c); err != nil {
		t.Errorf("Expected no errors, but got: %v", err)
	}
}
