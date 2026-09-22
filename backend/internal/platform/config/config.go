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
	databaseURL, err := RequireEnv("SELF_REMEMBER_DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	jwtSecret, err := RequireEnv("SELF_REMEMBER_JWT_SECRET")
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:    envOrDefault("SELF_REMEMBER_HTTP_ADDR", ":8080"),
		LogLevel:    envOrDefault("SELF_REMEMBER_LOG_LEVEL", "info"),
		DatabaseURL: databaseURL,
		JWTSecret:   []byte(jwtSecret),
	}, nil
}

// RequireEnv lê uma variável de ambiente obrigatória. Usado para segredos e
// strings de conexão, onde um default silencioso é o problema — cmd/api e
// cmd/admin compartilham essa regra mesmo sem compartilhar Config inteiro.
func RequireEnv(key string) (string, error) {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return "", fmt.Errorf("%s é obrigatório", key)
	}
	return v, nil
}

func envOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
