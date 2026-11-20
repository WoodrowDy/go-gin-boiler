package config

import (
	"maps"
	"testing"
	"time"
)

// TestLoad 는 Load() 가 환경변수에 따라 성공/실패하는지, 기본값이 채워지는지 검증한다.
// "테이블 주도 테스트" — 케이스들을 슬라이스로 나열하고 for 문으로 같은 검증을 반복한다.
// jest 의 describe/it 대신 Go 는 이 형태가 표준이다.
func TestLoad(t *testing.T) {
	// 1) 케이스 표. 각 행이 하나의 테스트다.
	//    envs   : 이 케이스에서 세팅할 환경변수
	//    wantErr: Load() 가 에러를 내야 하는가(true) / 성공해야 하는가(false)
	//    필수값이 늘어서(Step 5: DB_*) 케이스마다 전부 적으면 표가 읽기 어려워진다.
	//    "정상" 한 벌을 만들어 두고, 각 케이스는 거기서 뺄 키(omit)와 덮어쓸 값(override)만 적는다.
	valid := map[string]string{
		"APP_ENV":     "local",
		"PORT":        "8080",
		"DB_HOST":     "localhost",
		"DB_PORT":     "5435",
		"DB_NAME":     "go-gin-boiler",
		"DB_USERNAME": "root",
		"DB_PASSWORD": "password",
	}

	tests := []struct {
		name     string
		omit     []string
		override map[string]string
		wantErr  bool
	}{
		{name: "정상", wantErr: false},
		{name: "PORT 없음", omit: []string{"PORT"}, wantErr: true},
		{name: "DB_HOST 없음", omit: []string{"DB_HOST"}, wantErr: true},
		{name: "DB_PASSWORD 없음", omit: []string{"DB_PASSWORD"}, wantErr: true},
		{name: "잘못된 APP_ENV", override: map[string]string{"APP_ENV": "prd"}, wantErr: true},
		{name: "숫자가 아닌 DB_PORT", override: map[string]string{"DB_PORT": "abc"}, wantErr: true},
	}

	// 2) 각 케이스를 t.Run 으로 감싸 실행한다. 실패 시 어느 케이스인지 이름으로 나온다.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 3) 이 케이스의 환경변수를 세팅한다.
			//    t.Setenv 는 이 서브테스트가 끝나면 원래 값으로 자동 복원해준다.
			//    (그래서 케이스끼리 서로 오염되지 않는다. 대신 t.Parallel() 과는 못 쓴다.)
			//    maps.Clone 으로 복사한 뒤 고친다. valid 를 직접 고치면 다음 케이스가 오염된다
			//    (map 은 참조라서, 대입만 하면 같은 map 을 가리킨다).
			envs := maps.Clone(valid)
			for _, k := range tt.omit {
				delete(envs, k)
			}
			maps.Copy(envs, tt.override)

			for k, v := range envs {
				t.Setenv(k, v)
			}

			// 4) 실제 호출. 같은 package 라 config. 접두어 없이 그냥 Load().
			cfg, err := Load()

			// 5) 에러가 났는지(err != nil) 가 기대(wantErr)와 일치하는가?
			//    (err != nil) 은 "에러가 났다"는 bool. 이걸 wantErr 와 직접 비교하는 게 Go 관용구다.
			if (err != nil) != tt.wantErr {
				t.Fatalf("Load() error = %v, wantErr = %v", err, tt.wantErr)
			}

			// 6) ★ 에러 케이스는 여기서 끝낸다.
			//    Load() 는 에러일 때 cfg 로 nil 을 돌려준다.
			//    return 없이 아래로 내려가면 nil.Server 를 건드려 패닉난다.
			//    "함수가 (값, error) 를 주면, 에러일 때 값은 믿지 않는다"는 Go 관례.
			if tt.wantErr {
				return
			}

			// 7) 여기부터는 정상 케이스만 도달한다 → cfg 안전하게 사용 가능.
			//    PORT 만 줬으니 ReadTimeout 은 envDefault("5s") 로 채워져 있어야 한다.
			if cfg.Server.ReadTimeout != 5*time.Second {
				t.Fatalf("ReadTimeout = %v, want 5s", cfg.Server.ReadTimeout)
			}

			// 8) DB 도 같다. 필수값은 준 대로, 주지 않은 것은 envDefault 로.
			if cfg.DB.Port != 5435 {
				t.Fatalf("DB.Port = %d, want 5435", cfg.DB.Port)
			}
			if cfg.DB.Schema != "public" || cfg.DB.MaxOpenConns != 10 {
				t.Fatalf("DB 기본값이 다르다: Schema = %q, MaxOpenConns = %d", cfg.DB.Schema, cfg.DB.MaxOpenConns)
			}
		})
	}
}
