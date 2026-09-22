// Package user contém as regras puras sobre a conta de um usuário: política
// de senha e verificação de hash. Nada aqui importa Gin, GORM ou golang-jwt —
// é a camada que ARCHITECTURE.md §5 exige testável sem rede e sem banco.
package user

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Parâmetros do argon2id. Não são configuráveis: mudar exige rehash de todas
// as senhas existentes, então qualquer ajuste é uma decisão deliberada no
// código, não uma variável de ambiente.
const (
	argonTime    = 1
	argonMemory  = 64 * 1024
	argonThreads = 4
	argonKeyLen  = 32
	saltLen      = 16
)

// ErrPasswordTooShort é retornado por ValidatePassword quando a senha não
// atende à política mínima.
var ErrPasswordTooShort = errors.New("senha precisa ter ao menos 8 caracteres")

// ErrInvalidHash é retornado por VerifyPassword quando o hash armazenado não
// está no formato esperado.
var ErrInvalidHash = errors.New("formato de hash inválido")

// ValidatePassword aplica a política mínima de senha do sistema.
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ErrPasswordTooShort
	}
	return nil
}

// HashPassword calcula o hash argon2id de uma senha em texto puro, incluindo
// salt e parâmetros no valor retornado para que VerifyPassword não precise
// deles de outra fonte.
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("gerar salt: %w", err)
	}

	hash := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	encoded := fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(hash),
	)
	return encoded, nil
}

// VerifyPassword compara, em tempo constante, uma senha em texto puro com um
// hash gerado por HashPassword.
func VerifyPassword(encodedHash, password string) (bool, error) {
	parts := strings.Split(encodedHash, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrInvalidHash
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return false, fmt.Errorf("%w: versão", ErrInvalidHash)
	}

	var memory, timeCost uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &timeCost, &threads); err != nil {
		return false, fmt.Errorf("%w: parâmetros", ErrInvalidHash)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, fmt.Errorf("%w: salt", ErrInvalidHash)
	}

	wantHash, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, fmt.Errorf("%w: hash", ErrInvalidHash)
	}

	gotHash := argon2.IDKey([]byte(password), salt, timeCost, memory, threads, uint32(len(wantHash)))

	return subtle.ConstantTimeCompare(wantHash, gotHash) == 1, nil
}
