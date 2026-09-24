package jobs

import (
	"context"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/riverqueue/river"

	"self-remember/backend/internal/domain/channel"
)

// VerificadorCanalInterval é "a cada minuto" (ADR-0005) — mais fino que o
// ciclo de 5 minutos do resto do Guardião, por isso vive num PeriodicJob
// próprio em vez de dentro do GuardiaoWorker.
const VerificadorCanalInterval = time.Minute

// VerificadorCanalArgs não tem parâmetros: sempre verifica a mesma sessão
// WhatsApp da instância.
type VerificadorCanalArgs struct{}

func (VerificadorCanalArgs) Kind() string { return "verificador_canal" }

// VerificadorCanalWorker checa a sessão do WhatsApp na EvolutionAPI e, se
// caída, dispara um alerta por e-mail — Canal que não depende da Evolution
// para funcionar (ADR-0005). O alerta é disparado só na transição de
// saudável para caída, não a cada minuto enquanto continuar caída, para não
// inundar a caixa de entrada do operador.
type VerificadorCanalWorker struct {
	river.WorkerDefaults[VerificadorCanalArgs]

	whatsapp channel.Channel
	mailer   channel.Channel
	opsEmail string
	logger   *slog.Logger

	// wasDown é lido e escrito por Work, que o River pode chamar de
	// goroutines concorrentes na mesma instância de worker.
	wasDown atomic.Bool
}

func NewVerificadorCanalWorker(whatsapp, mailer channel.Channel, opsEmail string, logger *slog.Logger) *VerificadorCanalWorker {
	return &VerificadorCanalWorker{whatsapp: whatsapp, mailer: mailer, opsEmail: opsEmail, logger: logger}
}

func (w *VerificadorCanalWorker) Work(ctx context.Context, _ *river.Job[VerificadorCanalArgs]) error {
	err := w.whatsapp.HealthCheck(ctx)
	if err == nil {
		if w.wasDown.CompareAndSwap(true, false) {
			w.logger.Info("sessão do whatsapp recuperada")
		}
		return nil
	}

	w.logger.Error("health check da evolutionapi falhou", "error", err)

	if !w.wasDown.CompareAndSwap(false, true) {
		return nil
	}

	if w.opsEmail == "" {
		w.logger.Warn("alerta de sessão caída não enviado: e-mail de operação não configurado")
		return nil
	}

	if _, sendErr := w.mailer.Send(ctx, channel.Message{
		To:   w.opsEmail,
		Body: fmt.Sprintf("Sessão do WhatsApp (EvolutionAPI) desconectada: %s", err),
	}); sendErr != nil {
		w.logger.Error("falha ao enviar alerta por e-mail", "error", sendErr)
	}
	return nil
}
