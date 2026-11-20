package config

import (
	"net/url"
	"testing"
)

func TestDB_DSN(t *testing.T) {
	tests := []struct {
		name         string
		db           DB
		want         string // 비어 있으면 문자열 전체 비교를 건너뛴다
		wantPassword string
	}{
		{
			name: "로컬 기본값",
			db: DB{
				Host: "localhost", Port: 5435, Name: "go-gin-boiler",
				Username: "root", Password: "password", Schema: "public", SSLMode: "disable",
			},
			want:         "postgres://root:password@localhost:5435/go-gin-boiler?search_path=public&sslmode=disable",
			wantPassword: "password",
		},
		{
			name: "비밀번호의 특수 문자는 이스케이프되고, 풀면 원래 값이다",
			db: DB{
				Host: "db.internal", Port: 5432, Name: "app",
				Username: "app", Password: "p@ss/w ord?&", Schema: "public", SSLMode: "require",
			},
			wantPassword: "p@ss/w ord?&",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.db.DSN()

			if tt.want != "" && got != tt.want {
				t.Errorf("DSN() = %q\nwant    %q", got, tt.want)
			}

			parsed, err := url.Parse(got)
			if err != nil {
				t.Fatalf("DSN 이 URL 로 풀리지 않는다: %v", err)
			}
			if password, _ := parsed.User.Password(); password != tt.wantPassword {
				t.Errorf("풀어낸 비밀번호 = %q, want %q", password, tt.wantPassword)
			}
			if parsed.Hostname() != tt.db.Host {
				t.Errorf("host = %q, want %q", parsed.Hostname(), tt.db.Host)
			}
			if parsed.Path != "/"+tt.db.Name {
				t.Errorf("path = %q, want /%s", parsed.Path, tt.db.Name)
			}
		})
	}
}
