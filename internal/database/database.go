// Package database 는 DB 연결과 모든 엔티티가 공유하는 기반을 둔다.
package database

import (
	"context"
	"fmt"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/WoodrowDy/go-gin-boiler/internal/global/config"
)

// pingTimeout 은 기동할 때 DB 응답을 기다리는 시간이다.
const pingTimeout = 5 * time.Second

// Open 은 DB 에 연결하고, 실제로 닿는지 확인한 뒤 *gorm.DB 를 돌려준다.
//
// *gorm.DB 는 커넥션 하나가 아니라 풀이다. 앱 전체가 하나를 만들어 나눠 쓰고, 고루틴 여럿이 동시에 써도 된다.
// nest-boiler 의 DataSource 에 해당한다.
func Open(ctx context.Context, cfg config.DB) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		// GORM 은 기본으로 느린 쿼리와 에러를 자기 포맷으로 표준 출력에 찍는다. 끈다.
		// 우리 로그는 slog 하나로 나가고, 에러는 에러 미들웨어가 traceId 와 함께 남긴다.
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// GORM 아래에는 표준 라이브러리의 *sql.DB 가 있다. 풀 설정과 Ping · Close 는 그쪽 일이다.
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// 연결은 처음 쓸 때 맺어진다(lazy). 여기서 한 번 찔러 보지 않으면,
	// 접속 정보가 틀려도 서버는 뜨고 첫 요청에서야 실패한다.
	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()

	if err := sqlDB.PingContext(pingCtx); err != nil {
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return db, nil
}

// Close 는 풀의 커넥션을 모두 닫는다. 서버가 요청을 다 처리한 뒤(Shutdown 다음)에 부른다.
func Close(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}

	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("close database: %w", err)
	}

	return nil
}
