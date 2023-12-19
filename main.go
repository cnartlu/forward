package main

import (
	"context"
	"flag"
	"io"
	"log"
	"net"
	"strings"
)

type Config struct {
	rules    StringsFlag
	orgs     StringsFlag
	domains  StringsFlag
	remoteDB string
}

type CheckNetConnHandle func(conn net.Conn) bool

func DefaultCheckNetConnHandle(conf Config, db *mmdb) CheckNetConnHandle {
	return func(conn net.Conn) bool {
		addr := conn.RemoteAddr()
		ip := addr.(*net.TCPAddr).IP
		if ip.IsPrivate() || ip.IsLoopback() {
			return true
		}
		result := AsnIpinfo{}
		if err := db.Reader().Lookup(ip, &result); err != nil {
			return false
		}
		for _, org := range conf.orgs {
			if strings.EqualFold(org, result.Name) {
				return true
			}
		}
		for _, domain := range conf.domains {
			if strings.EqualFold(domain, result.Domain) {
				return true
			}
		}
		return false
	}
}

func tcpForward(l net.Listener, remoteAddr string, fn CheckNetConnHandle) {
	for {
		s_conn, err := l.Accept()
		if err != nil {
			continue
		}
		if fn != nil && !fn(s_conn) {
			s_conn.Close()
			continue
		}
		d_tcpAddr, _ := net.ResolveTCPAddr("tcp", remoteAddr)
		d_conn, err := net.DialTCP("tcp", nil, d_tcpAddr)
		if err != nil {
			s_conn.Close()
			continue
		}
		go io.Copy(s_conn, d_conn)
		go io.Copy(d_conn, s_conn)
	}
}

func main() {
	conf := Config{}
	flag.Var(&conf.rules, "r", "Forwarding rule string [source addr]/[destination addr]")
	flag.Var(&conf.orgs, "o", "The organization to which the IP belongs")
	flag.Var(&conf.domains, "d", "The domains to which the IP belongs")
	flag.StringVar(&conf.remoteDB, "remote-db", "https://github.com/cnartlu/geoip2/releases/download/V2023121819/asn.mmdb", "geoip2 database remote address")
	flag.Parse()

	routeAddrs := []string{}
	for _, rule := range conf.rules {
		addrs := strings.SplitN(rule, "/", 2)
		if len(addrs) != 2 {
			panic(`"` + rule + `" configuration error, missing source or destination addr`)
		}
		for _, addr := range addrs {
			addr := strings.TrimSpace(addr)
			_, err := net.ResolveTCPAddr("tcp", addr)
			if err != nil {
				log.Fatalln("resolve tcp addr", addr, err)
			}
			routeAddrs = append(routeAddrs, addr)
		}
	}

	db := InitMMDB(conf.remoteDB, "asn.mmdb")
	checkHandle := DefaultCheckNetConnHandle(conf, db)
	c := make(chan struct{})
	for i := 0; i < len(routeAddrs); i += 2 {
		local := routeAddrs[i]
		remote := routeAddrs[i+1]
		go func(localAddr, remoteAddr string) {
			lc := net.ListenConfig{}
			l, err := lc.Listen(context.Background(), "tcp", localAddr)
			if err != nil {
				log.Default().Fatalf("%s listen: %s\n", localAddr, err)
			}
			defer l.Close()
			tcpForward(l, remoteAddr, checkHandle)
		}(local, remote)
	}
	<-c
}
