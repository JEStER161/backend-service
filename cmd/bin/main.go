package main

import (
	"flag"
	"fmt"
	"os"

	"backend-service/configs"
	"backend-service/internal/app"
	"backend-service/pkg/api/logger"
)

// main читает аргументы командной строки и запускает приложение
func main() {
	configPath := flag.String("config", "env.yaml", "specify YAML config file to use")
	dryRun := flag.Bool("dry-run", false, "check the PostgreSQL connection and exit")
	flag.Parse()

	if err := run(*configPath, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "application stopped: %v\n", err)
		os.Exit(1)
	}
}

// run загружает конфигурацию и выбирает обычный запуск или проверку зависимостей
func run(configPath string, dryRun bool) error {
	config, err := configs.Load(configPath)
	if err != nil {
		return err
	}
	log, err := logger.New(os.Stdout, config.Logging)
	if err != nil {
		return fmt.Errorf("create logger: %w", err)
	}

	application := app.NewApp(&config)
	if dryRun {
		return application.DryRun(log)
	}
	return application.Start(log)
}
