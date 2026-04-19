package app

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"rm_ai_agent/internal/artifact"
	"rm_ai_agent/internal/channel/dingtalk"
	"rm_ai_agent/internal/channel/feishu"
	"rm_ai_agent/internal/channel/weixin"
	"rm_ai_agent/internal/compression"
	"rm_ai_agent/internal/config"
	"rm_ai_agent/internal/detailagent"
	"rm_ai_agent/internal/edr"
	"rm_ai_agent/internal/i18n"
	"rm_ai_agent/internal/knowledge"
	"rm_ai_agent/internal/logx"
	"rm_ai_agent/internal/memory"
	"rm_ai_agent/internal/model"
	"rm_ai_agent/internal/planner"
	"rm_ai_agent/internal/progress"
	"rm_ai_agent/internal/protocol"
	"rm_ai_agent/internal/prompt"
	"rm_ai_agent/internal/router"
	"rm_ai_agent/internal/scheduler"
	"rm_ai_agent/internal/session"
	"rm_ai_agent/internal/store/sqlite"
)

type App struct {
	config     config.Config
	logger     *logx.Logger
	httpServer *http.Server
	feishu     *feishu.Handler
	dingtalk   *dingtalk.Handler
	weixin     *weixin.Handler
	scheduler  *scheduler.Service
}

