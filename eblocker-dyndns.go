package main

import (
	_ "github.com/eblocker/eblocker-dyndns/redisdyndns"

	_ "github.com/coredns/coredns/plugin/bufsize"
	_ "github.com/coredns/coredns/plugin/debug"
	_ "github.com/coredns/coredns/plugin/errors"
	_ "github.com/coredns/coredns/plugin/log"
	_ "github.com/coredns/coredns/plugin/timeouts"

	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/coremain"
)

var directives = []string{
	"timeouts",
	"bufsize",
	"debug",
	"errors",
	"log",
	"redisdyndns",
}

func init() {
	dnsserver.Directives = directives
}

func main() {
	coremain.Run()
}
