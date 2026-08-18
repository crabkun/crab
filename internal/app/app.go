package app

import (
	"context"
	"fmt"

	"github.com/crabkun/crab/internal/client"
	"github.com/crabkun/crab/internal/config"
	"github.com/crabkun/crab/internal/logx"
	"github.com/crabkun/crab/internal/master"
	"github.com/crabkun/crab/internal/server"
	"github.com/crabkun/crab/internal/version"
)

func Run(ctx context.Context, configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	return RunConfig(ctx, cfg)
}

func RunConfig(ctx context.Context, cfg *config.Config) error {
	logger, err := logx.New(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("unsupported log level: %s", cfg.LogLevel)
	}

	logger.Info("Crab "+version.Version, "mode", cfg.Mode)

	switch cfg.Mode {
	case config.ModeMaster:
		return master.New(cfg.Master, logger).Run(ctx)
	case config.ModeServer:
		return server.New(cfg.Server, logger).Run(ctx)
	case config.ModeClient:
		return client.New(cfg.Client, logger).Run(ctx)
	default:
		return fmt.Errorf("unsupported mode %q", cfg.Mode)
	}
}
