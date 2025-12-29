package handler

import (
	"io"
	"net/http"
	"net/url"
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
	// Extract host from request
	host := r.Host
	if host == "" {
		host = r.Header.Get("X-Forwarded-Host")
	}
	if host == "" {
		host = r.Header.Get("X-Original-Host")
	}

	if host == "" {
		http.Error(w, "Missing Host header", http.StatusBadRequest)
		return
	}

	// Resolve tenant info (ID + project route)
	tenantInfo, err := h.tenantManager.GetTenantInfo(host)
	if err != nil {
		http.Error(w, "Invalid tenant domain", http.StatusNotFound)
		return
	}

	// Build backend URL
	backendURL, err := url.Parse(h.backendURL)
	if err != nil {
		http.Error(w, "Invalid backend URL configuration", http.StatusInternalServerError)
		return
	}

	// Override port if tenant has a specific project port
	if tenantInfo.ProjectPort != nil {
		// Reconstruct URL with tenant-specific port
		backendURL.Host = backendURL.Hostname() + ":" + strconv.Itoa(*tenantInfo.ProjectPort)
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
	
	// Ensure projectRoute ends without / (to avoid double slashes)
	projectRoute = strings.TrimSuffix(projectRoute, "/")
	
	// Construct path: projectRoute + originalPath
	backendPath := projectRoute + r.URL.Path
	if r.URL.RawQuery != "" {
		backendPath += "?" + r.URL.RawQuery
	}

	backendReqURL := backendURL.ResolveReference(&url.URL{Path: backendPath})

	// Create request to backend
	backendReq, err := http.NewRequestWithContext(r.Context(), r.Method, backendReqURL.String(), r.Body)
	if err != nil {
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

	// Inject tenant ID header
	backendReq.Header.Set("X-Tenant-ID", tenantInfo.TenantID)
	
	// Set proper Host header for backend
	backendReq.Host = backendURL.Host

	// Forward request
	resp, err := h.client.Do(backendReq)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

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
		// Response already started, can't change status
		// Log error in production
		return
	}
}

func (h *ProxyHandler) RegisterRoutes(r chi.Router) {
	r.HandleFunc("/*", h.Handle)
}

