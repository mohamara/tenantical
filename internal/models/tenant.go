package models

import "time"

type Tenant struct {
	Domain      string    `json:"domain"`
	TenantID    string    `json:"tenant_id"`
	ProjectRoute string   `json:"project_route"` // مثال: /projects/backend
	CreatedAt   time.Time `json:"created_at"`
}

type TenantConfig struct {
	Domain      string
	TenantID    string
	ProjectRoute string
}

type TenantInfo struct {
	TenantID    string
	ProjectRoute string
}

