package handler

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/tenantical/router/internal/database"
)

type ProxyHandler struct {
	tenantManager *database.TenantManager
	backendURL    string
	client        *http.Client
}

func NewProxyHandler(tm *database.TenantManager, backendURL string, timeout time.Duration, maxIdleConns int, idleConnTimeout time.Duration, disableKeepAlive bool) *ProxyHandler {
	transport := &http.Transport{
		MaxIdleConns:        maxIdleConns,
		IdleConnTimeout:     idleConnTimeout,
		DisableKeepAlives:   disableKeepAlive,
		MaxIdleConnsPerHost: 10,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	return &ProxyHandler{
		tenantManager: tm,
		backendURL:    backendURL,
		client:        client,
	}
}

func (h *ProxyHandler) Handle(w http.ResponseWriter, r *http.Request) {
	log.Printf("[PROXY] Processing request: %s %s", r.Method, r.URL.Path)

	// Extract host from request
	host := r.Host
	if host == "" {
		host = r.Header.Get("X-Forwarded-Host")
	}
	if host == "" {
		host = r.Header.Get("X-Original-Host")
	}

	log.Printf("[PROXY] Host identified: %s", host)

	if host == "" {
		log.Printf("[PROXY] ERROR: Missing Host header")
		http.Error(w, "Missing Host header", http.StatusBadRequest)
		return
	}

	// Resolve tenant info (ID + project route)
	tenantInfo, err := h.tenantManager.GetTenantInfo(host)
	if err != nil {
		log.Printf("[PROXY] ERROR: Tenant not found for domain: %s (error: %v)", host, err)
		http.Error(w, "Invalid tenant domain", http.StatusNotFound)
		return
	}

	log.Printf("[PROXY] Tenant found - ID: %s, Route: %s, Port: %v, BackendDomain: %v",
		tenantInfo.TenantID, tenantInfo.ProjectRoute,
		tenantInfo.ProjectPort, tenantInfo.BackendDomain)

	// Build backend URL
	baseURL, err := parseBackendURL(h.backendURL)
	if err != nil {
		log.Printf("[PROXY] ERROR: Invalid backend URL configuration: %s", h.backendURL)
		http.Error(w, "Invalid backend URL configuration", http.StatusInternalServerError)
		return
	}

	log.Printf("[PROXY] Backend URL parsed: %s", baseURL.String())

	// Determine scheme (default to http)
	scheme := baseURL.Scheme
	if scheme == "" {
		scheme = "http"
	}

	var backendURL *url.URL

	// Override domain if tenant has a specific backend domain
	if tenantInfo.BackendDomain != nil && *tenantInfo.BackendDomain != "" {
		// Use tenant-specific backend domain
		hostname := *tenantInfo.BackendDomain
		
		// Convert localhost/127.0.0.1 to host.docker.internal when in Docker (for accessing host services)
		// This allows containers to reach services on the Docker host
		if hostname == "localhost" || hostname == "127.0.0.1" {
			// Check if we should use host.docker.internal (default behavior in Docker)
			// User can override by setting DOCKER_HOST_ALIAS env var (e.g., to use container name)
			dockerHost := os.Getenv("DOCKER_HOST_ALIAS")
			if dockerHost == "" {
				dockerHost = "host.docker.internal" // Default for Docker Desktop and newer Docker
			}
			hostname = dockerHost
		} else if strings.HasSuffix(hostname, ".localhost") {
			// For .localhost domains, convert to host.docker.internal (or custom alias)
			// This allows admin.localhost -> host.docker.internal
			dockerHost := os.Getenv("DOCKER_HOST_ALIAS")
			if dockerHost == "" {
				dockerHost = "host.docker.internal"
			}
			hostname = dockerHost
		}
		
		port := baseURL.Port()
		if tenantInfo.ProjectPort != nil {
			port = strconv.Itoa(*tenantInfo.ProjectPort)
		}
		
		// Build new URL from scratch
		backendURL = &url.URL{
			Scheme: scheme,
			Host:   hostname,
		}
		if port != "" {
			backendURL.Host = hostname + ":" + port
		}
	} else if tenantInfo.ProjectPort != nil {
		// Override port if tenant has a specific project port (but keep original domain)
		backendURL = &url.URL{
			Scheme: scheme,
			Host:   baseURL.Hostname() + ":" + strconv.Itoa(*tenantInfo.ProjectPort),
		}
	} else {
		backendURL = baseURL
		if backendURL.Scheme == "" {
			backendURL.Scheme = scheme
		}
	}

	// Construct full backend path with project route
	// Format: {backendURL}{projectRoute}{originalPath}
	projectRoute := tenantInfo.ProjectRoute
	if projectRoute == "" {
		projectRoute = "/projects/backend"
	}
	
	// Ensure projectRoute starts with /
	if !strings.HasPrefix(projectRoute, "/") {
		projectRoute = "/" + projectRoute
	}
	
	// Handle root path: if projectRoute is "/", use empty string
	if projectRoute == "/" {
		projectRoute = ""
	} else {
		// Ensure projectRoute ends without / (to avoid double slashes)
		projectRoute = strings.TrimSuffix(projectRoute, "/")
	}
	
	// Construct path: projectRoute + originalPath
	backendPath := projectRoute + r.URL.Path
	
	// Ensure path starts with / for proper URL resolution
	if !strings.HasPrefix(backendPath, "/") {
		backendPath = "/" + backendPath
	}

	// Build the full URL by combining base URL with path and query
	backendReqURL := backendURL.ResolveReference(&url.URL{
		Path:     backendPath,
		RawQuery: r.URL.RawQuery,
	})

	log.Printf("[PROXY] Final backend URL: %s", backendReqURL.String())

	// Create request to backend
	backendReq, err := http.NewRequestWithContext(r.Context(), r.Method, backendReqURL.String(), r.Body)
	if err != nil {
		log.Printf("[PROXY] ERROR: Failed to create backend request: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Copy headers from original request
	for key, values := range r.Header {
		// Skip headers that should not be forwarded
		switch key {
		case "Connection", "Keep-Alive", "Proxy-Authenticate",
			"Proxy-Authorization", "Te", "Trailers", "Transfer-Encoding", "Upgrade":
			continue
		}
		
		for _, value := range values {
			backendReq.Header.Add(key, value)
		}
	}

	// Request goes to backend without any tenant identification headers
	// Tenant identification is done via domain-based routing only

	// Set proper Host header for backend
	// If backend domain was specified, preserve the original domain in Host header for proper routing
	// This is important for nginx virtual hosts that route based on Host header
	// Note: Most web servers (nginx, Apache) don't need port in Host header when using standard ports
	if tenantInfo.BackendDomain != nil && *tenantInfo.BackendDomain != "" {
		// Use the original backend domain (e.g., admin.localhost) in Host header
		// But connect to host.docker.internal:85 for the actual connection
		// Don't include port in Host header - nginx routes based on domain name only
		originalDomain := *tenantInfo.BackendDomain
		backendReq.Host = originalDomain
	} else {
		backendReq.Host = backendURL.Host
	}

	// Forward request
	log.Printf("[PROXY] Forwarding request to backend: %s %s", backendReq.Method, backendReq.URL.String())
	resp, err := h.client.Do(backendReq)
	if err != nil {
		log.Printf("[PROXY] ERROR: Backend request failed: %v", err)
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	log.Printf("[PROXY] Backend response: %d %s", resp.StatusCode, resp.Status)

	// Copy response headers (must be done before WriteHeader)
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Set status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("[PROXY] ERROR: Failed to copy response body: %v", err)
		// Response already started, can't change status
		// Log error in production
		return
	}

	log.Printf("[PROXY] Request completed successfully - forwarded %s %s to backend", r.Method, r.URL.Path)
}

func (h *ProxyHandler) RegisterRoutes(r chi.Router) {
	r.HandleFunc("/*", h.Handle)
}

func parseBackendURL(rawURL string) (*url.URL, error) {
	baseURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	if baseURL.Host != "" {
		return baseURL, nil
	}

	baseURL, err = url.Parse("http://" + rawURL)
	if err != nil {
		return nil, err
	}

	if baseURL.Host == "" {
		return nil, fmt.Errorf("backend URL is missing host")
	}

	return baseURL, nil
}
