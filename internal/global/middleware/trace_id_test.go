package middleware

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/WoodrowDy/go-gin-boiler/internal/global/requestctx"
)

var uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestTraceID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name     string
		incoming string
		wantSame bool // true 면 들어온 값을 그대로 써야 한다
	}{
		{name: "헤더가 없으면 새로 만든다", incoming: "", wantSame: false},
		{name: "들어온 값을 그대로 쓴다", incoming: "gateway-abc-123", wantSame: true},
		{name: "너무 길면 새로 만든다", incoming: strings.Repeat("a", maxTraceIDLength+1), wantSame: false},
		{name: "공백이 있으면 새로 만든다", incoming: "abc 123", wantSame: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var inHandler string

			r := gin.New()
			r.Use(TraceID())
			r.GET("/", func(c *gin.Context) {
				inHandler = requestctx.TraceID(c.Request.Context())
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.incoming != "" {
				req.Header.Set(HeaderRequestID, tt.incoming)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			inHeader := w.Header().Get(HeaderRequestID)

			if inHandler == "" {
				t.Fatal("핸들러의 ctx 에 traceId 가 없다")
			}
			if inHandler != inHeader {
				t.Errorf("ctx 의 값(%q)과 응답 헤더(%q)가 다르다", inHandler, inHeader)
			}
			if tt.wantSame && inHeader != tt.incoming {
				t.Errorf("응답 헤더 = %q, want 들어온 값 %q", inHeader, tt.incoming)
			}
			if !tt.wantSame && !uuidV4.MatchString(inHeader) {
				t.Errorf("응답 헤더 = %q, want UUID v4", inHeader)
			}
		})
	}
}
