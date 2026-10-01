package prompts

import (
	"bytes"
	"embed"
	"fmt"
	"strings"
	"text/template"
)

// promptFiles 将独立 TXT 正文随二进制交付，运行时不依赖工作目录或外部文件。
//
//go:embed *.txt
var promptFiles embed.FS

// promptTemplates 只在初始化时解析；执行时不修改模板，供并发 AI 请求复用。
var promptTemplates = loadPromptTemplates()

// loadPromptTemplates 校验内置模板语法；去掉文件末尾一个换行，保留正文有意的空白。
func loadPromptTemplates() *template.Template {
	set := template.New("prompts").Option("missingkey=error")
	entries, err := promptFiles.ReadDir(".")
	if err != nil {
		panic(fmt.Errorf("read embedded prompts: %w", err))
	}
	for _, entry := range entries {
		body, err := promptFiles.ReadFile(entry.Name())
		if err != nil {
			panic(fmt.Errorf("read prompt %s: %w", entry.Name(), err))
		}
		if _, err := set.New(entry.Name()).Parse(strings.TrimSuffix(string(body), "\n")); err != nil {
			panic(fmt.Errorf("parse prompt %s: %w", entry.Name(), err))
		}
	}
	return set
}

// renderPrompt 仅渲染受控模板一次；源资料中的模板符号保持原文，不再次求值。
// 模板名与变量由本包固定提供，缺失表示程序或内置资源错误，不能静默发送残缺提示词。
func renderPrompt(name string, values map[string]string) string {
	var out bytes.Buffer
	if err := promptTemplates.ExecuteTemplate(&out, name, values); err != nil {
		panic(fmt.Errorf("render prompt %s: %w", name, err))
	}
	return out.String()
}

// Definition 将提示词正文与既有审计版本一起传递，调用方不再比较正文来判断用途。
type Definition struct {
	Text    string
	Version string
}

// TopicVocabularyPrompt 返回题材词汇归纳模板及其既有版本。
func TopicVocabularyPrompt() Definition {
	return Definition{Text: renderPrompt("topic-vocabulary.txt", nil), Version: "topic-vocabulary-v3"}
}

// TopicClassificationPrompt 返回影片分类模板及其既有版本。
func TopicClassificationPrompt() Definition {
	return Definition{Text: renderPrompt("topic-classification.txt", nil), Version: "topic-classification-v1"}
}

// MemoryCheckpointPrompt 返回对话记忆摘要规则，输入资料仍由运行层单独传递。
func MemoryCheckpointPrompt() string {
	return renderPrompt("memory-checkpoint.txt", nil)
}

// AnswerCorrectionPrompt 返回回答校验失败后唯一一次纠正所用的规则。
func AnswerCorrectionPrompt() string {
	return renderPrompt("answer-correction.txt", nil)
}

// HistoryOmittedPrompt 返回历史因预算被截断时的补充说明。
func HistoryOmittedPrompt() string {
	return renderPrompt("history-omitted.txt", nil)
}

// ProviderProbePrompt 返回连接探针的最小输出指令。
func ProviderProbePrompt() string {
	return renderPrompt("provider-probe.txt", nil)
}
