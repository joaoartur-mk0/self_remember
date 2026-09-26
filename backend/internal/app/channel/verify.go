// Package channel contém o caso de uso de verificação de Canal: provar que
// o usuário controla o endereço informado (e-mail ou telefone) antes que o
// motor de cobrança confie nele para entregar uma Cobrança.
package channel

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"self-remember/backend/internal/adapters/postgres"
	domainchannel "self-remember/backend/internal/domain/channel"
)

var (
	// ErrUnsupportedKind cobre um Kind sem adaptador de envio configurado —
	// não deveria acontecer com os dois Kinds hoje suportados, mas o mapa de
	// remetentes é injetado, então o caso de uso não pode assumir isso.
	ErrUnsupportedKind = errors.New("tipo de canal não suportado para verificação")
	// ErrNoCodeRequested cobre Confirm chamado sem um Request anterior.
	ErrNoCodeRequested = errors.New("nenhum código de verificação foi solicitado")
	ErrCodeExpired     = errors.New("código de verificação expirado")
	ErrInvalidCode     = errors.New("código de verificação inválido")
)

// verificationCodeTTL é quanto tempo um código de verificação vale — curto
// de propósito, é um segredo de vida curta (mesma lógica do comentário em
// migration 0009).
const verificationCodeTTL = 15 * time.Minute

// VerificationUseCase envia e confirma o código de verificação de um Canal.
type VerificationUseCase struct {
	channels *postgres.ChannelRepository
	senders  map[domainchannel.Kind]domainchannel.Channel
	now      func() time.Time
}

func NewVerificationUseCase(
	channels *postgres.ChannelRepository,
	senders map[domainchannel.Kind]domainchannel.Channel,
	now func() time.Time,
) *VerificationUseCase {
	return &VerificationUseCase{channels: channels, senders: senders, now: now}
}

// Request gera um novo código, grava o hash em channels, e o envia pelo
// próprio Canal a verificar — provar que o usuário recebe mensagens nele é
// o próprio teste.
func (uc *VerificationUseCase) Request(ctx context.Context, channelID, userID uuid.UUID) error {
	ch, err := uc.channels.FindByIDForUser(ctx, channelID, userID)
	if err != nil {
		return err
	}

	sender, ok := uc.senders[domainchannel.Kind(ch.Kind)]
	if !ok {
		return ErrUnsupportedKind
	}

	code, err := generateCode()
	if err != nil {
		return fmt.Errorf("gerar código: %w", err)
	}

	if err := uc.channels.SetVerificationCode(ctx, ch.ID, hashCode(code), uc.now().Add(verificationCodeTTL)); err != nil {
		return err
	}

	_, err = sender.Send(ctx, domainchannel.Message{
		To:            ch.Address,
		Body:          fmt.Sprintf("Seu código de verificação do Self Remember é %s. Ele expira em 15 minutos.", code),
		CorrelationID: ch.ID.String(),
	})
	return err
}

// Confirm compara o código informado com o hash gravado por Request, em
// tempo constante, e marca o Canal como verificado se coincidir.
func (uc *VerificationUseCase) Confirm(ctx context.Context, channelID, userID uuid.UUID, code string) error {
	ch, err := uc.channels.FindByIDForUser(ctx, channelID, userID)
	if err != nil {
		return err
	}

	if ch.VerificationCodeHash == nil || ch.VerificationExpiresAt == nil {
		return ErrNoCodeRequested
	}
	if uc.now().After(*ch.VerificationExpiresAt) {
		return ErrCodeExpired
	}
	if subtle.ConstantTimeCompare([]byte(hashCode(code)), []byte(*ch.VerificationCodeHash)) != 1 {
		return ErrInvalidCode
	}

	return uc.channels.MarkVerified(ctx, ch.ID, uc.now())
}

// generateCode sorteia um código numérico de 6 dígitos, com zeros à
// esquerda preservados.
func generateCode() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	n := binary.BigEndian.Uint32(b) % 1_000_000
	return fmt.Sprintf("%06d", n), nil
}

func hashCode(code string) string {
	sum := sha256.Sum256([]byte(code))
	return hex.EncodeToString(sum[:])
}
