package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type auditMiddlewareCaptureRepo struct {
	mu   sync.Mutex
	logs []*service.AuditLog
}

func (r *auditMiddlewareCaptureRepo) BatchInsert(_ context.Context, logs []*service.AuditLog) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, logs...)
	return int64(len(logs)), nil
}

func (r *auditMiddlewareCaptureRepo) Insert(_ context.Context, log *service.AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logs = append(r.logs, log)
	return nil
}

func (r *auditMiddlewareCaptureRepo) List(context.Context, *service.AuditLogFilter) (*service.AuditLogList, error) {
	return &service.AuditLogList{}, nil
}

func (r *auditMiddlewareCaptureRepo) GetByID(context.Context, int64) (*service.AuditLog, error) {
	return nil, nil
}

func (r *auditMiddlewareCaptureRepo) Count(context.Context) (int64, error) {
	return 0, nil
}

func (r *auditMiddlewareCaptureRepo) TruncateAll(context.Context) error {
	return nil
}

func (r *auditMiddlewareCaptureRepo) DeleteBefore(context.Context, time.Time, int) (int64, error) {
	return 0, nil
}

func TestDeriveAuditAction(t *testing.T) {
	cases := []struct {
		method string
		path   string
		want   string
	}{
		{"PUT", "/api/v1/admin/accounts/:id", "admin.accounts.update"},
		{"POST", "/api/v1/admin/accounts", "admin.accounts.create"},
		{"DELETE", "/api/v1/admin/backups/:id", "admin.backups.delete"},
		{"GET", "/api/v1/admin/users/:id/api-keys", "admin.users.api_keys.read"},
		{"POST", "/api/v1/admin/redeem-codes/batch", "admin.redeem_codes.batch.create"},
	}
	for _, tc := range cases {
		if got := deriveAuditAction(tc.method, tc.path); got != tc.want {
			t.Fatalf("deriveAuditAction(%q, %q) = %q, want %q", tc.method, tc.path, got, tc.want)
		}
	}
}

func TestAuditLogMiddlewareOmitsChatPromptBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	repo := &auditMiddlewareCaptureRepo{}
	auditService := service.NewAuditLogService(repo, nil)
	auditService.Start()

	router := gin.New()
	router.POST("/api/v1/chat/completions", gin.HandlerFunc(NewAuditLogMiddleware(auditService)), func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	const secretPrompt = "prompt-that-must-never-reach-audit-storage"
	req := httptest.NewRequest(http.MethodPost, "/api/v1/chat/completions", strings.NewReader(`{"messages":[{"role":"user","content":"`+secretPrompt+`"}]}`))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	auditService.Stop()

	if resp.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", resp.Code, http.StatusNoContent)
	}
	if len(repo.logs) != 1 {
		t.Fatalf("captured logs = %d, want 1", len(repo.logs))
	}
	if got := repo.logs[0].RequestBody; got != "<sensitive chat body omitted>" {
		t.Fatalf("request body = %q, want sensitive chat body omission", got)
	}
}
