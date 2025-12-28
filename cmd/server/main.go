package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/tenantical/router/internal/config"
	"github.com/tenantical/router/internal/database"
	"github.com/tenantical/router/internal/handler"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize tenant manager
	tm, err := database.NewTenantManager(cfg.Database.Path, true)
	if err != nil {
		log.Fatalf("Failed to initialize tenant manager: %v", err)
	}
	defer tm.Close()

	// Initialize handlers
	proxyHandler := handler.NewProxyHandler(
		tm,
		cfg.Proxy.BackendURL,
		cfg.Proxy.Timeout,
		cfg.Proxy.MaxIdleConns,
		cfg.Proxy.IdleConnTimeout,
		cfg.Proxy.DisableKeepAlive,
	)

	adminHandler := handler.NewAdminHandler(tm)
	adminUIHandler := handler.NewAdminUIHandler()

	// Setup router
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"*"},
		ExposedHeaders:   []string{"*"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Middleware to handle admin domain root path
	adminDomain := cfg.Server.AdminDomain
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Only handle root path "/"
			if r.URL.Path == "/" {
				host := r.Host
				if host == "" {
					host = r.Header.Get("X-Forwarded-Host")
				}
				if host == "" {
					host = r.Header.Get("X-Original-Host")
				}
				
				// If request is from admin domain, redirect to /admin
				if host == adminDomain {
					http.Redirect(w, r, "/admin", http.StatusFound)
					return
				}
			}
			
			// Continue to next handler
			next.ServeHTTP(w, r)
		})
	})

	// Health check endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","service":"tenant-router"}`)
	})

	// Admin UI route
	adminUIHandler.RegisterRoutes(r)

	// Admin API routes (optional, can be protected with auth)
	adminHandler.RegisterRoutes(r)

	// Proxy routes (catch-all)
	proxyHandler.RegisterRoutes(r)

	// Setup HTTP server
	srv := &http.Server{
		Addr:         cfg.ServerAddress(),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting tenant router server on %s", cfg.ServerAddress())
		log.Printf("Backend URL: %s", cfg.Proxy.BackendURL)
		log.Printf("Database path: %s", cfg.Database.Path)
		
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}

