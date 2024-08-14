DNS_HOME = $(DESTDIR)/opt/eblocker-dyndns

test:
	go test ./...

eblocker-dyndns:
	go build

install: eblocker-dyndns
	mkdir -p $(DNS_HOME) $(DNS_HOME)/bin
	cp eblocker-dyndns $(DNS_HOME)/bin
	cp Corefile $(DNS_HOME)

package:
	dpkg-buildpackage -us -uc -b
