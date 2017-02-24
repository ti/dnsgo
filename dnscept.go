package main

//intercept and proxy dns in very sample way
//use nslookup example.com 127.0.0.1 to check

import (
	"github.com/miekg/dns"
	"strings"
	"net"
	"log"
	"path/filepath"
	"os"
	"flag"
	"bufio"
)

func main() {
	var proxyPass,conf string
	var logAble bool
	var ttl uint
	var serverHosts map[string][]net.IP
	serverHosts = make(map[string][]net.IP)
	flag.UintVar(&ttl,"ttl", 600, "time to live")
	flag.BoolVar(&logAble, "log", false, "log requests to stdout")
	flag.StringVar(&conf, "c", "/etc/dns/hosts.conf", "server hosts config file")
	flag.StringVar(&proxyPass, "proxy", "8.8.8.8", "default proxy dns server")
	flag.Parse()
	if file, err := os.Open(conf); err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if !strings.HasPrefix(line, "#") {
				fields := strings.Fields(line)
				serverHosts[fields[0]] = []net.IP{net.ParseIP(fields[1])}
				if len(fields) > 2 && strings.Contains(fields[2],":") {
					serverHosts[fields[0]]  = append(serverHosts[fields[0]],net.ParseIP(fields[2]))
				}
			}
		}
	} else {
		if p, err := filepath.Abs(conf); err == nil {
			conf = p
		}
		log.Printf("error for load conf file %s for %s \n", conf, err)
	}
	//start server
	h := NewDnsHandler("tcp",serverHosts)
	h.TTL = uint32(ttl)
	h.LogAble = logAble
	h.ProxyPass = proxyPass
	go func() {
		srv := &dns.Server{Addr: ":53", Net: "udp", Handler: h.NewProto("udp")}
		err := srv.ListenAndServe()
		if err != nil {
			log.Fatalf("Error for set udp listener %s\n", err)
		}
	}()
	log.Println("run dns server on port 53")
	srv := &dns.Server{Addr: ":53", Net: "tcp", Handler: h}
	err := srv.ListenAndServe();
	if err != nil {
		log.Fatalf("Error for set tcp listener %s\n", err)
	}

}


func NewDnsHandler(proto string,serverHosts map[string][]net.IP) *dnsHandler {
	h := &dnsHandler{Proto:proto,ServerHosts:serverHosts}
	h.TTL = 600
	h.ProxyPass = "8.8.8.8"
	return h
}

type dnsHandler struct {
	Proto      string
	ServerHosts map[string][]net.IP
	ProxyPass  string
	TTL  uint32
	LogAble bool
}

func (h *dnsHandler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	domain := strings.ToLower(r.Question[0].Name)
	if h.LogAble  {
		ip, _, _ := net.SplitHostPort(w.RemoteAddr().String())
		log.Printf("recive request from %s\t%s\n", ip, domain)
	}
	host, ok := h.ServerHosts[domain[:len(domain)-1]]
	if !ok {
		host, ok = h.ServerHosts["*"]
	}
	var msg *dns.Msg
	if ok {
		msg = new(dns.Msg)
		msg.SetReply(r)
		msg.Authoritative = true
		rrA := new(dns.A)
		rrA.Hdr = dns.RR_Header{Name: domain, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: h.TTL}
		rrA.A = host[0]
		msg.Answer = []dns.RR{rrA}
		if len(host) > 1 {
			rrAAAA := new(dns.AAAA)
			rrAAAA.Hdr = dns.RR_Header{Name: domain, Rrtype: dns.TypeAAAA, Class: dns.ClassINET, Ttl: h.TTL}
			rrAAAA.AAAA = host[1]
			msg.Answer = append(msg.Answer,rrAAAA)
		}
	}  else {
		r.Question[0].Name = domain
		//if you want cache, do something here
		client := new(dns.Client)
		client.Net = h.Proto
		var err error
		if msg, _, err = client.Exchange(r, h.ProxyPass + ":53"); err == nil {
			msg.Question[0].Name = strings.ToLower(msg.Question[0].Name)
			for i := 0; i < len(msg.Answer); i++ {
				msg.Answer[i].Header().Name = strings.ToLower(msg.Answer[i].Header().Name)
			}
		} else {
			log.Println(err)
			//https://support.opendns.com/hc/en-us/articles/227986827-FAQ-What-are-common-DNS-return-or-response-codes-
			msg = new(dns.Msg)
			msg.SetReply(r)
			msg.Authoritative = true
			msg.SetRcode(r,2)
		}
	}
	w.WriteMsg(msg)
}

func (h *dnsHandler) NewProto(proto string) *dnsHandler{
	return &dnsHandler{Proto:proto,ServerHosts:h.ServerHosts,ProxyPass:h.ProxyPass,TTL:h.TTL,LogAble:h.LogAble}
}

