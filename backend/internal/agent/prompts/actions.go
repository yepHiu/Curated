package prompts

import (
	"fmt"
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

// CommentActionPrompt 渲染笔记润色 TXT，保留当前源资料放置与消息行为。
func CommentActionPrompt(body string) string {
	return renderPrompt("polish-comment.txt", map[string]string{"Source": body})
}

// FormatCommentActionUser 保留笔记动作现有的用户消息格式。
func FormatCommentActionUser(body string) string {
	return fmt.Sprintf("Note:\n%s", body)
}

// TranslateDisplayPrompt 渲染展示文本翻译 TXT，空语言与类型沿用原默认值。
func TranslateDisplayPrompt(kind, source, locale string) string {
	target := strings.TrimSpace(locale)
	if target == "" {
		target = "zh-CN"
	}
	label := strings.TrimSpace(kind)
	if label == "" {
		label = "text"
	}
	return renderPrompt("translate-display.txt", map[string]string{"Kind": label, "Locale": target, "Source": source})
}

// TranslateTitlePrompt 为标题翻译指定展示类型。
func TranslateTitlePrompt(title, locale string) string {
	return TranslateDisplayPrompt("title", title, locale)
}

// TranslateSummaryPrompt 为简介翻译指定展示类型。
func TranslateSummaryPrompt(summary, locale string) string {
	return TranslateDisplayPrompt("synopsis", summary, locale)
}

// InsightsNarrativePrompt 渲染影片个人洞察 TXT，仅使用给定统计资料。
func InsightsNarrativePrompt(locale, payload string) string {
	lang := strings.TrimSpace(locale)
	if lang == "" {
		lang = "zh-CN"
	}
	return renderPrompt("insights-narrative.txt", map[string]string{"Locale": lang, "Source": payload})
}

// FormatPlainUser 保留独立动作的原始用户消息格式。
func FormatPlainUser(kind, body string) string {
	return fmt.Sprintf("%s:\n%s", kind, body)
}
