package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"sync"
	"time"
)

const configFile = "config.json"

func main() {
	config, err := LoadConfig(configFile)
	if err != nil {
		panic("Failed to load config:" + err.Error())
	}

	healthCheckInterval, err := time.ParseDuration(config.HealthCheckInterval)
	if err != nil {
		fmt.Printf("Failed to parse health check interval:" + err.Error())
		return
	}

	servers := initServers(config.ServersURLs, healthCheckInterval)

	lb := &LoadBalancer{CurrentIndex: 0}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		server := lb.GetServer(servers)
		if server == nil {
			http.Error(w, "No healthy server found", http.StatusServiceUnavailable)
			return
		}

		w.Header().Add("X-Forwarded-Server", server.URL.String())
		server.ReverseProxy().ServeHTTP(w, r)
	})

	fmt.Printf("Load balancer is running on port %s\n", config.Port)
	err = http.ListenAndServe(config.Port, nil)
	if err != nil {
		fmt.Printf("Failed to start load balancer:" + err.Error())
	}
}

type LoadBalancer struct {
	CurrentIndex int
	mu   sync.Mutex
}

func (lb *LoadBalancer) GetServer(servers []*Server) *Server {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	// Round-robin load balancing
	for i := 0; i < len(servers); i++ {
		idx := lb.CurrentIndex % len(servers)
		server := servers[idx]
		lb.CurrentIndex++
		server.mu.Lock()
		isHealthy := server.HealthStatus
		server.mu.Unlock()
		if isHealthy {
			return server
		}
	}

	return nil
}

func HealthCheck(server *Server, interval time.Duration) {
	for range time.Tick(interval) {
		// Do health check
		resp, err := http.Head(server.URL.String())
		server.mu.Lock()
		if err != nil || resp.StatusCode != http.StatusOK {
			server.HealthStatus = false
		} else {
			server.HealthStatus = true
		}
		server.mu.Unlock()
	}
}

type Server struct {
	URL          *url.URL // URL of the server
	HealthStatus bool     // Whether the server is healthy or not
	mu           sync.Mutex
}

func (s *Server) ReverseProxy() *httputil.ReverseProxy {
	return httputil.NewSingleHostReverseProxy(s.URL)
}

type Config struct {
	ServersURLs         []string "json:servers_urls"
	HealthCheckInterval string   "json:health_check_interval"
	Port                string   "json:port"
}

func LoadConfig(filename string) (*Config, error) {
	// Load config from file
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = json.NewDecoder(bytes.NewReader(data)).Decode(&config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func initServers(serversURLs []string, healthCheckInterval time.Duration) []*Server {
	servers := make([]*Server, 0, len(serversURLs))
	for _, serverURL := range serversURLs {
		u, _ := url.Parse(serverURL)
		server := &Server{
			URL:          u,
			HealthStatus: true,
		}
		servers = append(servers, server)
		go HealthCheck(server, healthCheckInterval)
	}

	return servers
}
