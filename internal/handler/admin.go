package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/tenantical/router/internal/database"
)

type AdminHandler struct {
	tenantManager *database.TenantManager
}

func NewAdminHandler(tm *database.TenantManager) *AdminHandler {
	return &AdminHandler{
		tenantManager: tm,
	}
}

func (h *AdminHandler) AddTenant(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domain      string `json:"domain"`
		TenantID    string `json:"tenant_id"`
		ProjectRoute string `json:"project_route"` // Optional, defaults to /projects/backend
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Domain == "" || req.TenantID == "" {
		http.Error(w, "domain and tenant_id are required", http.StatusBadRequest)
		return
	}

	if err := h.tenantManager.AddTenant(req.Domain, req.TenantID, req.ProjectRoute); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":      "Tenant added successfully",
		"domain":       req.Domain,
		"tenant_id":    req.TenantID,
		"project_route": req.ProjectRoute,
	})
}

func (h *AdminHandler) DeleteTenant(w http.ResponseWriter, r *http.Request) {
	domain := chi.URLParam(r, "domain")
	
	if domain == "" {
		http.Error(w, "domain parameter is required", http.StatusBadRequest)
		return
	}

	if err := h.tenantManager.DeleteTenant(domain); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Tenant deleted successfully",
		"domain":  domain,
	})
}

func (h *AdminHandler) ListTenants(w http.ResponseWriter, r *http.Request) {
	tenants, err := h.tenantManager.ListTenants()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"tenants": tenants,
		"count":   len(tenants),
	})
}

func (h *AdminHandler) RegisterRoutes(r chi.Router) {
	r.Route("/admin/tenants", func(r chi.Router) {
		r.Post("/", h.AddTenant)
		r.Delete("/{domain}", h.DeleteTenant)
		r.Get("/", h.ListTenants)
	})
}

