package main

import (
	"flag"
	"log"
	"os"

	"github.com/tenantical/router/internal/database"
)

func main() {
	dbPath := flag.String("db", "./tenants.db", "Path to SQLite database file")
	flag.Parse()

	tm, err := database.NewTenantManager(*dbPath, false)
	if err != nil {
		log.Fatalf("Failed to initialize tenant manager: %v", err)
	}
	defer tm.Close()

	// Example tenants
	tenants := []struct {
		domain   string
		tenantID string
	}{
		{"tenant1.example.com", "tenant-123"},
		{"tenant2.example.com", "tenant-456"},
		{"*.saas.com", "tenant-789"},
	}

	log.Printf("Initializing database at %s", *dbPath)

	for _, tenant := range tenants {
		if err := tm.AddTenant(tenant.domain, tenant.tenantID); err != nil {
			log.Printf("Failed to add tenant %s: %v", tenant.domain, err)
			os.Exit(1)
		}
		log.Printf("Added tenant: %s -> %s", tenant.domain, tenant.tenantID)
	}

	log.Println("Database initialized successfully!")
}

