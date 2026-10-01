package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"

	"self-remember/backend/internal/adapters/evolution"
	"self-remember/backend/internal/adapters/postgres"
	"self-remember/backend/internal/adapters/smtp"
	commitmentapp "self-remember/backend/internal/app/commitment"
	"self-remember/backend/internal/domain/channel"
	"self-remember/backend/internal/jobs"
	"self-remember/backend/internal/platform/config"
	"self-remember/backend/internal/platform/database"
	"self-remember/backend/internal/platform/logging"
	"self-remember/backend/migrations"
)

func main() {
	cfg, err := config.LoadWorker()
	if err != nil {
		os.Stderr.WriteString("failed to load config: " + err.Error() + "\n")
		os.Exit(1)
	}

	logger := logging.New(cfg.LogLevel)
	logger.Info("starting worker")

	sqlDB, err := database.OpenSQL(cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer sqlDB.Close()

	if err := database.Migrate(context.Background(), sqlDB, migrations.FS); err != nil {
		logger.Error("failed to apply migrations", "error", err)
		os.Exit(1)
	}

	gormDB, err := database.NewGORM(sqlDB)
	if err != nil {
		logger.Error("failed to initialize gorm", "error", err)
		os.Exit(1)
	}

	commitmentRepo := postgres.NewCommitmentRepository(gormDB)
	occurrenceRepo := postgres.NewOccurrenceRepository(gormDB)
	deliveryRepo := postgres.NewDeliveryRepository(gormDB)
	channelRepo := postgres.NewChannelRepository(gormDB)

	dbPool, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to open pgx pool", "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	riverDriver := riverpgxv5.New(dbPool)

	// River tem migrations próprias, aplicadas separadamente das nossas
	// (ADR-0007): a tabela river_job não é gerenciada pelo goose.
	migrator, err := rivermigrate.New(riverDriver, nil)
	if err != nil {
		logger.Error("failed to create river migrator", "error", err)
		os.Exit(1)
	}
	if _, err := migrator.Migrate(context.Background(), rivermigrate.DirectionUp, nil); err != nil {
		logger.Error("failed to apply river migrations", "error", err)
		os.Exit(1)
	}

	smtpChannel, err := smtp.New(smtp.Config{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		Username: cfg.SMTP.Username,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
	})
	if err != nil {
		logger.Error("failed to configure smtp channel", "error", err)
		os.Exit(1)
	}
	evolutionChannel := evolution.New(evolution.Config{
		BaseURL:              cfg.Evolution.BaseURL,
		APIKey:               cfg.Evolution.APIKey,
		Instance:             cfg.Evolution.Instance,
		MaxMessagesPerMinute: cfg.Evolution.MaxMessagesPerMinute,
	}, nil)

	senders := map[channel.Kind]channel.Channel{
		channel.KindEmail:    smtpChannel,
		channel.KindWhatsApp: evolutionChannel,
	}

	workers := river.NewWorkers()
	materializeCommitment := commitmentapp.NewMaterializeUseCase(occurrenceRepo)
	river.AddWorker(workers, jobs.NewMaterializadorWorker(commitmentRepo, materializeCommitment, logger, time.Now))
	river.AddWorker(workers, jobs.NewScannerWorker(dbPool, logger, time.Now))
	river.AddWorker(workers, jobs.NewEntregadorWorker(deliveryRepo, channelRepo, occurrenceRepo, senders, logger))
	river.AddWorker(workers, jobs.NewGuardiaoWorker(occurrenceRepo, logger, time.Now))
	river.AddWorker(workers, jobs.NewVerificadorCanalWorker(evolutionChannel, smtpChannel, cfg.OpsEmail, logger))

	riverClient, err := river.NewClient(riverDriver, &river.Config{
		Queues: map[string]river.QueueConfig{
			river.QueueDefault: {MaxWorkers: 10},
		},
		Workers: workers,
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(
				river.PeriodicInterval(jobs.MaterializadorInterval),
				func() (river.JobArgs, *river.InsertOpts) {
					return jobs.MaterializadorArgs{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: true},
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(jobs.ScannerInterval),
				func() (river.JobArgs, *river.InsertOpts) {
					return jobs.ScannerArgs{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: true},
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(jobs.GuardiaoInterval),
				func() (river.JobArgs, *river.InsertOpts) {
					return jobs.GuardiaoArgs{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: true},
			),
			river.NewPeriodicJob(
				river.PeriodicInterval(jobs.VerificadorCanalInterval),
				func() (river.JobArgs, *river.InsertOpts) {
					return jobs.VerificadorCanalArgs{}, nil
				},
				&river.PeriodicJobOpts{RunOnStart: true},
			),
		},
	})
	if err != nil {
		logger.Error("failed to create river client", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := riverClient.Start(ctx); err != nil {
		logger.Error("failed to start river client", "error", err)
		os.Exit(1)
	}

	logger.Info("worker started")
	<-ctx.Done()

	logger.Info("shutting down")
	stopCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := riverClient.Stop(stopCtx); err != nil {
		logger.Error("graceful shutdown failed", "error", err)
	}
}
