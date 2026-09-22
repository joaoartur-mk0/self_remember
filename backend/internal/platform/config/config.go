// Package config carrega a configuração da aplicação a partir de variáveis
// de ambiente prefixadas com SELF_REMEMBER_.
package config

import (
	"fmt"
	"os"
)

// Config contém a configuração necessária para o processo cmd/api.
type Config struct {
	HTTPAddr    string
	LogLevel    string
	DatabaseURL string
	JWTSecret   []byte
}

// Load lê a configuração do ambiente, aplicando defaults apenas para valores
// que não são segredo nem string de conexão. DatabaseURL e JWTSecret são
// obrigatórios e não têm default: subir com um segredo implícito (ou pior,
// hardcoded) já foi a causa de mais de um incidente de segurança.
func Load() (Config, error) {
	databaseURL, ok := os.LookupEnv("SELF_REMEMBER_DATABASE_URL")
	if !ok || databaseURL == "" {
		return Config{}, fmt.Errorf("SELF_REMEMBER_DATABASE_URL é obrigatório")
	}

	jwtSecret, ok := os.LookupEnv("SELF_REMEMBER_JWT_SECRET")
	if !ok || jwtSecret == "" {
		return Config{}, fmt.Errorf("SELF_REMEMBER_JWT_SECRET é obrigatório")
	}

	return Config{
		HTTPAddr:    envOrDefault("SELF_REMEMBER_HTTP_ADDR", ":8080"),
		LogLevel:    envOrDefault("SELF_REMEMBER_LOG_LEVEL", "info"),
		DatabaseURL: databaseURL,
		JWTSecret:   []byte(jwtSecret),
	}, nil
}

func envOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
