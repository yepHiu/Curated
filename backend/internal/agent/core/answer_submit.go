package core

import (
	"context"
	"encoding/json"
)

// AnswerSubmission is server-only: it is rendered once, not sent back to the
// model for another free-form retelling of the verified facts.
type AnswerSubmission struct {
	Items []AnswerItem
	Query string
}

type AnswerItem struct {
	Ref         AnswerRef
	Fields      []string
	ReasonFacts []string
}

// SubmitAnswerTool 注册终结回答的只读工具，校验引用和字段后交给服务端渲染。
func SubmitAnswerTool() ToolDefinition {
	fields := []string{"code", "title", "actors", "runtimeMinutes", "tags", "userRating", "metadataRating", "isFavorite", "year", "studio", "releaseDate", "score", "playState"}
	fieldList := Schema{Type: "array", MaxItems: len(fields), Items: &Schema{Type: "string", Enum: fields}}
	item := Schema{Type: "object", Required: []string{"refId"}, Properties: map[string]Schema{
		"refId":  {Type: "string", MinLength: 1, MaxLength: 120},
		"fields": fieldList, "reasonFacts": fieldList,
	}}
	return ToolDefinition{
		Name: SubmitAnswerName, Permission: PermissionRead, Domain: DomainPresent,
		Description: "Finish a concrete movie answer using this request's answerRefs. Submit this tool ALONE after retrieval. Select 1-6 refIds and available field names; the server fills facts and source labels. Default fields: code,title when available. reasonFacts selects recorded facts, not a claim that user constraints match. Never supply literal titles, codes, actor values or prose. Source-page prose and old conversation are not answerRefs.",
		ParamsSchema: Schema{Type: "object", Properties: map[string]Schema{
			"items":    {Type: "array", MinItems: 1, MaxItems: PresentMoviesMaxItems, Items: &item},
			"queryRef": {Type: "string", Enum: []string{"user_input"}, Description: "Instead of items, quote the user's current query as UNVERIFIED. Does not confirm any work exists."},
		}},
		// 消费当前请求的可信记录，拒绝模型自行填入的事实。
		Handler: func(ctx context.Context, call Call) (Result, error) {
			var args struct {
				QueryRef string `json:"queryRef"`
				Items    []struct {
					RefID       string   `json:"refId"`
					Fields      []string `json:"fields"`
					ReasonFacts []string `json:"reasonFacts"`
				} `json:"items"`
			}
			_ = json.Unmarshal(call.Args, &args)
			store := AnswerRefsFromContext(ctx)
			if args.QueryRef != "" {
				if len(args.Items) != 0 || store.Query() == "" {
					return answerRejected("Use queryRef alone for the current user's unverified query."), nil
				}
				return Result{OK: true, Answer: &AnswerSubmission{Query: store.Query()}, Data: map[string]any{"quotedQuery": true}}, nil
			}
			if len(args.Items) == 0 {
				return answerRejected("Provide items or an unverified queryRef."), nil
			}
			submission := &AnswerSubmission{}
			seen := map[string]bool{}
			for _, item := range args.Items {
				ref, ok := store.Lookup(item.RefID)
				if !ok || seen[item.RefID] {
					return answerRejected("Unknown, duplicate or expired refId; retrieve a current answerRef."), nil
				}
				seen[item.RefID] = true
				if len(item.Fields) == 0 {
					for _, key := range []string{"code", "title"} {
						if _, ok := ref.Fields[key]; ok {
							item.Fields = append(item.Fields, key)
						}
					}
				}
				for _, key := range append(append([]string{}, item.Fields...), item.ReasonFacts...) {
					if _, ok := ref.Fields[key]; !ok {
						return answerRejected("Requested fact is unavailable. Use only field keys listed in answerRefs."), nil
					}
				}
				submission.Items = append(submission.Items, AnswerItem{Ref: ref, Fields: item.Fields, ReasonFacts: item.ReasonFacts})
			}
			return Result{OK: true, Answer: submission, Data: map[string]any{"submitted": true}}, nil
		},
	}
}

// answerRejected 生成不包含模型草稿的固定拒绝结果。
func answerRejected(message string) Result {
	return Result{Error: &ToolError{Code: "AI_ANSWER_REJECTED", Message: message}}
}
