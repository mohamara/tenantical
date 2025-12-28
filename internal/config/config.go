package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Proxy    ProxyConfig
}

type ServerConfig struct {
	Host         string
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	Path string
}

type ProxyConfig struct {
	BackendURL       string
	Timeout          time.Duration
	MaxIdleConns     int
	IdleConnTimeout  time.Duration
	DisableKeepAlive bool
}

func Load() (*Config, error) {
	port, _ := strconv.Atoi(getEnv("PORT", "8080"))
	readTimeout, _ := strconv.Atoi(getEnv("READ_TIMEOUT", "10"))
	writeTimeout, _ := strconv.Atoi(getEnv("WRITE_TIMEOUT", "10"))
	idleTimeout, _ := strconv.Atoi(getEnv("IDLE_TIMEOUT", "120"))

	proxyTimeout, _ := strconv.Atoi(getEnv("PROXY_TIMEOUT", "30"))
	maxIdleConns, _ := strconv.Atoi(getEnv("PROXY_MAX_IDLE_CONNS", "100"))
	idleConnTimeout, _ := strconv.Atoi(getEnv("PROXY_IDLE_CONN_TIMEOUT", "90"))

	cfg := &Config{
		Server: ServerConfig{
			Host:         getEnv("HOST", "0.0.0.0"),
			Port:         port,
			ReadTimeout:  time.Duration(readTimeout) * time.Second,
			WriteTimeout: time.Duration(writeTimeout) * time.Second,
			IdleTimeout:  time.Duration(idleTimeout) * time.Second,
		},
		Database: DatabaseConfig{
			Path: getEnv("DB_PATH", "./tenants.db"),
		},
		Proxy: ProxyConfig{
			BackendURL:       getEnv("BACKEND_URL", "http://localhost:3000"),
			Timeout:          time.Duration(proxyTimeout) * time.Second,
			MaxIdleConns:     maxIdleConns,
			IdleConnTimeout:  time.Duration(idleConnTimeout) * time.Second,
			DisableKeepAlive: getEnv("PROXY_DISABLE_KEEPALIVE", "false") == "true",
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func (c *Config) ServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}

