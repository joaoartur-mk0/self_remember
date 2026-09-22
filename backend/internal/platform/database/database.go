// Package database abre a conexão com o Postgres e aplica as migrations do
// goose. A aplicação nunca chama gorm.AutoMigrate (ADR-0007): o schema só é
// alterado pelos arquivos versionados em backend/migrations.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// OpenSQL abre e verifica (ping) uma conexão pgx via database/sql.
func OpenSQL(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("open sql.DB: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}

// Migrate aplica todas as migrations pendentes em migrationsFS via goose. Uma
// falha aqui deve abortar o boot: o processo nunca sobe com schema divergente.
func Migrate(ctx context.Context, db *sql.DB, migrationsFS fs.FS) error {
	// Provider.Close() fecha a conexão que recebeu, então não é chamado aqui:
	// db continua em uso pelo GORM depois da migration.
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrationsFS)
	if err != nil {
		return fmt.Errorf("create goose provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("apply migrations: %w", err)
	}

	return nil
}

// NewGORM envolve uma conexão *sql.DB já aberta com GORM, sem abrir um pool
// próprio e sem AutoMigrate.
func NewGORM(db *sql.DB) (*gorm.DB, error) {
	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open gorm: %w", err)
	}

	return gormDB, nil
}
