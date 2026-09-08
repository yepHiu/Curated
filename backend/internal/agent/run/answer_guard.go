package run

import (
	"encoding/json"
	"html"
	"regexp"
	"strings"
	"unicode"

	"curated-backend/internal/agent/core"
	"golang.org/x/text/unicode/norm"
)

const maxAnswerBytes = 64 * 1024

// This detector is a supplementary guard, not a proof of arbitrary prose.
// Concrete movie facts are published only by the structured renderer.
var catalogCodePattern = regexp.MustCompile(`(?i)\b(?:fc2[- _]?(?:ppv[- _]?)?[0-9]{3,10}|[a-z]{2,12}[- _]?[0-9]{2,8}(?:[- _](?:cd)?[0-9]{1,4})?|[0-9]{6}[- _][0-9]{2,4})\b`)
var answerLinkPattern = regexp.MustCompile(`(?i)(?:https?://|www\.|\]\s*\(|<\s*/?\s*[a-z]|ref_[a-z0-9_]+|/api/library/movies/)`)

// normalizedAnswerText 统一全角、HTML 实体与常见排版形式，避免简单格式绕过检查。
func normalizedAnswerText(text string) string {
	text = norm.NFKC.String(html.UnescapeString(text))
	// 逐字符统一用于检测的文本；保留原始资料供最终展示。
	return strings.Map(func(r rune) rune {
		if unicode.Is(unicode.Cf, r) || r == '*' || r == '`' || r == '\\' {
			return -1
		}
		switch r {
		case '‐', '‑', '‒', '–', '—', '−':
			return '-'
		}
		return unicode.ToLower(r)
	}, text)
}

// canonicalCode 为校验比较移除编号分隔符，不据此合并不同来源记录。
func canonicalCode(text string) string {
	// 逐字符统一用于检测的文本；保留原始资料供最终展示。
	return strings.Map(func(r rune) rune {
		if r == '-' || r == '_' || unicode.IsSpace(r) {
			return -1
		}
		return r
	}, normalizedAnswerText(text))
}

// proseNeedsReferences 检查自由文字是否需要改用结构化作品引用，不代替语义事实验证。
func proseNeedsReferences(text string, refs *core.AnswerRefStore) bool {
	if len(text) > maxAnswerBytes {
		return true
	}
	n := normalizedAnswerText(text)
	if catalogCodePattern.MatchString(n) || answerLinkPattern.MatchString(n) {
		return true
	}
	for _, ref := range refs.All() {
		for _, key := range []string{"code", "title"} {
			value, _ := ref.Fields[key].(string)
			value = normalizedAnswerText(strings.TrimSpace(value))
			if value != "" && (key == "code" || len([]rune(value)) >= 4) && strings.Contains(n, value) {
				return true
			}
		}
	}
	return false
}

// A user's exact text may be kept in a write draft; that does not make it a
// verified movie. Newly introduced code-shaped strings require a read record.
func writeHasUnsupportedCodes(args json.RawMessage, refs *core.AnswerRefStore, userText string) bool {
	allowed := map[string]bool{}
	for _, code := range catalogCodePattern.FindAllString(normalizedAnswerText(userText), -1) {
		allowed[canonicalCode(code)] = true
	}
	for _, ref := range refs.All() {
		if code, ok := ref.Fields["code"].(string); ok {
			allowed[canonicalCode(code)] = true
		}
	}
	var values any
	if json.Unmarshal(args, &values) != nil {
		return true
	}
	var unsupported func(any) bool
	// 递归检查参数中的字符串，避免嵌套预览字段绕过番号检查。
	unsupported = func(v any) bool {
		switch value := v.(type) {
		case string:
			for _, code := range catalogCodePattern.FindAllString(normalizedAnswerText(value), -1) {
				if !allowed[canonicalCode(code)] {
					return true
				}
			}
		case map[string]any:
			for key, child := range value {
				// A local foreign key may itself look like a catalog code. The
				// write handler validates its target; it is not generated prose.
				if key == "movieId" {
					continue
				}
				if unsupported(child) {
					return true
				}
			}
		case []any:
			for _, child := range value {
				if unsupported(child) {
					return true
				}
			}
		}
		return false
	}
	return unsupported(values)
}

// rejectedAnswerResult 向模型返回固定的修正说明，避免把错误草稿发送给用户。
func rejectedAnswerResult() core.Result {
	return core.Result{Error: &core.ToolError{Code: "AI_ANSWER_REJECTED", Message: "Answer not published. Use current answerRefs with submit_answer alone; do not introduce unsupported movie facts."}}
}
