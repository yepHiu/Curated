package core

import "context"

const PermissionUserTagTask = "user-tag-task"
const UserTagTaskApplyName = "apply_organization_user_tags"

type userTagTaskGrantKey struct{}

// WithUserTagTaskGrant 仅由服务端 worker 持有；模型参数不能制造范围授权。
func WithUserTagTaskGrant(ctx context.Context, jobID string) context.Context {
	return context.WithValue(ctx, userTagTaskGrantKey{}, jobID)
}

// hasUserTagTaskGrant 验证工具、通道与任务范围身份，不扩展普通写工具权限。
func hasUserTagTaskGrant(ctx context.Context, call Call) bool {
	jobID, _ := ctx.Value(userTagTaskGrantKey{}).(string)
	return jobID != "" && jobID == call.SessionID && call.Name == UserTagTaskApplyName && call.Channel == ChannelAction
}
