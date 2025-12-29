package database

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/sync/singleflight"
)

type TenantInfo struct {
	TenantID    string
	ProjectRoute string
	ProjectPort  *int // Optional port, nil means use default from config
}

type TenantManager struct {
	db          *sql.DB
	sf          *singleflight.Group
	cache       map[string]TenantInfo
	cacheMutex  sync.RWMutex
	cacheEnabled bool
}

func NewTenantManager(dbPath string, enableCache bool) (*TenantManager, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	tm := &TenantManager{
		db:           db,
		sf:           &singleflight.Group{},
		cache:        make(map[string]TenantInfo),
		cacheEnabled: enableCache,
	}

	if err := tm.initDB(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	return tm, nil
}

func (tm *TenantManager) initDB() error {
	createTable := `
	CREATE TABLE IF NOT EXISTS tenants (
		domain TEXT PRIMARY KEY,
		tenant_id TEXT NOT NULL,
		project_route TEXT NOT NULL DEFAULT '/projects/backend',
		project_port INTEGER,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_domain ON tenants(domain);
	CREATE INDEX IF NOT EXISTS idx_tenant_id ON tenants(tenant_id);
	`
	_, err := tm.db.Exec(createTable)
	
	// Migration: Add project_route column if it doesn't exist (for existing databases)
	_, _ = tm.db.Exec("ALTER TABLE tenants ADD COLUMN project_route TEXT DEFAULT '/projects/backend'")
	
	// Migration: Add project_port column if it doesn't exist (for existing databases)
	_, _ = tm.db.Exec("ALTER TABLE tenants ADD COLUMN project_port INTEGER")
	
	return err
}

func (tm *TenantManager) Close() error {
	return tm.db.Close()
}

// GetTenantInfo returns both tenant ID and project route for a given host
func (tm *TenantManager) GetTenantInfo(host string) (*TenantInfo, error) {
	// Normalize host (remove port if present)
	host = strings.ToLower(strings.Split(host, ":")[0])

	// Check cache first
	if tm.cacheEnabled {
		tm.cacheMutex.RLock()
		if info, found := tm.cache[host]; found {
			tm.cacheMutex.RUnlock()
			return &info, nil
		}
		tm.cacheMutex.RUnlock()
	}

	// Use singleflight to prevent thundering herd
	result, err, _ := tm.sf.Do(host, func() (interface{}, error) {
		return tm.resolveTenantInfo(host)
	})

	if err != nil {
		return nil, err
	}

	info := result.(*TenantInfo)

	// Update cache
	if tm.cacheEnabled && info.TenantID != "" {
		tm.cacheMutex.Lock()
		tm.cache[host] = *info
		tm.cacheMutex.Unlock()
	}

	return info, nil
}

// GetTenantID is kept for backward compatibility
func (tm *TenantManager) GetTenantID(host string) (string, error) {
	info, err := tm.GetTenantInfo(host)
	if err != nil {
		return "", err
	}
	return info.TenantID, nil
}

func (tm *TenantManager) resolveTenantInfo(host string) (*TenantInfo, error) {
	// Direct match
	var tenantID, projectRoute string
	var projectPort sql.NullInt64
	err := tm.db.QueryRow(
		"SELECT tenant_id, project_route, project_port FROM tenants WHERE domain = ?",
		host,
	).Scan(&tenantID, &projectRoute, &projectPort)

	if err == nil {
		// Set default if empty
		if projectRoute == "" {
			projectRoute = "/projects/backend"
		}
		
		var port *int
		if projectPort.Valid {
			p := int(projectPort.Int64)
			port = &p
		}
		
		return &TenantInfo{
			TenantID:    tenantID,
			ProjectRoute: projectRoute,
			ProjectPort:  port,
		}, nil
	}

	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("database error: %w", err)
	}

	// Wildcard match (e.g., *.example.com)
	rows, err := tm.db.Query("SELECT domain, tenant_id, project_route, project_port FROM tenants WHERE domain LIKE '%*%'")
	if err != nil {
		return nil, fmt.Errorf("wildcard query error: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var domain, tid, route string
		var projectPort sql.NullInt64
		if err := rows.Scan(&domain, &tid, &route, &projectPort); err != nil {
			continue
		}

		// Convert wildcard pattern to match
		if tm.matchWildcard(host, domain) {
			if route == "" {
				route = "/projects/backend"
			}
			
			var port *int
			if projectPort.Valid {
				p := int(projectPort.Int64)
				port = &p
			}
			
			return &TenantInfo{
				TenantID:    tid,
				ProjectRoute: route,
				ProjectPort:  port,
			}, nil
		}
	}

	return nil, fmt.Errorf("tenant not found for domain: %s", host)
}

func (tm *TenantManager) matchWildcard(host, pattern string) bool {
	pattern = strings.ToLower(pattern)

	if !strings.Contains(pattern, "*") {
		return host == pattern
	}

	// Simple wildcard matching: *.example.com matches subdomain.example.com
	if strings.HasPrefix(pattern, "*.") {
		suffix := pattern[2:] // Remove "*."
		return strings.HasSuffix(host, suffix) && strings.Count(host, ".") >= strings.Count(suffix, ".")+1
	}

	if strings.HasSuffix(pattern, ".*") {
		prefix := pattern[:len(pattern)-2] // Remove ".*"
		return strings.HasPrefix(host, prefix) && strings.Count(host, ".") >= strings.Count(prefix, ".")+1
	}

	// Full pattern matching (more complex cases)
	parts := strings.Split(pattern, "*")
	if len(parts) != 2 {
		return false
	}

	return strings.HasPrefix(host, parts[0]) && strings.HasSuffix(host, parts[1])
}

func (tm *TenantManager) AddTenant(domain, tenantID, projectRoute string, projectPort *int) error {
	domain = strings.ToLower(domain)
	
	// Set default if empty
	if projectRoute == "" {
		projectRoute = "/projects/backend"
	}
	
	var portValue interface{}
	if projectPort != nil {
		portValue = *projectPort
	} else {
		portValue = nil
	}
	
	_, err := tm.db.Exec(
		"INSERT OR REPLACE INTO tenants (domain, tenant_id, project_route, project_port) VALUES (?, ?, ?, ?)",
		domain, tenantID, projectRoute, portValue,
	)
	
	if err != nil {
		return fmt.Errorf("failed to add tenant: %w", err)
	}

	// Invalidate cache
	if tm.cacheEnabled {
		tm.cacheMutex.Lock()
		delete(tm.cache, domain)
		tm.cacheMutex.Unlock()
	}

	return nil
}

func (tm *TenantManager) DeleteTenant(domain string) error {
	domain = strings.ToLower(domain)
	
	_, err := tm.db.Exec("DELETE FROM tenants WHERE domain = ?", domain)
	
	if err != nil {
		return fmt.Errorf("failed to delete tenant: %w", err)
	}

	// Invalidate cache
	if tm.cacheEnabled {
		tm.cacheMutex.Lock()
		delete(tm.cache, domain)
		tm.cacheMutex.Unlock()
	}

	return nil
}

func (tm *TenantManager) ListTenants() ([]map[string]interface{}, error) {
	rows, err := tm.db.Query("SELECT domain, tenant_id, project_route, project_port, created_at FROM tenants ORDER BY domain")
	if err != nil {
		return nil, fmt.Errorf("failed to list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []map[string]interface{}
	for rows.Next() {
		var domain, tenantID, projectRoute string
		var projectPort sql.NullInt64
		var createdAt string
		
		if err := rows.Scan(&domain, &tenantID, &projectRoute, &projectPort, &createdAt); err != nil {
			continue
		}

		if projectRoute == "" {
			projectRoute = "/projects/backend"
		}

		tenant := map[string]interface{}{
			"domain":       domain,
			"tenant_id":    tenantID,
			"project_route": projectRoute,
			"created_at":   createdAt,
		}
		
		if projectPort.Valid {
			tenant["project_port"] = projectPort.Int64
		}

		tenants = append(tenants, tenant)
	}

	return tenants, nil
}

func (tm *TenantManager) ClearCache() {
	if tm.cacheEnabled {
		tm.cacheMutex.Lock()
		tm.cache = make(map[string]TenantInfo)
		tm.cacheMutex.Unlock()
	}
}

