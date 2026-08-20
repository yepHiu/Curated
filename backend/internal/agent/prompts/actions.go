package prompts

import (
	"fmt"
	"strings"
)

const (
	ActionPolishComment     = "polish_comment"
	ActionCleanSummary      = "clean_summary"
	ActionTranslateTitle    = "translate_title"
	ActionInsightsNarrative = "insights_narrative"
)

func IsCommentAction(name string) bool {
	return strings.TrimSpace(name) == ActionPolishComment
}

func IsDisplayAction(name string) bool {
	switch strings.TrimSpace(name) {
	case ActionCleanSummary, ActionTranslateTitle:
		return true
	default:
		return false
	}
}

func IsInsightsAction(name string) bool {
	return strings.TrimSpace(name) == ActionInsightsNarrative
}

func KnownAction(name string) bool {
	return IsCommentAction(name) || IsDisplayAction(name) || IsInsightsAction(name)
}

func CommentActionPrompt(body string) string {
	var b strings.Builder
	b.WriteString("You rewrite a user's private movie note for Curated. ")
	b.WriteString("Return only the rewritten note text. No title, no quotes, no markdown fences. ")
	b.WriteString("Do not invent facts, titles, actors, scores, or plot that the source note does not contain. ")
	b.WriteString("Detect the language of the source note yourself and keep the rewrite in that same language. ")
	b.WriteString("Do not translate into another language. Polish the wording. Keep length similar.\n")
	b.WriteString("<source>\n")
	b.WriteString(body)
	b.WriteString("\n</source>")
	return b.String()
}

func FormatCommentActionUser(body string) string {
	return fmt.Sprintf("Note:\n%s", body)
}

func CleanSummaryPrompt(summary string) string {
	var b strings.Builder
	b.WriteString("You clean a scraped movie synopsis for Curated. ")
	b.WriteString("Return only the cleaned synopsis. No title, no quotes, no markdown fences. ")
	b.WriteString("Remove ads, watermarks, site signatures, download links, and promotional filler. ")
	b.WriteString("Keep the plot description. Do not invent plot, actors, or facts that are not in the source. ")
	b.WriteString("Keep the source language. If the synopsis is already clean, return it unchanged.\n")
	b.WriteString("<source>\n")
	b.WriteString(summary)
	b.WriteString("\n</source>")
	return b.String()
}

func TranslateTitlePrompt(title, locale string) string {
	target := strings.TrimSpace(locale)
	if target == "" {
		target = "zh-CN"
	}
	var b strings.Builder
	b.WriteString("You localize a movie display title for Curated. ")
	b.WriteString("Return only the localized title text. No quotes, no markdown fences, no extra commentary. ")
	b.WriteString("Target language: ")
	b.WriteString(target)
	b.WriteString(". Keep the meaning. Do not add codes, actor names, or marketing copy that the source title does not contain. ")
	b.WriteString("If the title is already in the target language, return it unchanged.\n")
	b.WriteString("<source>\n")
	b.WriteString(title)
	b.WriteString("\n</source>")
	return b.String()
}

func InsightsNarrativePrompt(locale, payload string) string {
	lang := strings.TrimSpace(locale)
	if lang == "" {
		lang = "zh-CN"
	}
	var b strings.Builder
	b.WriteString("You write a short personal-insights readout for Curated. ")
	b.WriteString("Use only numbers and names inside the <source> JSON. Never invent counts, durations, ratings, or people. ")
	b.WriteString("Write 1-2 short paragraphs plus at most two actionable hints. ")
	b.WriteString("State that actor/tag shares use full-per-entity attribution and may sum over 100%. ")
	b.WriteString("If a rate is null, say the denominator is empty instead of 0%. ")
	b.WriteString("Answer in ")
	b.WriteString(lang)
	b.WriteString(".\n")
	b.WriteString("<source>\n")
	b.WriteString(payload)
	b.WriteString("\n</source>")
	return b.String()
}

func FormatPlainUser(kind, body string) string {
	return fmt.Sprintf("%s:\n%s", kind, body)
}
