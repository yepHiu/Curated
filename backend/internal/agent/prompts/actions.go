package prompts

import (
	"strings"
)

const (
	ActionPolishComment     = "polish_comment"
	ActionTranslateSummary  = "translate_summary"
	ActionTranslateTitle    = "translate_title"
	ActionInsightsNarrative = "insights_narrative"
)

// IsCommentAction 判断动作是否为笔记润色。
func IsCommentAction(name string) bool {
	return strings.TrimSpace(name) == ActionPolishComment
}

// IsDisplayAction 判断动作是否为展示文本翻译。
func IsDisplayAction(name string) bool {
	switch strings.TrimSpace(name) {
	case ActionTranslateSummary, ActionTranslateTitle:
		return true
	default:
		return false
	}
}

// IsInsightsAction 判断动作是否为个人洞察说明。
func IsInsightsAction(name string) bool {
	return strings.TrimSpace(name) == ActionInsightsNarrative
}

// KnownAction 限定受支持的独立 AI 动作。
func KnownAction(name string) bool {
	return IsCommentAction(name) || IsDisplayAction(name) || IsInsightsAction(name)
}

// CommentActionPrompt returns reusable instructions; source data is sent once in user.
func CommentActionPrompt() string {
	return renderPrompt("polish-comment.txt", nil)
}

// FormatCommentActionUser sends the note once as user source data.
func FormatCommentActionUser(body string) string {
	return FormatPlainUser("Note", body)
}

// TranslateDisplayPrompt 渲染展示文本翻译 TXT，空语言与类型沿用原默认值。
func TranslateDisplayPrompt(kind, locale string) string {
	target := strings.TrimSpace(locale)
	if target == "" {
		target = "zh-CN"
	}
	label := strings.TrimSpace(kind)
	if label == "" {
		label = "text"
	}
	return renderPrompt("translate-display.txt", map[string]string{"Kind": label, "Locale": target})
}

// TranslateTitlePrompt 为标题翻译指定展示类型。
func TranslateTitlePrompt(locale string) string {
	return TranslateDisplayPrompt("title", locale)
}

// TranslateSummaryPrompt 为简介翻译指定展示类型。
func TranslateSummaryPrompt(locale string) string {
	return TranslateDisplayPrompt("synopsis", locale)
}

// InsightsNarrativePrompt 渲染影片个人洞察 TXT，仅使用给定统计资料。
func InsightsNarrativePrompt(locale string) string {
	lang := strings.TrimSpace(locale)
	if lang == "" {
		lang = "zh-CN"
	}
	return renderPrompt("insights-narrative.txt", map[string]string{"Locale": lang})
}

// FormatPlainUser renders source data once without interpreting its template syntax.
func FormatPlainUser(kind, body string) string {
	return renderPrompt("action-source.txt", map[string]string{"Kind": kind, "Source": body})
}
