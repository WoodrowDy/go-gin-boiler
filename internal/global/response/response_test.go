package response

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/WoodrowDy/go-gin-boiler/internal/global/requestctx"
)

type sample struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// toMap 은 응답을 JSON 으로 만들었다가 다시 풀어, 실제로 나가는 모양을 본다.
func toMap(t *testing.T, value any) map[string]any {
	t.Helper()

	raw, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("JSON 으로 만들지 못했다: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("JSON 을 풀지 못했다: %v", err)
	}

	return decoded
}

func TestObject(t *testing.T) {
	ctx := requestctx.WithTraceID(context.Background(), "abc-123")

	got := toMap(t, Object(ctx, sample{ID: 1, Name: "홍길동"}))

	row, _ := got["row"].(map[string]any)
	if row["id"] != float64(1) || row["name"] != "홍길동" {
		t.Errorf("row = %v", got["row"])
	}

	meta, _ := got["meta"].(map[string]any)
	if meta["traceId"] != "abc-123" {
		t.Errorf("meta.traceId = %v, want abc-123", meta["traceId"])
	}

	timestamp, _ := meta["timestamp"].(string)
	if _, err := time.Parse(timestampLayout, timestamp); err != nil {
		t.Errorf("meta.timestamp = %q, ISO 8601 UTC 가 아니다: %v", timestamp, err)
	}
}

func TestMeta_OmitsEmptyTraceID(t *testing.T) {
	got := toMap(t, NewMeta(context.Background()))

	if _, ok := got["traceId"]; ok {
		t.Errorf("traceId 가 없을 때는 필드가 없어야 한다: %v", got)
	}
}

func TestList(t *testing.T) {
	tests := []struct {
		name     string
		rows     []sample
		count    int64
		wantRows int
	}{
		{name: "행이 있으면 그대로", rows: []sample{{ID: 1}, {ID: 2}}, count: 10, wantRows: 2},
		{name: "nil 이면 빈 배열", rows: nil, count: 0, wantRows: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := toMap(t, List(context.Background(), tt.rows, tt.count))

			// null 이면 이 단언이 실패한다 — 배열이어야 한다.
			rows, ok := got["rows"].([]any)
			if !ok {
				t.Fatalf("rows 가 배열이 아니다: %v", got["rows"])
			}
			if len(rows) != tt.wantRows {
				t.Errorf("rows 길이 = %d, want %d", len(rows), tt.wantRows)
			}
			if got["count"] != float64(tt.count) {
				t.Errorf("count = %v, want %d", got["count"], tt.count)
			}
		})
	}
}

func TestError(t *testing.T) {
	ctx := requestctx.WithTraceID(context.Background(), "abc-123")

	got := toMap(t, Error(ctx, http.StatusNotFound, "사용자를 찾을 수 없습니다."))

	if got["message"] != "사용자를 찾을 수 없습니다." {
		t.Errorf("message = %v", got["message"])
	}
	if got["error"] != "Not Found" {
		t.Errorf("error = %v, want Not Found", got["error"])
	}
	if got["statusCode"] != float64(http.StatusNotFound) {
		t.Errorf("statusCode = %v, want 404", got["statusCode"])
	}
	if meta, _ := got["meta"].(map[string]any); meta["traceId"] != "abc-123" {
		t.Errorf("meta.traceId = %v, want abc-123", meta["traceId"])
	}
}
