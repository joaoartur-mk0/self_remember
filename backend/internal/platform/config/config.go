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
}

// Load lê a configuração do ambiente, aplicando defaults apenas para valores
// que não são segredo nem string de conexão. DatabaseURL é obrigatório e não
// tem default: subir com uma string de conexão implícita já foi a causa de
// mais de um incidente de "estava usando o banco errado".
func Load() (Config, error) {
	databaseURL, ok := os.LookupEnv("SELF_REMEMBER_DATABASE_URL")
	if !ok || databaseURL == "" {
		return Config{}, fmt.Errorf("SELF_REMEMBER_DATABASE_URL é obrigatório")
	}

	return Config{
		HTTPAddr:    envOrDefault("SELF_REMEMBER_HTTP_ADDR", ":8080"),
		LogLevel:    envOrDefault("SELF_REMEMBER_LOG_LEVEL", "info"),
		DatabaseURL: databaseURL,
	}, nil
}

func envOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
