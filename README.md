# DnsGo

intercept and proxy dns in very sample way

## What's This

Proxy all Dns Request But Specific Domains

## Hwo to use

### 1. write your hosts.conf

```conf
example.com    127.0.0.1
test.example.com 192.168.1.1 2404:6800:4008:c06::6a
```
if request domain does not in the hosts.conf, it will be proxy by other dns server, default is 8.8.8.8

### 2. Run

```bash
go get github.com/ti/dnsgo
sudo dnsgo -c hosts.conf -proxy 8.8.8.8 -log true
```


## Library used in this project

* github.com/miekg/dns
