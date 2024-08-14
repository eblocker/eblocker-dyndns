# This script tests whether the DNS server crashes if the source UDP port is zero.
from scapy.all import DNS, DNSQR, IP, send, UDP

# Define the target IP and target port
target_ip = "127.0.0.1"
target_port = 5053

# Create a DNS query packet
# DNSQR: DNS Question Record
dns_query = DNS(rd=1, qd=DNSQR(qname="foo.home.eblocker.com"))

# Create a UDP packet specifying source and destination ports
udp = UDP(sport=0, dport=target_port)

# Create an IP packet
ip = IP(dst=target_ip)

# Stack the layers
packet = ip / udp / dns_query

# Send the packet
send(packet)
