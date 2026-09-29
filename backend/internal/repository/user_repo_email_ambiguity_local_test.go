package repository

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

// 本地定制用例：GetByEmail 在「规范化邮箱匹配到多行」时必须返回哨兵错误
// service.ErrUserEmailAmbiguous，而不是裸 fmt.Errorf。
//
// 依据：按邮箱写入的等级指派（user_tier_service.go resolveUserByEmail）用
// errors.Is 把「多行匹配（必须人工核对）」与「数据库读取失败（必须原样上抛）」
// 分开处置；若这里退回裸错误，该区分失效，基础设施故障会被误报成重复用户。
//
// 上游 main 的同场景用例（TestUserRepositoryGetByEmailReportsNormalizedEmailConflict）
// 只断言错误文案、不覆盖错误类型，故本地单列一例锁定 sentinel 语义，
// 不改动上游测试文件。
func TestUserRepositoryGetByEmailReturnsAmbiguousSentinel(t *testing.T) {
	repo, client := newUserEntRepo(t)
	ctx := context.Background()

	// 两个用户经 LOWER(TRIM(email)) 规范化后同为一个邮箱；另加一个单行对照。
	for _, spec := range []struct{ email, username string }{
		{"Conflict@Example.com", "conflict-user-1"},
		{" conflict@example.com ", "conflict-user-2"},
		{"single-user@example.com", "single-user"},
	} {
		_, err := client.User.Create().
			SetEmail(spec.email).
			SetUsername(spec.username).
			SetPasswordHash("hash").
			SetRole(service.RoleUser).
			SetStatus(service.StatusActive).
			Save(ctx)
		require.NoError(t, err)
	}

	_, err := repo.GetByEmail(ctx, "conflict@example.com")
	require.Error(t, err)
	require.ErrorIs(t, err, service.ErrUserEmailAmbiguous)

	// 对照：单行匹配不得被判为歧义
	single, err := repo.GetByEmail(ctx, "single-user@example.com")
	require.NoError(t, err)
	require.Equal(t, "single-user", single.Username)
}
