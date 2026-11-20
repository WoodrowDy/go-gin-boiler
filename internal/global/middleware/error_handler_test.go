package middleware

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/WoodrowDy/go-gin-boiler/internal/global/apperror"
)

func newErrorTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(TraceID(), AccessLog(AccessLogOptions{}), Recovery(), ErrorHandler())
	r.NoRoute(NoRoute())

	r.GET("/ok", func(c *gin.Context) { c.JSON(http.StatusOK, map[string]string{"message": "ok"}) })
	r.GET("/not-found", func(c *gin.Context) {
		_ = c.Error(apperror.NotFound("사용자를 찾을 수 없습니다."))
	})
	r.GET("/wrapped", func(c *gin.Context) {
		_ = c.Error(fmt.Errorf("get user: %w", apperror.Conflict("이미 존재하는 이메일입니다.")))
	})
	r.GET("/unknown", func(c *gin.Context) {
		_ = c.Error(errors.New(`pq: relation "users" does not exist`))
	})
	r.GET("/panic", func(_ *gin.Context) { panic("boom") })

	return r
}

func TestErrorHandler(t *testing.T) {
	tests := []struct {
		name        string
		path        string
		wantStatus  int
		wantMessage string
		wantError   string
	}{
		{
			name: "apperror 는 그 상태와 메시지로", path: "/not-found",
			wantStatus: http.StatusNotFound, wantMessage: "사용자를 찾을 수 없습니다.", wantError: "Not Found",
		},
		{
			name: "감싼 apperror 도 찾는다", path: "/wrapped",
			wantStatus: http.StatusConflict, wantMessage: "이미 존재하는 이메일입니다.", wantError: "Conflict",
		},
		{
			name: "모르는 에러는 500 과 정해진 문구", path: "/unknown",
			wantStatus: http.StatusInternalServerError, wantMessage: messageInternalError, wantError: "Internal Server Error",
		},
		{
			name: "panic 도 같은 모양", path: "/panic",
			wantStatus: http.StatusInternalServerError, wantMessage: messageInternalError, wantError: "Internal Server Error",
		},
		{
			name: "없는 경로도 같은 모양", path: "/nope",
			wantStatus: http.StatusNotFound, wantMessage: "Cannot GET /nope", wantError: "Not Found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			captureLogs(t)

			w := httptest.NewRecorder()
			newErrorTestRouter().ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if w.Code != tt.wantStatus {
				t.Fatalf("상태 코드 = %d, want %d\n본문: %s", w.Code, tt.wantStatus, w.Body.String())
			}

			var body struct {
				Message    string `json:"message"`
				Error      string `json:"error"`
				StatusCode int    `json:"statusCode"`
				Meta       struct {
					TraceID   string `json:"traceId"`
					Timestamp string `json:"timestamp"`
				} `json:"meta"`
			}
			if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
				t.Fatalf("본문이 JSON 이 아니다: %v\n본문: %s", err, w.Body.String())
			}

			if body.Message != tt.wantMessage {
				t.Errorf("message = %q, want %q", body.Message, tt.wantMessage)
			}
			if body.Error != tt.wantError {
				t.Errorf("error = %q, want %q", body.Error, tt.wantError)
			}
			if body.StatusCode != tt.wantStatus {
				t.Errorf("statusCode = %d, want %d", body.StatusCode, tt.wantStatus)
			}
			if body.Meta.TraceID == "" || body.Meta.TraceID != w.Header().Get(HeaderRequestID) {
				t.Errorf("meta.traceId(%q)가 응답 헤더(%q)와 다르다", body.Meta.TraceID, w.Header().Get(HeaderRequestID))
			}
			if body.Meta.Timestamp == "" {
				t.Error("meta.timestamp 가 비어 있다")
			}
		})
	}
}

func TestErrorHandler_SuccessUntouched(t *testing.T) {
	w := httptest.NewRecorder()
	newErrorTestRouter().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))

	if w.Code != http.StatusOK {
		t.Errorf("상태 코드 = %d, want 200", w.Code)
	}
	if got := strings.TrimSpace(w.Body.String()); got != `{"message":"ok"}` {
		t.Errorf("본문 = %s", got)
	}
}

// 모르는 에러의 내용은 응답이 아니라 로그로 간다.
func TestErrorHandler_UnknownErrorGoesToLogOnly(t *testing.T) {
	buf := captureLogs(t)

	w := httptest.NewRecorder()
	newErrorTestRouter().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/unknown", nil))

	if strings.Contains(w.Body.String(), "users") {
		t.Errorf("응답에 내부 에러 내용이 실렸다: %s", w.Body.String())
	}

	var logged bool
	for _, entry := range decodeLogs(t, buf) {
		if entry["msg"] == "request failed" && strings.Contains(fmt.Sprint(entry["err"]), "does not exist") {
			logged = true
		}
	}
	if !logged {
		t.Error("5xx 의 원인이 로그에 없다")
	}
}

// 4xx 는 에러 로그를 남기지 않는다 — 접근 로그의 상태 코드로 충분하다.
func TestErrorHandler_ClientErrorNotLogged(t *testing.T) {
	buf := captureLogs(t)

	w := httptest.NewRecorder()
	newErrorTestRouter().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/not-found", nil))

	for _, entry := range decodeLogs(t, buf) {
		if entry["msg"] == "request failed" {
			t.Errorf("4xx 인데 에러 로그가 남았다: %v", entry)
		}
	}
}
