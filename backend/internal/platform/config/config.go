// Package config carrega a configuração da aplicação a partir de variáveis
// de ambiente prefixadas com SELF_REMEMBER_.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config contém a configuração necessária para o processo cmd/api. Carrega
// SMTP e Evolution também: desde que a API passou a expor
// verificação de Canal e o painel de status da instância WhatsApp
// (POST /channels/:id/verify/*, GET /whatsapp-instance/*), ela envia
// mensagens pelos dois, não só o worker.
type Config struct {
	HTTPAddr    string
	LogLevel    string
	DatabaseURL string
	JWTSecret   []byte
	SMTP        SMTPConfig
	Evolution   EvolutionConfig
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

	smtpCfg, err := loadSMTP()
	if err != nil {
		return Config{}, err
	}

	evolutionCfg, err := loadEvolution()
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:    envOrDefault("SELF_REMEMBER_HTTP_ADDR", ":8080"),
		LogLevel:    envOrDefault("SELF_REMEMBER_LOG_LEVEL", "info"),
		DatabaseURL: databaseURL,
		JWTSecret:   []byte(jwtSecret),
		SMTP:        smtpCfg,
		Evolution:   evolutionCfg,
	}, nil
}

// SMTPConfig é a credencial de instância do Canal de e-mail (ADR-0001: de
// instância, não por usuário) — o Canal que "não pode falhar" porque
// carrega o alerta de EvolutionAPI desconectada.
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// EvolutionConfig é a credencial de instância do Canal de WhatsApp
// (ADR-0005).
type EvolutionConfig struct {
	BaseURL              string
	APIKey               string
	Instance             string
	MaxMessagesPerMinute int
}

// WorkerConfig contém a configuração necessária para o processo cmd/worker.
// Separado de Config porque o worker não serve HTTP e não assina JWT — exigir
// SELF_REMEMBER_JWT_SECRET dele só para satisfazer um Load() genérico seria
// pedir um segredo que esse processo nunca usa.
type WorkerConfig struct {
	LogLevel    string
	DatabaseURL string
	SMTP        SMTPConfig
	Evolution   EvolutionConfig
	// OpsEmail recebe o alerta de sessão do WhatsApp caída (ADR-0005). Vazio
	// desativa o alerta sem impedir o boot — é a EvolutionAPI que não pode
	// ser uma dependência rígida do resto do sistema, não o contrário.
	OpsEmail string
}

// LoadWorker lê a configuração do cmd/worker.
func LoadWorker() (WorkerConfig, error) {
	databaseURL, err := RequireEnv("SELF_REMEMBER_DATABASE_URL")
	if err != nil {
		return WorkerConfig{}, err
	}

	smtpCfg, err := loadSMTP()
	if err != nil {
		return WorkerConfig{}, err
	}

	evolutionCfg, err := loadEvolution()
	if err != nil {
		return WorkerConfig{}, err
	}

	return WorkerConfig{
		LogLevel:    envOrDefault("SELF_REMEMBER_LOG_LEVEL", "info"),
		DatabaseURL: databaseURL,
		SMTP:        smtpCfg,
		Evolution:   evolutionCfg,
		OpsEmail:    envOrDefault("SELF_REMEMBER_OPS_EMAIL", ""),
	}, nil
}

// loadSMTP lê a credencial de instância do Canal de e-mail — compartilhada
// entre cmd/api (verificação de Canal) e cmd/worker (Entrega e alerta de
// EvolutionAPI desconectada).
func loadSMTP() (SMTPConfig, error) {
	host, err := RequireEnv("SELF_REMEMBER_SMTP_HOST")
	if err != nil {
		return SMTPConfig{}, err
	}
	portStr, err := RequireEnv("SELF_REMEMBER_SMTP_PORT")
	if err != nil {
		return SMTPConfig{}, err
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return SMTPConfig{}, fmt.Errorf("SELF_REMEMBER_SMTP_PORT inválido: %w", err)
	}
	username, err := RequireEnv("SELF_REMEMBER_SMTP_USERNAME")
	if err != nil {
		return SMTPConfig{}, err
	}
	password, err := RequireEnv("SELF_REMEMBER_SMTP_PASSWORD")
	if err != nil {
		return SMTPConfig{}, err
	}
	from, err := RequireEnv("SELF_REMEMBER_SMTP_FROM")
	if err != nil {
		return SMTPConfig{}, err
	}

	return SMTPConfig{Host: host, Port: port, Username: username, Password: password, From: from}, nil
}

// loadEvolution lê a credencial de instância do Canal de WhatsApp —
// compartilhada entre cmd/api (verificação de Canal e painel de status da
// instância) e cmd/worker (Entrega e health check, ADR-0005).
func loadEvolution() (EvolutionConfig, error) {
	baseURL, err := RequireEnv("SELF_REMEMBER_EVOLUTION_BASE_URL")
	if err != nil {
		return EvolutionConfig{}, err
	}
	apiKey, err := RequireEnv("SELF_REMEMBER_EVOLUTION_API_KEY")
	if err != nil {
		return EvolutionConfig{}, err
	}
	instance, err := RequireEnv("SELF_REMEMBER_EVOLUTION_INSTANCE")
	if err != nil {
		return EvolutionConfig{}, err
	}
	maxPerMinuteStr := envOrDefault("SELF_REMEMBER_EVOLUTION_MAX_MESSAGES_PER_MINUTE", "20")
	maxPerMinute, err := strconv.Atoi(maxPerMinuteStr)
	if err != nil {
		return EvolutionConfig{}, fmt.Errorf("SELF_REMEMBER_EVOLUTION_MAX_MESSAGES_PER_MINUTE inválido: %w", err)
	}

	return EvolutionConfig{
		BaseURL:              baseURL,
		APIKey:               apiKey,
		Instance:             instance,
		MaxMessagesPerMinute: maxPerMinute,
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
