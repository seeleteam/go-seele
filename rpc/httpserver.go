/**
*  @file
*  @copyright defined in go-seele/LICENSE
 */

package rpc

import (
	"errors"
	"io"
	"net"
	"net/http"
	"net/rpc"
	"strings"

	"github.com/rs/cors"
)

var (
	// ErrInvalidHost will be returned when the host is not in the whitelist
	ErrInvalidHost = errors.New("invalid host name")
)

// HTTPServer represents a HTTP RPC server
type HTTPServer struct {
	rpc *rpc.Server
}

// NewHTTPServer returns a new HttpServer and a http handler used by cors
func NewHTTPServer(whitehosts []string, corsList []string) (*HTTPServer, *hostFilter) {
	server := &HTTPServer{
		rpc: &rpc.Server{},
	}
	// cors
	c := cors.New(cors.Options{
		AllowedOrigins: corsList,
		AllowedMethods: []string{http.MethodPost, http.MethodConnect},
		AllowedHeaders: []string{"*"},
		MaxAge:         600,
	})

	// whitelist
	wMap := make(map[string]struct{})
	for _, whitehost := range whitehosts {
		wMap[strings.ToLower(whitehost)] = struct{}{}
	}
	hFilter := hostFilter{wMap, c.Handler(server)}

	return server, &hFilter
}

// ServeHTTP implements an http.Handler that answers RPC requests.
// Supports POST and CONNECT http method.
// POST handles requests from the browser
// CONNECT handles requests form other go rpc.Client
func (server *HTTPServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case http.MethodConnect:
		server.rpc.ServeHTTP(w, req)
	case http.MethodPost:
		w.Header().Set("Content-Type", "application/json")
		conn := &httpReadWriteCloser{req.Body, w}
		server.rpc.ServeRequest(NewJSONCodec(conn, server.rpc))
	default:
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusMethodNotAllowed)
		io.WriteString(w, "405 must POST or CONNECT\n")
	}
}

// GetRPCServer return rpc server of the HTTPServer
func (server *HTTPServer) GetRPCServer() *rpc.Server {
	return server.rpc
}

// httpReadWriteCloser wraps a io.Reader and io.Writer
type httpReadWriteCloser struct {
	io.Reader
	io.Writer
}

func (t *httpReadWriteCloser) Close() error {
	return nil
}

// hostFilter handlers the incoming requests and filters the Host-header.
// To prevent DNS rebinding attacks which do not utilize CORS-headers.
// We use a whitelist to validate the Host-header in domains.
type hostFilter struct {
	whitehosts map[string]struct{}
	handler    http.Handler
}

// ServeHTTP handlers the incoming requests and validate the Host-header
func (h *hostFilter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.isValideHost(r) {
		h.handler.ServeHTTP(w, r)
	} else {
		http.Error(w, ErrInvalidHost.Error(), http.StatusForbidden)
	}
}

func (h *hostFilter) isValideHost(r *http.Request) bool {
	if r.Host == "" {
		return true
	}

	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		// no port or too many colons is ok
		// we just filter the whitelist
		host = r.Host
	}

	//ip address is ok
	if ip := net.ParseIP(host); ip != nil {
		return true
	}

	// * and nil whitehost do not need to validate
	_, exist := h.whitehosts["*"]
	if exist || len(h.whitehosts) == 0 {
		return true
	}

	if _, exist := h.whitehosts[host]; exist {
		return true
	}

	return false
}
