package main

import (
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/as/log"
)

var (
	dir  = flag.String("-d", ".", "directory to serve")
	addr = flag.String("-a", ":8086", "address to listen on")
)

func main() {
	flag.Parse()
	log.Service = "httpd"
	log.Info.Add("addr", *addr, "dir", *dir).Printf("init")
	if err := http.ListenAndServe(*addr, Log{http.FileServer(http.Dir(*dir))}); err != nil {
		log.Error.F("%v", err)
	}
}

type Log struct {
	http.Handler
}

func (l Log) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ip, port := parseIP(r)
	w.Header().Set("X-RID", fmt.Sprint(rand.Uint64()))
	r.URL.Host = r.Host
	r.URL.Scheme = r.Header.Get("X-Forwarded-Proto")
	if r.URL.Scheme == "" {
		r.URL.Scheme = "http"
	}
	start := time.Now()

	l.Handler.ServeHTTP(w, r)

	log.Info.Add(
		"rid", w.Header().Get("X-RID"),
		"method", r.Method,
		"path", r.URL.Path,
		"size", r.Header.Get("Content-Length"),
		"duration", time.Since(start).Seconds(),
		"ip", ip[0],
		"ip_cdn", ip[1],
		"ip_forwarded", r.Header.Get("X-Forwarded-For"),
		"port", port,
		"raddr", r.RemoteAddr,
		"referer", r.Referer(),
		"ua", r.UserAgent(),
		"url", r.URL.String(),
	).Printf("served")
}
func parseIP(r *http.Request) (ip [2]string, port string) {
	copy(ip[:], strings.Split(r.Header.Get("X-Forwarded-For"), ","))
	for i := range ip[:] {
		ip[i] = strings.TrimSpace(ip[i])
	}
	if ip[0] == "" {
		ip[0], _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	port = r.Header.Get("X-Forwarded-Port")
	if port == "" {
		_, port, _ = net.SplitHostPort(r.RemoteAddr)
	}
	return ip, port
}
