// Package evolution implementa a porta Channel por WhatsApp via uma
// instância self-hosted da EvolutionAPI (ADR-0005). Traz as duas mitigações
// obrigatórias descritas na ADR que não são opcionais: teto de mensagens por
// minuto com espaçamento aleatório, e HealthCheck que reporta sessão caída.
package evolution

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"time"

	"golang.org/x/time/rate"

	"self-remember/backend/internal/domain/channel"
)

// Config é a credencial de instância da EvolutionAPI (ADR-0001: de
// instância, não por usuário).
type Config struct {
	BaseURL string
	APIKey  string
	// Instance é o nome da instância WhatsApp já pareada por QR code na
	// EvolutionAPI.
	Instance string
	// MaxMessagesPerMinute é o teto de mensagens por minuto por instância
	// (ADR-0005). Zero desativa o limite — não recomendado em produção.
	MaxMessagesPerMinute int
}

// maxJitter é o espaçamento aleatório máximo entre envios (ADR-0005), além
// do que o rate limiter já impõe.
const maxJitter = 2 * time.Second

// Channel envia mensagens de texto via EvolutionAPI e verifica o estado da
// sessão WhatsApp.
type Channel struct {
	cfg        Config
	httpClient *http.Client
	limiter    *rate.Limiter
}

func New(cfg Config, httpClient *http.Client) *Channel {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}

	var limiter *rate.Limiter
	if cfg.MaxMessagesPerMinute > 0 {
		limiter = rate.NewLimiter(rate.Limit(float64(cfg.MaxMessagesPerMinute)/60.0), 1)
	}

	return &Channel{cfg: cfg, httpClient: httpClient, limiter: limiter}
}

func (c *Channel) Kind() channel.Kind {
	return channel.KindWhatsApp
}

type sendTextRequest struct {
	Number string `json:"number"`
	Text   string `json:"text"`
}

// sendTextResponse cobre apenas o campo que usamos (key.id); a EvolutionAPI
// não documenta formalmente o corpo de sucesso deste endpoint, então um
// campo ausente ou de formato inesperado não é tratado como falha de envio
// — a mensagem já foi aceita (201), só a correlação com o provedor se perde.
type sendTextResponse struct {
	Key struct {
		ID string `json:"id"`
	} `json:"key"`
}

func (c *Channel) Send(ctx context.Context, msg channel.Message) (channel.ProviderID, error) {
	if c.limiter != nil {
		if err := c.limiter.Wait(ctx); err != nil {
			return "", fmt.Errorf("aguardar limite de taxa: %w", err)
		}
	}
	if err := sleepJitter(ctx); err != nil {
		return "", err
	}

	body, err := json.Marshal(sendTextRequest{Number: msg.To, Text: msg.Body})
	if err != nil {
		return "", fmt.Errorf("montar corpo da requisição: %w", err)
	}

	url := fmt.Sprintf("%s/message/sendText/%s", c.cfg.BaseURL, c.cfg.Instance)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("criar requisição: %w", err)
	}
	req.Header.Set("apikey", c.cfg.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("enviar mensagem: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("evolutionapi respondeu %d ao enviar mensagem", resp.StatusCode)
	}

	var result sendTextResponse
	_ = json.NewDecoder(resp.Body).Decode(&result)
	return channel.ProviderID(result.Key.ID), nil
}

type connectionStateResponse struct {
	Instance struct {
		State string `json:"state"`
	} `json:"instance"`
}

// ConnectionState consulta o estado da sessão da única instância WhatsApp
// desta instalação (ADR-0001: instância, não por usuário) — "open",
// "connecting" ou "close". Usado tanto pelo HealthCheck quanto pelo painel
// de status exposto na API para qualquer usuário logado.
func (c *Channel) ConnectionState(ctx context.Context) (string, error) {
	url := fmt.Sprintf("%s/instance/connectionState/%s", c.cfg.BaseURL, c.cfg.Instance)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("criar requisição: %w", err)
	}
	req.Header.Set("apikey", c.cfg.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("consultar estado da conexão: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("evolutionapi respondeu %d ao consultar estado da conexão", resp.StatusCode)
	}

	var result connectionStateResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decodificar estado da conexão: %w", err)
	}
	return result.Instance.State, nil
}

// HealthCheck reporta erro se a sessão WhatsApp não estiver com state=open
// (ADR-0005: "Health check da instância Evolution a cada minuto. Sessão
// desconectada é um incidente que dispara alerta pelo canal de e-mail").
func (c *Channel) HealthCheck(ctx context.Context) error {
	state, err := c.ConnectionState(ctx)
	if err != nil {
		return err
	}
	if state != "open" {
		return fmt.Errorf("sessão do WhatsApp desconectada (state=%s)", state)
	}
	return nil
}

type connectResponse struct {
	// Base64 é a QR code como data URI ("data:image/png;base64,...") — vazio
	// quando a instância já está conectada ou aguardando um pareamento
	// anterior ainda válido; o chamador decide o que fazer com esse caso
	// consultando ConnectionState.
	Base64 string `json:"base64"`
}

// Connect inicia (ou retoma) o pareamento da única instância WhatsApp desta
// instalação e devolve o QR code a exibir, se houver um novo disponível.
func (c *Channel) Connect(ctx context.Context) (qrCodeBase64 string, err error) {
	url := fmt.Sprintf("%s/instance/connect/%s", c.cfg.BaseURL, c.cfg.Instance)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("criar requisição: %w", err)
	}
	req.Header.Set("apikey", c.cfg.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("solicitar pareamento: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("evolutionapi respondeu %d ao solicitar pareamento", resp.StatusCode)
	}

	var result connectResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decodificar resposta de pareamento: %w", err)
	}
	return result.Base64, nil
}

func (c *Channel) SupportsInboundAck() bool {
	return false
}

func sleepJitter(ctx context.Context) error {
	jitter := time.Duration(rand.Int64N(int64(maxJitter)))
	timer := time.NewTimer(jitter)
	defer timer.Stop()

	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
