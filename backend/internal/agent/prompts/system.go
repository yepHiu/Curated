package prompts

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
)

const Version = "agent-system-v8"
const maxMentions = 8
const maxMentionLabelRunes = 80

// systemPromptTemplate 是独立 TXT 的静态正文，动态页面数据由下方 Go 投影提供。
var systemPromptTemplate = renderPrompt("system.txt", nil)

// SystemPrompt is the versioned base + safety instructions for the experimental agent.
func SystemPrompt(locale string) string {
	var b strings.Builder
	b.WriteString(strings.TrimSpace(systemPromptTemplate))
	b.WriteString("\n\n")
	b.WriteString(renderPrompt("response-language.txt", map[string]string{"Locale": strings.TrimSpace(locale)}))
	return b.String()
}

// PageContextPrompt projects only this request's page data. Keep it after shared
// rules and history so navigation does not invalidate their cacheable prefix.
func PageContextPrompt(page *contracts.AIChatContext) string {
	page = core.MoviePageContext(page)
	var b strings.Builder
	if page != nil {
		parts := make([]string, 0, 7)
		if page.Route != "" {
			parts = append(parts, "route="+page.Route)
		}
		if page.MovieID != "" {
			parts = append(parts, "movieId="+page.MovieID)
		}
		if page.ComicID != "" {
			parts = append(parts, "comicId="+page.ComicID)
		}
		if page.PhotoID != "" {
			parts = append(parts, "photoId="+page.PhotoID)
		}
		if page.ActorName != "" {
			parts = append(parts, "actor="+page.ActorName)
		}
		if page.Query != "" {
			parts = append(parts, "query="+page.Query)
		}
		if len(page.SelectedMovieIDs) > 0 {
			parts = append(parts, "selectedMovieIds="+strings.Join(page.SelectedMovieIDs, ","))
		}
		if len(page.SelectedActors) > 0 {
			parts = append(parts, "selectedActors="+strings.Join(page.SelectedActors, ","))
		}
		if len(page.SelectedComicIDs) > 0 {
			parts = append(parts, "selectedComicIds="+strings.Join(page.SelectedComicIDs, ","))
		}
		if len(page.SelectedPhotoIDs) > 0 {
			parts = append(parts, "selectedPhotoIds="+strings.Join(page.SelectedPhotoIDs, ","))
		}
		if len(parts) > 0 {
			b.WriteString(renderPrompt("page-context.txt", map[string]string{"Context": strings.Join(parts, "; ")}))
		}
		writeActiveFilters(&b, page.ActiveFilters)
		writeMentions(&b, page.Mentions)
	}
	return b.String()
}

// writeActiveFilters 仅投影白名单筛选，再填入独立 TXT 上下文模板。
func writeActiveFilters(b *strings.Builder, filters *contracts.AIChatActiveFilters) {
	if filters == nil {
		return
	}
	parts := make([]string, 0, 5)
	if filters.Query != "" {
		parts = append(parts, "query="+filters.Query)
	}
	if filters.Tag != "" {
		parts = append(parts, "tag="+filters.Tag)
	}
	if filters.Actor != "" {
		parts = append(parts, "actor="+filters.Actor)
	}
	if filters.PlayState != "" {
		parts = append(parts, "playState="+filters.PlayState)
	}
	if filters.Runtime != "" {
		parts = append(parts, "runtime="+filters.Runtime)
	}
	if filters.Favorite != nil {
		if *filters.Favorite {
			parts = append(parts, "favorite=true")
		} else {
			parts = append(parts, "favorite=false")
		}
	}
	if filters.ReadStatus != "" {
		parts = append(parts, "readStatus="+filters.ReadStatus)
	}
	if len(parts) == 0 {
		return
	}
	b.WriteString(renderPrompt("active-filters.txt", map[string]string{"Source": strings.Join(parts, "\n")}))
}

// writeMentions 限制提及类型、数量与长度，再填入独立 TXT 模板。
func writeMentions(b *strings.Builder, mentions []contracts.AIChatMention) {
	if len(mentions) == 0 {
		return
	}
	count := 0
	var lines []string
	for _, mention := range mentions {
		kind := strings.ToLower(strings.TrimSpace(mention.Kind))
		if kind != "movie" && kind != "actor" && kind != "tag" && kind != "comic" && kind != "photo" {
			continue
		}
		id := strings.TrimSpace(mention.ID)
		label := clipRunes(strings.TrimSpace(mention.Label), maxMentionLabelRunes)
		if id == "" && label == "" {
			continue
		}
		count++
		if count > maxMentions {
			break
		}
		lines = append(lines, fmt.Sprintf("%s id=%s label=%s", kind, id, label))
	}
	if len(lines) == 0 {
		return
	}
	b.WriteString(renderPrompt("mentions.txt", map[string]string{"Source": strings.Join(lines, "\n")}))
}

// clipRunes 按 Unicode 字符截短标签，避免截断 UTF-8 编码。
func clipRunes(value string, max int) string {
	if utf8.RuneCountInString(value) <= max {
		return value
	}
	return string([]rune(value)[:max])
}
