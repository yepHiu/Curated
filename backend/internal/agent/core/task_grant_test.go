package core

import (
	"context"
	"encoding/json"
	"testing"
)

// TestUserTagTaskGrantIsNarrow 验证缺少授权、伪造范围、通道及只读状态均不能写。
func TestUserTagTaskGrantIsNarrow(t *testing.T) {
	for _, tc := range []struct {
		name, grant, job, channel string
		readOnly                  bool
		allowed                   bool
	}{
		{"missing", "", "job", ChannelAction, false, false},
		{"wrong-job", "other", "job", ChannelAction, false, false},
		{"chat", "job", "job", ChannelChat, false, false},
		{"mcp", "job", "job", ChannelMCP, false, false},
		{"readonly", "job", "job", ChannelAction, true, false},
		{"worker", "job", "job", ChannelAction, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// 各用例拥有独立注册表和计数，防止测试间授权泄漏。
			calls := 0
			reg := NewRegistry()
			if err := reg.Register(ToolDefinition{Name: UserTagTaskApplyName, Permission: PermissionUserTagTask, ParamsSchema: Schema{Type: "object"}, Handler: func(context.Context, Call) (Result, error) { /* 仅计数已通过网关的写入。 */
				calls++
				return Result{OK: true}, nil
			}}); err != nil {
				t.Fatal(err)
			}
			gw := NewGateway(reg, nil, nil, func() Settings { /* 模拟实时只读配置。 */ return Settings{ReadOnly: tc.readOnly} })
			ctx := context.Background()
			if tc.grant != "" {
				ctx = WithUserTagTaskGrant(ctx, tc.grant)
			}
			r := gw.Invoke(ctx, Call{Name: UserTagTaskApplyName, Args: json.RawMessage(`{}`), SessionID: tc.job, Channel: tc.channel})
			if (calls == 1) != tc.allowed || r.OK != tc.allowed {
				t.Fatalf("calls=%d result=%+v", calls, r)
			}
			if tc.allowed {
				r = gw.Invoke(ctx, Call{Name: UserTagTaskApplyName, Args: json.RawMessage(`{"metadataTags":["bad"]}`), SessionID: tc.job, Channel: tc.channel})
				if r.OK || calls != 1 {
					t.Fatal("NFO field accepted")
				}
			}
		})
	}
}
