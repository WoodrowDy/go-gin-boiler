// Package response 는 HTTP 응답 본문의 모양을 정한다.
//
//	성공(단건)  { "row": …, "meta": { "traceId", "timestamp" } }
//	성공(목록)  { "rows": […], "count": n, "meta": { … } }
//	실패        { "message", "error", "statusCode", "meta": { … } }
//
// nest-boiler 의 ObjectResponse / ListResponse / 전역 예외 필터와 같은 모양이다.
package response

import (
	"context"
	"net/http"
	"time"

	"github.com/WoodrowDy/go-gin-boiler/internal/global/requestctx"
)

// timestampLayout 은 JS 의 Date.toISOString() 과 같은 모양이다(UTC, 밀리초 3자리, 끝에 Z).
// Go 의 시간 포맷은 "2006-01-02 15:04:05" 라는 기준 시각을 원하는 모양으로 적어서 정한다.
const timestampLayout = "2006-01-02T15:04:05.000Z"

// Meta 는 모든 응답에 붙는 공통 정보다. 성공과 실패가 같은 것을 쓴다.
type Meta struct {
	// TraceID 는 요청 추적 ID. 로그의 traceId 와 같은 값이라, 이 값으로 그 요청의 로그를 찾는다.
	// omitempty — 요청 밖에서 만든 응답처럼 값이 없으면 필드를 아예 싣지 않는다.
	TraceID string `json:"traceId,omitempty"`

	// Timestamp 는 응답을 만든 시각(ISO 8601 UTC).
	Timestamp string `json:"timestamp"`
}

// NewMeta 는 ctx 에서 traceId 를 꺼내 Meta 를 만든다.
func NewMeta(ctx context.Context) Meta {
	return Meta{
		TraceID:   requestctx.TraceID(ctx),
		Timestamp: time.Now().UTC().Format(timestampLayout),
	}
}

// ObjectResponse 는 단건 성공 응답이다.
//
// [T any] 는 타입 파라미터다. ObjectResponse[UserDetailResponse] 처럼 쓰는 쪽이 T 를 정하면,
// Row 의 타입이 컴파일 시점에 그것으로 굳는다. TS 의 ObjectResponse<T> 와 같다.
type ObjectResponse[T any] struct {
	Row  T    `json:"row"`
	Meta Meta `json:"meta"`
}

// ListResponse 는 목록 성공 응답이다. Count 는 이 페이지의 개수가 아니라 조건에 맞는 전체 개수다.
type ListResponse[T any] struct {
	Rows  []T   `json:"rows"`
	Count int64 `json:"count"`
	Meta  Meta  `json:"meta"`
}

// Object 는 단건 응답을 만든다. T 는 인자에서 추론되므로 Object(ctx, dto) 로 부르면 된다.
func Object[T any](ctx context.Context, row T) ObjectResponse[T] {
	return ObjectResponse[T]{Row: row, Meta: NewMeta(ctx)}
}

// List 는 목록 응답을 만든다.
func List[T any](ctx context.Context, rows []T, count int64) ListResponse[T] {
	// nil 슬라이스는 JSON 에서 null 이 된다. 받는 쪽이 배열로 믿고 순회하다 터지지 않게 [] 로 바꾼다.
	if rows == nil {
		rows = []T{}
	}

	return ListResponse[T]{Rows: rows, Count: count, Meta: NewMeta(ctx)}
}

// ErrorResponse 는 실패 응답이다.
type ErrorResponse struct {
	// Message 는 사람이 읽는 설명. 도메인의 constants.go 에서 온다.
	Message string `json:"message"`

	// Error 는 상태 코드의 이름("Not Found", "Conflict").
	Error string `json:"error"`

	StatusCode int  `json:"statusCode"`
	Meta       Meta `json:"meta"`
}

// Error 는 실패 응답을 만든다.
func Error(ctx context.Context, status int, message string) ErrorResponse {
	return ErrorResponse{
		Message:    message,
		Error:      http.StatusText(status),
		StatusCode: status,
		Meta:       NewMeta(ctx),
	}
}
