package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
)

type Endpoint interface {
	// URI provides the URI to access the Endpoint
	URI() string

	// Available checks if the Endpoint is ready to handle requests
	Available() bool

	// HandleRequest processes the incoming HTTP request
	HandleRequest(responseWriter http.ResponseWriter, request *http.Request)
}

type basicEndpoint struct {
	uri   string
	proxy *httputil.ReverseProxy
}

func (be *basicEndpoint) URI() string { return be.uri }

func (be *basicEndpoint) Available() bool { return true }

func (be *basicEndpoint) HandleRequest(responseWriter http.ResponseWriter, request *http.Request) {
	be.proxy.ServeHTTP(responseWriter, request)
}

func newBasicEndpoint(uri string) *basicEndpoint {
	endpointURL, err := url.Parse(uri)
	handleError(err)

	return &basicEndpoint{
		uri:   uri,
		proxy: httputil.NewSingleHostReverseProxy(endpointURL),
	}
}

type TrafficManager struct {
	listeningPort     string
	sequentialCounter int
	endpoints         []Endpoint
}

func NewTrafficManager(listeningPort string, endpoints []Endpoint) *TrafficManager {
	return &TrafficManager{
		listeningPort:     listeningPort,
		sequentialCounter: 0,
		endpoints:         endpoints,
	}
}

// handleError logs the error and exits the program
// This is just a simple demonstration error handler and should not be used in production.
func handleError(err error) {
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

// selectNextEndpoint provides the URI of the next active endpoint using a round-robin method
func (tm *TrafficManager) selectNextEndpoint() Endpoint {
	endpoint := tm.endpoints[tm.sequentialCounter%len(tm.endpoints)]
	for !endpoint.Available() {
		tm.sequentialCounter++
		endpoint = tm.endpoints[tm.sequentialCounter%len(tm.endpoints)]
	}
	tm.sequentialCounter++

	return endpoint
}

func (tm *TrafficManager) proxyRequest(responseWriter http.ResponseWriter, request *http.Request) {
	selectedEndpoint := tm.selectNextEndpoint()
	fmt.Printf("routing request to URI %q\n", selectedEndpoint.URI())
	selectedEndpoint.HandleRequest(responseWriter, request)
}

func main() {
	endpoints := []Endpoint{
		newBasicEndpoint("https://www.fampay.com"),
		newBasicEndpoint("https://www.stackoverflow.com"),
		newBasicEndpoint("https://www.github.com"),
	}

	tm := NewTrafficManager("8000", endpoints)
	redirectHandler := func(responseWriter http.ResponseWriter, request *http.Request) {
		tm.proxyRequest(responseWriter, request)
	}

	// set up the HTTP handler for proxy requests
	http.HandleFunc("/", redirectHandler)

	fmt.Printf("handling requests on 'localhost:%s'\n", tm.listeningPort)
	http.ListenAndServe(":"+tm.listeningPort, nil)
}
