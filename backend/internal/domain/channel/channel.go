// Package channel define a porta Channel (ARCHITECTURE.md §5): o contrato que
// todo adaptador de canal externo (WhatsApp, e-mail, e futuramente Telegram,
// Discord, Slack) implementa. Não importa Gin, GORM ou golang-jwt — é
// domínio puro, testável sem rede.
package channel

import "context"

// Kind identifica o tipo de um Canal. Um valor tipado em vez de string cru
// evita que qualquer string vire um Kind válido por acidente.
type Kind string

const (
	KindWhatsApp Kind = "whatsapp"
	KindEmail    Kind = "email"
)

// ProviderID é o identificador que o provedor externo atribui a um envio
// (ex.: message ID da EvolutionAPI). Persistido em deliveries.provider_message_id.
type ProviderID string

// Message é o conteúdo que uma Entrega tenta fazer chegar ao usuário.
// CorrelationID é gravado mesmo quando nenhum adaptador do MVP sabe
// correlacionar uma resposta a ele — é o dado que ADR-0006 identifica como
// impossível de reconstruir retroativamente se não for salvo desde já.
type Message struct {
	To            string
	Body          string
	CorrelationID string
}

// Channel é o contrato que toda Entrega usa para tentar alcançar o usuário
// por um meio específico.
type Channel interface {
	Kind() Kind
	Send(ctx context.Context, msg Message) (ProviderID, error)
	HealthCheck(ctx context.Context) error
	// SupportsInboundAck indica se este Canal sabe correlacionar uma resposta
	// do usuário a uma Ocorrência. Todo adaptador do MVP retorna false
	// (ADR-0006); o método existe desde já para que ligar o Reconhecimento
	// por Canal no futuro não exija mudar a porta.
	SupportsInboundAck() bool
}
