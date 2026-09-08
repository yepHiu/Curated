package run

import (
	"encoding/json"
	"fmt"
	"strings"

	"curated-backend/internal/agent/core"
	"curated-backend/internal/contracts"
)

// answerCopy 按界面语言选择服务端生成的提示语。
func answerCopy(locale, en, zh, ja string) string {
	if strings.HasPrefix(locale, "zh") {
		return zh
	}
	if strings.HasPrefix(locale, "ja") {
		return ja
	}
	return en
}

// escapeAnswerValue 转义来源字段中的 Markdown、HTML 和自动链接，按数据展示。
func escapeAnswerValue(v any) string {
	var text string
	switch value := v.(type) {
	case string:
		text = value
	case []any:
		var items []string
		for _, item := range value {
			if s, ok := item.(string); ok {
				items = append(items, s)
			}
		}
		text = strings.Join(items, ", ")
	default:
		text = fmt.Sprint(v)
	}
	text = strings.Join(strings.Fields(text), " ")
	// Escape Markdown and disable HTML/autolinks from library/provider fields.
	text = strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(text, "&", "&amp;"), "<", "&lt;"), ">", "&gt;")
	return strings.NewReplacer("\\", "\\\\", "`", "\\`", "*", "\\*", "_", "\\_", "[", "\\[", "]", "\\]", "(", "\\(", ")", "\\)", "#", "\\#", "!", "\\!", "|", "\\|", ":", "&#58;").Replace(text)
}

// factLabel 返回事实字段的三语标签，未知字段不参与模型自由改写。
func factLabel(locale, key string) string {
	labels := map[string][3]string{
		"code": {"Code", "番号", "品番"}, "title": {"Title", "标题", "タイトル"}, "actors": {"Actors", "演员", "出演者"},
		"runtimeMinutes": {"Runtime (minutes)", "片长（分钟）", "収録時間（分）"}, "tags": {"Tags", "标签", "タグ"},
		"userRating": {"Your rating", "用户评分", "ユーザー評価"}, "metadataRating": {"Source rating", "来源评分", "提供元の評価"},
		"score": {"Source rating", "来源评分", "提供元の評価"}, "isFavorite": {"Favorite", "已收藏", "お気に入り"},
		"year": {"Year", "年份", "年"}, "studio": {"Studio", "片商", "メーカー"}, "releaseDate": {"Release date", "发行日期", "発売日"},
		"playState": {"Playback state", "观看状态", "視聴状態"},
	}
	label, ok := labels[key]
	if !ok {
		return key
	}
	return answerCopy(locale, label[0], label[1], label[2])
}

// renderAnswer 从同一来源快照生成正文、卡片和历史依据；用户查询单独标注未核实。
func renderAnswer(answer *core.AnswerSubmission, locale string) (string, []contracts.AIAgentMovieCardDTO, *contracts.AIAnswerEvidenceDTO) {
	if answer.Query != "" {
		return answerCopy(locale, "Your query (unverified): ", "你的查询内容（尚未核实）：", "検索内容（未確認）：") + escapeAnswerValue(answer.Query), nil, nil
	}
	var text strings.Builder
	evidence := &contracts.AIAnswerEvidenceDTO{Version: 1}
	var cards []contracts.AIAgentMovieCardDTO
	for i, item := range answer.Items {
		ref := item.Ref
		source := answerCopy(locale, "Local record", "本地记录", "ライブラリの記録")
		if ref.Source == "provider" {
			source = answerCopy(locale, "Provider record", "来源站点记录", "提供元の記録")
			if provider, ok := ref.Fields["provider"].(string); ok && provider != "" {
				source += " · " + escapeAnswerValue(provider)
			}
			if ref.MovieID == "" {
				source += " · " + answerCopy(locale, "Not in library", "尚未入库", "ライブラリ未登録")
			}
		}
		fmt.Fprintf(&text, "%d. %s\n", i+1, source)
		seen := map[string]bool{}
		displayed := map[string]any{}
		for _, key := range []string{"provider", "metadataProvider", "homepage", "inLibrary"} {
			if value, ok := ref.Fields[key]; ok {
				displayed[key] = value
			}
		}
		for _, key := range append(append([]string{}, item.Fields...), item.ReasonFacts...) {
			if seen[key] {
				continue
			}
			seen[key] = true
			v := ref.Fields[key]
			displayed[key] = v
			fmt.Fprintf(&text, "   - %s：%s\n", factLabel(locale, key), escapeAnswerValue(v))
		}
		text.WriteString("\n")
		// Only local reads produce actionable local cards; provider rows retain
		// their source identity even when the code was reconciled with the library.
		if ref.Source == "local" && ref.MovieID != "" {
			raw, _ := json.Marshal(ref.Fields)
			var card contracts.AIAgentMovieCardDTO
			_ = json.Unmarshal(raw, &card)
			card.MovieID = ref.MovieID
			cards = append(cards, card)
			for _, key := range []string{"code", "title", "actors"} {
				if v, ok := ref.Fields[key]; ok {
					displayed[key] = v
				}
			}
		}
		evidence.Items = append(evidence.Items, contracts.AIAnswerEvidenceItemDTO{RefID: ref.RefID, MovieID: ref.MovieID, Source: ref.Source, Tool: ref.Tool, RetrievedAt: ref.RetrievedAt, Truncated: ref.Truncated, Fields: displayed})
	}
	return strings.TrimSpace(text.String()), cards, evidence
}
