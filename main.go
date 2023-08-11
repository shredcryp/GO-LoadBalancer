package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Server interface {
	Address() string
	IsAlive() bool
	Serve(rw http.ResponseWriter, r *http.Request)
}

type simpleServer struct {
	Addr  string `json:"Addr"`
	proxy *httputil.ReverseProxy
}

func handleErr(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

type LoadBalancer struct {
	port            string
	roundRobinCount int
	servers         []Server
}

func NewLoadBalancer(port string, servers []Server) *LoadBalancer {
	return &LoadBalancer{
		port:            port,
		roundRobinCount: 0,
		servers:         servers,
	}
}

func newSimpleServer(Addr string) *simpleServer {
	serverUrl, err := url.Parse(Addr)
	handleErr(err)

	return &simpleServer{
		Addr:  Addr,
		proxy: httputil.NewSingleHostReverseProxy(serverUrl),
	}
}

func (s *simpleServer) IsAlive() bool { return true }

func (s *simpleServer) Address() string { return s.Addr }

func (s *simpleServer) Serve(rw http.ResponseWriter, req *http.Request) {
	s.proxy.ServeHTTP(rw, req)
}

func (lb *LoadBalancer) getNextAvailableServer() Server {
	initialCount := lb.roundRobinCount
	for {
		server := lb.servers[lb.roundRobinCount%len(lb.servers)]
		if server.IsAlive() {
			lb.roundRobinCount++
			return server
		}
		lb.roundRobinCount++
		if lb.roundRobinCount%len(lb.servers) == initialCount {
			break
		}
	}
	return nil
}

func (lb *LoadBalancer) serveProxy(rw http.ResponseWriter, req *http.Request) {
	targetServer := lb.getNextAvailableServer()
	fmt.Printf("Forwarding request to address %q", targetServer.Address())
	targetServer.Serve(rw, req)
}

func main() {
	servers := []Server{
		newSimpleServer("https://www.facebook.com"),
		newSimpleServer("http://www.bing.com"),
		newSimpleServer("http://www.duckduckgo.com"),
	}
	lb := NewLoadBalancer("8000", servers)
	handleRedirect := func(rw http.ResponseWriter, req *http.Request) {
		lb.serveProxy(rw, req)
	}
	http.HandleFunc("/", handleRedirect)

	fmt.Printf("Server is serving reqs at Localhost %s", lb.port)
	http.ListenAndServe(":"+lb.port, nil)
}
