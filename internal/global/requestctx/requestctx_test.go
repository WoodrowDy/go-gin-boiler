package requestctx

import (
	"context"
	"testing"
)

func TestTraceID(t *testing.T) {
	tests := []struct {
		name string
		ctx  context.Context
		want string
	}{
		{name: "실린 값을 꺼낸다", ctx: WithTraceID(context.Background(), "abc-123"), want: "abc-123"},
		{name: "없으면 빈 문자열", ctx: context.Background(), want: ""},
		{
			name: "나중에 실은 값이 이긴다",
			ctx:  WithTraceID(WithTraceID(context.Background(), "first"), "second"),
			want: "second",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TraceID(tt.ctx); got != tt.want {
				t.Errorf("TraceID() = %q, want %q", got, tt.want)
			}
		})
	}
}
