// golang-reverse-proxy implements a reverse-proxy-like server that accepts a set of
// one or more backend servers to proxy requests to
package main

import (
	// Standard lib
	"flag"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"

	// Third-party
	log "github.com/Sirupsen/logrus"
)

const (
	// Default set of backend servers to use for proxying requests
	DEFAULT_BACKENDS = "http://127.0.0.1:6060,http://127.0.0.1:6061,http://127.0.0.1:6062"
	// Default port to use for the server
	DEFAULT_PORT = "8080"
)

type (
	// Struct representing a single backend server to proxy requests to
	BackendServer struct {
		Proxy *httputil.ReverseProxy
		Url   *url.URL
	}
)

var (
	// Port server will listen on
	port *string
	// Comma-separated string of backend servers requests should be sent to
	backends *string
	// Slice of zero or more backend server structs
	backendServers []*BackendServer
)

// handle is the main HTTP handler function for all requests to the server
func handle(w http.ResponseWriter, req *http.Request) {
	// Get backend server, checking for errors
	backendServer, err := getBackendServer()
	if err != nil {
		fmt.Fprint(w, err.Error())
		return
	}

	// Log request handling
	log.Infof("Proxying request for '%s' to backend server with address: %s", req.URL.String(), backendServer.Url.String())

	// Use backend server to serve the request
	backendServer.Proxy.ServeHTTP(w, req)
}

// returns a random backend server when possible, an error when not
func getBackendServer() (*BackendServer, error) {
	// Check for at least one backend server
	if len(backendServers) == 0 {
		return nil, fmt.Errorf("No backend servers available :(")
	}

	// TO-DO: Support for marking servers as "down"
	// TO-DO: Support for choosing a server based on number of concurrent requests to it

	// Return semi-random backend server
	return backendServers[rand.Intn(len(backendServers))], nil
}

// parses and configures all available backend servers
func parseBackends() {
	// Split up backends
	splitBackends := strings.Split(*backends, ",")

	// Loop through backends, creating a new proxy for each
	for _, backend := range splitBackends {
		// Remove leading and trailing spaces
		backend = strings.Trim(backend, " ")

		// TO-DO: Handle scheme checking

		// Parse backend address and check validity
		backendUrl, err := url.Parse(backend)
		if err != nil || len(backend) == 0 {
			continue
		}

		// Create new backend server
		backendServer := &BackendServer{
			// NOTE: `NewSingleHostReverseProxy` requires a scheme for backend URLs
			Proxy: httputil.NewSingleHostReverseProxy(backendUrl),
			Url:   backendUrl,
		}

		// Add backend to slice
		backendServers = append(backendServers, backendServer)
	}

	// Log backends
	log.Infof("Parsed %d backend servers", len(backendServers))
}

// configures and starts up an HTTP server on a desired port
func startServer() {
	// Create new mux instance
	mux := http.NewServeMux()

	// Create new server instance
	server := &http.Server{}

	// Set up server
	server.Addr = ":" + *port
	server.Handler = mux
	server.ReadTimeout = time.Duration(30) * time.Second
	server.WriteTimeout = time.Duration(30) * time.Second

	// Set up server routes
	mux.Handle("/", http.HandlerFunc(handle))

	// Log server start
	log.Infof("Server running on port %s", *port)

	// Attempt to start the server
	go server.ListenAndServe()
}

func main() {
	// Log start of the service
	log.Info("Server is starting")

	// Get port and backend servers from flags with fallback
	port = flag.String("port", DEFAULT_PORT, "default server port, ex: 8080")
	backends = flag.String("backends", DEFAULT_BACKENDS, "comma-separated list of backend servers, ex: localhost:6060,localhost:6061")

	// Parse flags
	flag.Parse()

	// Parse backend servers
	parseBackends()

	// Start server
	startServer()

	// Listen for and exit the application on SIGKILL or SIGINT
	stop := make(chan os.Signal)
	signal.Notify(stop, os.Interrupt, os.Kill)

	select {
	case <-stop:
		log.Info("Server is shutting down")
	}
}
