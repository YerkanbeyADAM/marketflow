package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"marketflow/cmd/cli"
	"marketflow/internal/adapters/inbound/httptransport"
	"marketflow/internal/adapters/outbound/cache"
	"marketflow/internal/adapters/outbound/db"
	"marketflow/internal/adapters/outbound/exchange/testsource"
	"marketflow/internal/app"
	"marketflow/internal/config"
	"marketflow/pkg/logger"
)

func main() {
	cli.InitFlags()
	logger, err := logger.NewCustomLogger()
	if err != nil {
		log.Fatalf("Не удалось создать логгер: %v", err)
	}
	logger.Info("Запуск приложения Marketflow...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cfg, err := config.LoadConfigs()
	if err != nil {
		logger.Error(fmt.Sprintf("Не удалось загрузить конфиг: %v", err))
		os.Exit(1)
	}

	cdg, err := db.NewPostgres(cfg.Postgres)
	if err != nil {
		logger.Error(fmt.Sprintf("Не удалось подключиться к PostgreSQL: %v", err))
		os.Exit(1)
	}
	logger.Info("Подключение к PostgreSQL успешно")
	defer cdg.Close()

	crg, err := cache.NewRedisRepo(cfg.Redis, ctx)
	if err != nil {
		logger.Error(fmt.Sprintf("Не удалось подключиться к Redis: %v", err))
		os.Exit(1)
	}
	logger.Info("Подключение к Redis успешно")
	defer crg.Close()

	svc := app.NewApp(cdg, crg, &testsource.TestDataSource{}, logger)
	svc.Ingest()

	srvPort := cfg.Server.Port
	if cli.PortFlagSet() {
		srvPort = *cli.Port
	}
	srv := httptransport.NewHTTPServer(svc, srvPort, logger)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Serve(); err != nil {
			logger.Error(fmt.Sprintf("Не удалось запустить HTTP сервер: %v", err))
			cancel()
		}
	}()

	logger.Info("MarketFlow приложение запущено и готово к работе.")

	<-stop
	logger.Info("Получен сигнал завершения, останавливаем приложение...")

	cancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelShutdown()
	if err := srv.Shutdown(ctxShutdown); err != nil {
		logger.Error(fmt.Sprintf("Ошибка при завершении HTTP сервера: %v", err))
	}

	logger.Info("Приложение остановлено корректно.")
}
