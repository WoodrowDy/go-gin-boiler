package config

import (
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
	tests := []struct {
		name    string
		envs    map[string]string
		wantErr bool
	}{
		{name: "PORT 없음", envs: map[string]string{"APP_ENV": "local"}, wantErr: true},
		{name: "잘못된 APP_ENV", envs: map[string]string{"APP_ENV": "prd", "PORT": "8080"}, wantErr: true},
		{name: "정상", envs: map[string]string{"APP_ENV": "local", "PORT": "8080"}, wantErr: false},
	}

	// 2) 각 케이스를 t.Run 으로 감싸 실행한다. 실패 시 어느 케이스인지 이름으로 나온다.
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 3) 이 케이스의 환경변수를 세팅한다.
			//    t.Setenv 는 이 서브테스트가 끝나면 원래 값으로 자동 복원해준다.
			//    (그래서 케이스끼리 서로 오염되지 않는다. 대신 t.Parallel() 과는 못 쓴다.)
			for k, v := range tt.envs {
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
		})
	}
}
