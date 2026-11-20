package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

// captureLogs 는 테스트 동안 기본 로거를 버퍼로 돌리고, 끝나면 되돌린다.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()

	var buf bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&buf, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	return &buf
}

// decodeLogs 는 한 줄에 하나씩 찍힌 JSON 로그를 풀어 돌려준다.
func decodeLogs(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()

	var logs []map[string]any
	decoder := json.NewDecoder(buf)
	for decoder.More() {
		var entry map[string]any
		if err := decoder.Decode(&entry); err != nil {
			t.Fatalf("로그 한 줄을 풀지 못했다: %v", err)
		}
		logs = append(logs, entry)
	}

	return logs
}

func newTestRouter() *gin.Engine {
	return newTestRouterWith(AccessLogOptions{LogRequest: true, SkipPaths: []string{"/skipped"}})
}

func newTestRouterWith(options AccessLogOptions) *gin.Engine {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.Use(TraceID(), AccessLog(options), Recovery())
	r.GET("/ok", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/missing", func(c *gin.Context) { c.Status(http.StatusNotFound) })
	r.GET("/skipped", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.GET("/panic", func(_ *gin.Context) { panic("boom") })

	return r
}

func TestAccessLog(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantLevel  string
	}{
		{name: "2xx 는 INFO", path: "/ok", wantStatus: http.StatusOK, wantLevel: "INFO"},
		{name: "4xx 는 WARN", path: "/missing", wantStatus: http.StatusNotFound, wantLevel: "WARN"},
		{name: "panic 은 500 · ERROR", path: "/panic", wantStatus: http.StatusInternalServerError, wantLevel: "ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := captureLogs(t)

			w := httptest.NewRecorder()
			newTestRouter().ServeHTTP(w, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if w.Code != tt.wantStatus {
				t.Fatalf("상태 코드 = %d, want %d", w.Code, tt.wantStatus)
			}

			logs := decodeLogs(t, buf)
			if len(logs) == 0 {
				t.Fatal("로그가 없다")
			}

			first, last := logs[0], logs[len(logs)-1]
			if first["msg"] != "HTTP REQ" {
				t.Errorf("첫 줄 msg = %v, want HTTP REQ", first["msg"])
			}
			if last["msg"] != "HTTP RES" {
				t.Errorf("마지막 줄 msg = %v, want HTTP RES", last["msg"])
			}
			if last["level"] != tt.wantLevel {
				t.Errorf("HTTP RES level = %v, want %v", last["level"], tt.wantLevel)
			}
			if got := last["status"]; got != float64(tt.wantStatus) {
				t.Errorf("HTTP RES status = %v, want %d", got, tt.wantStatus)
			}
		})
	}
}

func TestAccessLog_ResponseOnly(t *testing.T) {
	buf := captureLogs(t)

	// 옵션을 비워 두면 HTTP RES 한 줄만 남는다.
	w := httptest.NewRecorder()
	newTestRouterWith(AccessLogOptions{}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ok", nil))

	logs := decodeLogs(t, buf)
	if len(logs) != 1 {
		t.Fatalf("로그 %d줄, want 1", len(logs))
	}
	if logs[0]["msg"] != "HTTP RES" {
		t.Errorf("msg = %v, want HTTP RES", logs[0]["msg"])
	}
	for _, key := range []string{"method", "path", "status", "durationMs", "clientIp"} {
		if _, ok := logs[0][key]; !ok {
			t.Errorf("HTTP RES 에 %s 가 없다", key)
		}
	}
}

func TestAccessLog_SkipPath(t *testing.T) {
	buf := captureLogs(t)

	w := httptest.NewRecorder()
	newTestRouter().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/skipped", nil))

	if logs := decodeLogs(t, buf); len(logs) != 0 {
		t.Errorf("로그 %d줄, want 0", len(logs))
	}
}

func TestAccessLog_MasksCredentials(t *testing.T) {
	tests := []struct {
		name   string
		header string
		value  string
		want   string
	}{
		{name: "스킴은 남긴다", header: "Authorization", value: "Bearer secret-token", want: "Bearer ***"},
		{name: "스킴이 없으면 통째로", header: "Authorization", value: "secret-token", want: "***"},
		{name: "다른 헤더는 그대로", header: "User-Agent", value: "curl/8.0", want: "curl/8.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := captureLogs(t)

			req := httptest.NewRequest(http.MethodGet, "/ok", nil)
			req.Header.Set(tt.header, tt.value)
			newTestRouter().ServeHTTP(httptest.NewRecorder(), req)

			logs := decodeLogs(t, buf)
			headers, _ := logs[0]["headers"].(map[string]any)
			if got := headers[tt.header]; got != tt.want {
				t.Errorf("headers[%s] = %v, want %q", tt.header, got, tt.want)
			}
			if bytes.Contains(buf.Bytes(), []byte("secret-token")) {
				t.Error("로그에 자격증명이 그대로 남았다")
			}
		})
	}
}

func TestRecovery_LogsPanic(t *testing.T) {
	buf := captureLogs(t)

	w := httptest.NewRecorder()
	newTestRouter().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/panic", nil))

	var found map[string]any
	for _, entry := range decodeLogs(t, buf) {
		if entry["msg"] == "panic recovered" {
			found = entry
		}
	}

	if found == nil {
		t.Fatal("panic 로그가 없다")
	}
	if found["panic"] != "boom" {
		t.Errorf("panic = %v, want boom", found["panic"])
	}
	if stack, _ := found["stack"].(string); stack == "" {
		t.Error("스택이 비어 있다")
	}
	if bytes.Contains(w.Body.Bytes(), []byte("boom")) {
		t.Errorf("응답 본문에 panic 내용이 실렸다: %s", w.Body.String())
	}
}
