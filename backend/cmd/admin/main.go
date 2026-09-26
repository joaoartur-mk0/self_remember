// cmd/admin dá os comandos administrativos que o cadastro fechado por
// padrão (ADR-0001) exige para existir: hoje, criar o primeiro usuário.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"self-remember/backend/internal/adapters/postgres"
	"self-remember/backend/internal/domain/commitment"
	"self-remember/backend/internal/domain/user"
	"self-remember/backend/internal/platform/config"
	"self-remember/backend/internal/platform/database"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "uso: admin <comando>\n\ncomandos disponíveis:\n  create-user --email --name --timezone")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "create-user":
		if err := runCreateUser(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "create-user:", err)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "comando desconhecido: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runCreateUser(args []string) error {
	fs := flag.NewFlagSet("create-user", flag.ContinueOnError)
	email := fs.String("email", "", "e-mail do novo usuário")
	name := fs.String("name", "", "nome do novo usuário")
	timezone := fs.String("timezone", "", "fuso IANA do novo usuário (ex.: America/Sao_Paulo)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *email == "" || *name == "" || *timezone == "" {
		return fmt.Errorf("--email, --name e --timezone são obrigatórios")
	}

	// Um usuário nunca existe sem fuso: a expansão de recorrência e a Janela
	// de Silêncio dependem dele (migration 0001).
	if _, err := time.LoadLocation(*timezone); err != nil {
		return fmt.Errorf("--timezone inválido: %w", err)
	}

	password, err := readPasswordFromStdin()
	if err != nil {
		return fmt.Errorf("ler senha: %w", err)
	}

	if err := user.ValidatePassword(password); err != nil {
		return err
	}

	passwordHash, err := user.HashPassword(password)
	if err != nil {
		return fmt.Errorf("gerar hash da senha: %w", err)
	}

	databaseURL, err := config.RequireEnv("SELF_REMEMBER_DATABASE_URL")
	if err != nil {
		return err
	}

	sqlDB, err := database.OpenSQL(databaseURL)
	if err != nil {
		return fmt.Errorf("conectar ao banco: %w", err)
	}
	defer sqlDB.Close()

	gormDB, err := database.NewGORM(sqlDB)
	if err != nil {
		return fmt.Errorf("inicializar gorm: %w", err)
	}

	userID := uuid.New()
	err = gormDB.Transaction(func(tx *gorm.DB) error {
		users := postgres.NewUserRepository(tx)
		if err := users.Create(context.Background(), &postgres.User{
			ID:           userID,
			Email:        *email,
			PasswordHash: passwordHash,
			Name:         *name,
		}); err != nil {
			return fmt.Errorf("criar usuário: %w", err)
		}

		settings := postgres.NewUserSettingsRepository(tx)
		if err := settings.Upsert(context.Background(), &postgres.UserSettingsModel{
			UserID:   userID,
			Timezone: *timezone,
			// Mesmos valores do DEFAULT da migration 0013 — Upsert grava a
			// coluna explicitamente, então um zero-value aqui sobrescreveria
			// o default do banco com string vazia.
			DefaultNagProfile:      string(commitment.NagProfileNormal),
			DefaultCarryOverPolicy: string(commitment.CarryOverStack),
		}); err != nil {
			return fmt.Errorf("gravar ajustes: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	fmt.Printf("usuário %s criado\n", *email)
	return nil
}

// readPasswordFromStdin lê a senha de stdin, nunca de um argumento de linha de
// comando: argumentos ficam expostos em `ps` e no histórico do shell.
func readPasswordFromStdin() (string, error) {
	fmt.Fprint(os.Stderr, "senha: ")
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
