package core

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

// TestSubmitAnswerRejectsLiteralFactsAndMissingFields 验证 Submit Answer Rejects Literal Facts And Missing Fields 的行为与失败边界，使用隔离测试数据。
func TestSubmitAnswerRejectsLiteralFactsAndMissingFields(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Register(SubmitAnswerTool())
	store := NewAnswerRefStore()
	h := store.Capture("search_movies", answerResult(map[string]any{"id": "m1", "code": "TEST-101", "title": "Recorded"}))
	ctx := WithAnswerRefs(context.Background(), store)
	gw := NewGateway(reg, nil, nil, nil)
	for _, args := range []string{
		fmt.Sprintf(`{"items":[{"refId":%q,"code":"FAKE-999"}]}`, h[0].RefID),
		fmt.Sprintf(`{"items":[{"refId":%q,"fields":["actors"]}]}`, h[0].RefID),
		`{"items":[{"refId":"old_reference"}]}`,
		`{"items":[]}`,
	} {
		r := gw.Invoke(ctx, Call{Name: SubmitAnswerName, Args: json.RawMessage(args)})
		if r.OK || r.Answer != nil {
			t.Fatal(r)
		}
	}
	valid := fmt.Sprintf(`{"items":[{"refId":%q}]}`, h[0].RefID)
	r := gw.Invoke(ctx, Call{Name: SubmitAnswerName, Args: json.RawMessage(valid)})
	if !r.OK || r.Answer == nil || r.Answer.Items[0].Ref.Fields["code"] != "TEST-101" {
		t.Fatal(r)
	}
}
