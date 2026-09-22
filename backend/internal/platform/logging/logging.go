// Package logging fornece o logger estruturado em JSON usado por toda a aplicação.
package logging

import (
	"log/slog"
	"os"
)

// New cria um *slog.Logger que escreve JSON em stdout no nível informado.
// Um nível inválido cai em info, nunca falha o boot.
func New(level string) *slog.Logger {
	var lvl slog.Level
	if err := lvl.UnmarshalText([]byte(level)); err != nil {
		lvl = slog.LevelInfo
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: lvl})
	return slog.New(handler)
}
