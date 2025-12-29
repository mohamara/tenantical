package models

import "time"

type Tenant struct {
	Domain      string    `json:"domain"`
	TenantID    string    `json:"tenant_id"`
	ProjectRoute string   `json:"project_route"` // مثال: /projects/backend
	ProjectPort  *int     `json:"project_port,omitempty"` // Optional port for project
	CreatedAt   time.Time `json:"created_at"`
}

type TenantConfig struct {
	Domain      string
	TenantID    string
	ProjectRoute string
	ProjectPort  *int
}

type TenantInfo struct {
	TenantID     string
	ProjectRoute string
	ProjectPort  *int    // Optional port, nil means use default from config
	BackendDomain *string // Optional backend domain (e.g., localhost, admin.local), nil means use default from BACKEND_URL
}

