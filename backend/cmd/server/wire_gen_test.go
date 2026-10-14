package main

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/handler"
	"github.com/Wei-Shaw/sub2api/internal/securityaudit"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestProvideServiceBuildInfo(t *testing.T) {
	in := handler.BuildInfo{
		Version:   "v-test",
		BuildType: "release",
	}
	out := provideServiceBuildInfo(in)
	require.Equal(t, in.Version, out.Version)
	require.Equal(t, in.BuildType, out.BuildType)
}

func TestProvideCleanup_WithMinimalDependencies_NoPanic(t *testing.T) {
	cfg := &config.Config{}

	oauthSvc := service.NewOAuthService(nil, nil)
	openAIOAuthSvc := service.NewOpenAIOAuthService(nil, nil)
	geminiOAuthSvc := service.NewGeminiOAuthService(nil, nil, nil, nil, cfg)
	antigravityOAuthSvc := service.NewAntigravityOAuthService(nil)

	tokenRefreshSvc := service.NewTokenRefreshService(
		nil,
		oauthSvc,
		openAIOAuthSvc,
		geminiOAuthSvc,
		antigravityOAuthSvc,
		nil,
		nil,
		cfg,
		nil,
	)
	accountExpirySvc := service.NewAccountExpiryService(nil, time.Second)
	codexVersionSyncSvc := service.NewOpenAICodexVersionSyncService(nil, nil, nil, time.Second)
	claudeCodeVersionSyncSvc := service.NewClaudeCodeVersionSyncService(nil, nil, nil, time.Second)
	proxyExpirySvc := service.NewProxyExpiryService(nil, time.Second)
	subscriptionExpirySvc := service.NewSubscriptionExpiryService(nil, time.Second)
	pricingSvc := service.NewPricingService(cfg, nil)
	emailQueueSvc := service.NewEmailQueueService(nil, 1)
	billingCacheSvc := service.NewBillingCacheService(nil, nil, nil, nil, nil, nil, cfg, nil)
	idempotencyCleanupSvc := service.NewIdempotencyCleanupService(nil, cfg)
	schedulerSnapshotSvc := service.NewSchedulerSnapshotService(nil, nil, nil, nil, cfg)
	opsSystemLogSinkSvc := service.NewOpsSystemLogSink(nil)

	cleanup := provideCleanup(
		(*ent.Client)(nil),   // entClient
		(*redis.Client)(nil), // redis
		&service.OpsMetricsCollector{},
		&service.OpsAggregationService{},
		&service.OpsAlertEvaluatorService{},
		&service.OpsCleanupService{},
		&service.OpsScheduledReportService{},
		opsSystemLogSinkSvc,
		(*service.OpsService)(nil),                  // opsService
		(*service.OpsIngressRejectAggregator)(nil),  // opsIngressRejectAggregator
		(*service.APIKeyService)(nil),               // apiKeyService
		(*service.AuthCacheInvalidationWorker)(nil), // authCacheInvalidationWorker
		schedulerSnapshotSvc,
		tokenRefreshSvc,
		accountExpirySvc,
		(*service.BalanceExpiryService)(nil),          // balanceExpiry
		(*service.CNProviderBalanceCheckService)(nil), // cnProviderBalanceCheck
		codexVersionSyncSvc,
		claudeCodeVersionSyncSvc,
		proxyExpirySvc,
		subscriptionExpirySvc,
		&service.UsageCleanupService{},
		idempotencyCleanupSvc,
		service.NewGeminiCachedContentService(nil, nil, cfg),
		&service.BatchImageCleanupService{},
		(*service.BatchImageWorkerRuntime)(nil), // batchImageWorker
		pricingSvc,
		emailQueueSvc,
		billingCacheSvc,
		&service.UsageRecordWorkerPool{},
		(*service.ImagePlaygroundTaskWorkerPool)(nil), // imagePlaygroundTaskWorkerPool
		&service.SubscriptionService{},
		oauthSvc,
		openAIOAuthSvc,
		geminiOAuthSvc,
		antigravityOAuthSvc,
		(*service.GrokOAuthService)(nil),              // grokOAuth
		(*service.OpenAIGatewayService)(nil),          // openAIGateway
		(*service.ScheduledTestRunnerService)(nil),    // scheduledTestRunner
		(*service.BackupService)(nil),                 // backupSvc
		(*service.PaymentOrderExpiryService)(nil),     // paymentOrderExpiry
		(*service.ChannelMonitorRunner)(nil),          // channelMonitorRunner
		(*service.ChannelMonitorV2Aggregator)(nil),    // channelMonitorV2Aggregator
		(*service.UserPlatformQuotaUsageFlusher)(nil), // quotaFlusher
		(*service.UpstreamBillingProbeService)(nil),   // upstreamBillingProbe
		(*service.OllamaCloudUsageService)(nil),       // ollamaCloudUsage
		(*service.OpenCodeGoUsageService)(nil),        // opencodeGoUsage
		(*service.AuditLogService)(nil),               // auditLog
		(*service.OpenAIQuotaAutoResetService)(nil),   // openAIAutoReset
		(*securityaudit.PromptService)(nil),           // promptAudit
		(*service.PluginManager)(nil),                 // pluginManager
	)

	require.NotPanics(t, func() {
		cleanup()
	})
}
