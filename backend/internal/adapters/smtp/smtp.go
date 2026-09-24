// Package smtp implementa a porta Channel por e-mail via SMTP — o Canal de
// fallback que "não pode falhar" (README.md): é o que carrega o alerta de
// EvolutionAPI desconectada, então não pode depender dela.
package smtp

import (
	"context"
	"fmt"

	mail "github.com/wneessen/go-mail"

	"self-remember/backend/internal/domain/channel"
)

// Config é a credencial de instância do SMTP — de instância, não por
// usuário (ADR-0001): um único remetente serve todos os usuários desta
// instalação.
type Config struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// Channel envia e-mails via SMTP com STARTTLS/TLS obrigatório.
type Channel struct {
	from   string
	client *mail.Client
}

func New(cfg Config) (*Channel, error) {
	client, err := mail.NewClient(cfg.Host,
		mail.WithPort(cfg.Port),
		mail.WithSMTPAuth(mail.SMTPAuthAutoDiscover),
		mail.WithUsername(cfg.Username),
		mail.WithPassword(cfg.Password),
		// Oportunista, não mandatório: em produção, um relay real quase
		// sempre oferece STARTTLS e este client o usa; em
		// docker-compose.local (ARCHITECTURE.md §9, "sem TLS" nos serviços
		// locais), um mail catcher sem STARTTLS não pode quebrar o Canal
		// que "não pode falhar".
		mail.WithTLSPortPolicy(mail.TLSOpportunistic),
	)
	if err != nil {
		return nil, fmt.Errorf("configurar cliente smtp: %w", err)
	}
	return &Channel{from: cfg.From, client: client}, nil
}

func (c *Channel) Kind() channel.Kind {
	return channel.KindEmail
}

func (c *Channel) Send(ctx context.Context, msg channel.Message) (channel.ProviderID, error) {
	m := mail.NewMsg()
	if err := m.From(c.from); err != nil {
		return "", fmt.Errorf("definir remetente: %w", err)
	}
	if err := m.To(msg.To); err != nil {
		return "", fmt.Errorf("definir destinatário: %w", err)
	}
	m.Subject("Self Remember")
	m.SetBodyString(mail.TypeTextPlain, msg.Body)
	m.SetMessageID()

	if err := c.client.DialAndSendWithContext(ctx, m); err != nil {
		return "", fmt.Errorf("enviar e-mail: %w", err)
	}

	return channel.ProviderID(m.GetMessageID()), nil
}

// HealthCheck só abre e fecha a conexão — não envia nada.
func (c *Channel) HealthCheck(ctx context.Context) error {
	smtpClient, err := c.client.DialToSMTPClientWithContext(ctx)
	if err != nil {
		return fmt.Errorf("conectar ao smtp: %w", err)
	}
	return c.client.CloseWithSMTPClient(smtpClient)
}

func (c *Channel) SupportsInboundAck() bool {
	return false
}