func New(cfg config.Config) (*App, error) {
	logger, err := logx.New(cfg.Server.LogLevel.Level(), cfg.Server.LogFile)
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}

	dataStore, err := sqlite.New(cfg.Storage)
	if err != nil {
		_ = logger.Close()
		return nil, fmt.Errorf("create sqlite store: %w", err)
	}

	modelClient := model.NewFallbackClient(cfg.Models, logger)
	edrClient := edr.NewClient(cfg.EDR)
	promptService := prompt.NewService(".")
	compactor := compression.NewService(cfg.Compression, dataStore, modelClient, promptService)
	progressService := progress.NewService(cfg.Progress, modelClient, promptService, logger)
	detailAgentService := detailagent.NewService(cfg.DetailAgent, modelClient, promptService, logger)
	routerService := router.NewService(cfg.Routing, modelClient, promptService, logger)
	memoryService := memory.NewService(cfg.Memory, dataStore)
	artifactService := artifact.NewService(dataStore)
	i18nService := i18n.New(".")
	knowledgeService := knowledge.NewService(cfg.KnowledgeBase)
	plannerService := planner.NewService(modelClient, promptService, logger)
	sessionService := session.NewService(cfg, dataStore, modelClient, compactor, progressService, detailAgentService, routerService, plannerService, memoryService, artifactService, i18nService, knowledgeService, promptService, edrClient, logger)

	feishuClient := feishu.NewClient(cfg.Channel.Feishu, logger)
	weixinClient := weixin.NewClient(cfg.Channel.Weixin, logger)
	dingtalkClient := dingtalk.NewClient(cfg.Channel.Dingtalk, logger)

	// Build notifiers map for scheduler
	schedulerNotifiers := make(map[protocol.Channel]scheduler.Notifier)
	if cfg.Channel.Feishu.Enabled {
		schedulerNotifiers[protocol.ChannelFeishu] = feishuClient
	}
	if cfg.Channel.Weixin.Enabled {
		schedulerNotifiers[protocol.ChannelWeixin] = weixinClient
	}
	if cfg.Channel.Dingtalk.Enabled {
		schedulerNotifiers[protocol.ChannelDingtalk] = dingtalkClient
	}
	schedulerService := scheduler.NewService(cfg.Scheduler, dataStore, sessionService, schedulerNotifiers, logger)

	feishuHandler, err := feishu.NewHandler(cfg.Channel.Feishu, dataStore, sessionService, feishuClient, logger)
	if err != nil {
		_ = logger.Close()
		return nil, fmt.Errorf("create feishu handler: %w", err)
	}

	dingtalkHandler := dingtalk.NewHandler(cfg.Channel.Dingtalk, dataStore, sessionService, dingtalkClient, logger)
	if dingtalkHandler != nil {
		logger.Info("dingtalk handler created", "enabled", cfg.Channel.Dingtalk.Enabled)
	}

	weixinHandler := weixin.NewHandler(cfg.Channel.Weixin, dataStore, sessionService, weixinClient, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	if cfg.Channel.Feishu.Enabled && (strings.EqualFold(cfg.Channel.Feishu.Mode, "webhook") || strings.EqualFold(cfg.Channel.Feishu.Mode, "both")) {
		mux.Handle(cfg.Channel.Feishu.WebhookPath, feishuHandler)
	}
	if cfg.Channel.Feishu.Enabled {
		logger.Info(
			"feishu channel configured",
			"mode", cfg.Channel.Feishu.Mode,
			"app_id", mask(cfg.Channel.Feishu.AppID),
			"webhook_path", cfg.Channel.Feishu.WebhookPath,
		)
	}
	if cfg.Channel.Dingtalk.Enabled && (strings.EqualFold(cfg.Channel.Dingtalk.Mode, "webhook") || strings.EqualFold(cfg.Channel.Dingtalk.Mode, "both")) {
		mux.Handle(cfg.Channel.Dingtalk.WebhookPath, dingtalkHandler)
	}
	if cfg.Channel.Dingtalk.Enabled {
		logger.Info(
			"dingtalk channel configured",
			"mode", cfg.Channel.Dingtalk.Mode,
			"client_id", mask(cfg.Channel.Dingtalk.ClientID),
			"webhook_path", cfg.Channel.Dingtalk.WebhookPath,
		)
	}
	if cfg.Channel.Weixin.Enabled && (strings.EqualFold(cfg.Channel.Weixin.Mode, "webhook") || strings.EqualFold(cfg.Channel.Weixin.Mode, "both")) {
		mux.Handle(cfg.Channel.Weixin.WebhookPath, weixinHandler)
	}
	if cfg.Channel.Weixin.Enabled {
		logger.Info(
			"weixin channel configured",
			"mode", cfg.Channel.Weixin.Mode,
			"bot_id", mask(cfg.Channel.Weixin.BotID),
			"webhook_path", cfg.Channel.Weixin.WebhookPath,
		)
	}
	if strings.TrimSpace(cfg.Server.LogFile) != "" {
		logger.Info("local log file enabled", "path", cfg.Server.LogFile)
	}

	httpServer := &http.Server{
		Addr:              cfg.Server.Address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &App{
		config:     cfg,
		logger:     logger,
		httpServer: httpServer,
		feishu:     feishuHandler,
		dingtalk:   dingtalkHandler,
		weixin:     weixinHandler,
		scheduler:  schedulerService,
	}, nil
}

func (a *App) Close() error {
	if a == nil || a.logger == nil {
		return nil
	}
	return a.logger.Close()
}

func mask(value string) string {
	if len(value) <= 8 {
		return value
	}
	return value[:4] + "..." + value[len(value)-4:]
}

func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		if a.scheduler != nil {
			go a.scheduler.Start(ctx)
		}
		if a.feishu != nil && a.config.Channel.Feishu.Enabled && (strings.EqualFold(a.config.Channel.Feishu.Mode, "longconn") || strings.EqualFold(a.config.Channel.Feishu.Mode, "both")) {
			go func() {
				if err := a.feishu.StartLongConnection(ctx); err != nil {
					a.logger.Error("feishu long connection stopped", "error", err)
				}
			}()
		}
		if a.dingtalk != nil && a.config.Channel.Dingtalk.Enabled && (strings.EqualFold(a.config.Channel.Dingtalk.Mode, "longconn") || strings.EqualFold(a.config.Channel.Dingtalk.Mode, "both")) {
			go func() {
				if err := a.dingtalk.StartLongConnection(ctx); err != nil {
					a.logger.Error("dingtalk long connection stopped", "error", err)
				}
			}()
		}
		if a.weixin != nil && a.config.Channel.Weixin.Enabled && (strings.EqualFold(a.config.Channel.Weixin.Mode, "longconn") || strings.EqualFold(a.config.Channel.Weixin.Mode, "both")) {
			go func() {
				if err := a.weixin.StartLongConnection(ctx); err != nil {
					a.logger.Error("weixin long connection stopped", "error", err)
				}
			}()
		}

		a.logger.Info("http server listening", "address", a.config.Server.Address)
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		a.logger.Info("shutting down")
		return a.httpServer.Shutdown(shutdownCtx)
	case err := <-errCh:
		return err
	}
}
