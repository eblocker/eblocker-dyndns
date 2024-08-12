# eblocker-dyndns

EBlocker-DynDNS is a [CoreDNS](https://coredns.io/) plugin for eBlocker Mobile's DynDNS service.

IP addresses are read from Redis.

* For queries of type `A` the key is the domain (without a `.` at the end)
* For queries of type `AAAA` the key is the domain with `/AAAA` appended
* All other queries are answered with `NXDOMAIN`.
