// Package config carrega a configuração da aplicação a partir de variáveis
// de ambiente prefixadas com SELF_REMEMBER_.
package config

import "os"

// Config contém a configuração necessária para o processo cmd/api.
type Config struct {
	HTTPAddr string
	LogLevel string
}

// Load lê a configuração do ambiente, aplicando defaults apenas para valores
// que não são segredo nem string de conexão.
func Load() (Config, error) {
	return Config{
		HTTPAddr: envOrDefault("SELF_REMEMBER_HTTP_ADDR", ":8080"),
		LogLevel: envOrDefault("SELF_REMEMBER_LOG_LEVEL", "info"),
	}, nil
}

func envOrDefault(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
