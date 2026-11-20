package config

import (
	"net"
	"net/url"
	"strconv"
	"time"
)

// DB 는 데이터베이스 접속 설정이다. 키 이름은 nest-boiler 의 envs/.env.* 와 같다.
type DB struct {
	Host     string `env:"DB_HOST,required"`
	Port     int    `env:"DB_PORT,required"`
	Name     string `env:"DB_NAME,required"`
	Username string `env:"DB_USERNAME,required"`
	Password string `env:"DB_PASSWORD,required"`
	Schema   string `env:"DB_SCHEMA"   envDefault:"public"`

	// SSLMode 는 local 의 Docker Postgres 기준으로 disable 이 기본이다. 관리형 DB 는 보통 require.
	SSLMode string `env:"DB_SSL_MODE" envDefault:"disable"`

	// 커넥션 풀. Node 의 pg 는 풀 크기 하나만 정하면 됐지만, Go 의 database/sql 은 셋을 따로 정한다.
	// 정하지 않으면 MaxOpenConns 가 무제한이라, 요청이 몰릴 때 DB 의 max_connections 를 넘긴다.
	MaxOpenConns    int           `env:"DB_MAX_OPEN_CONNS"    envDefault:"10"`  // 동시에 열 수 있는 최대 커넥션
	MaxIdleConns    int           `env:"DB_MAX_IDLE_CONNS"    envDefault:"5"`   // 쓰지 않을 때 풀에 남겨 둘 커넥션
	ConnMaxLifetime time.Duration `env:"DB_CONN_MAX_LIFETIME" envDefault:"30m"` // 이 시간이 지난 커넥션은 닫고 새로 연다
}

// DSN 은 드라이버에 넘길 접속 문자열을 만든다.
//
//	postgres://root:password@localhost:5435/go-gin-boiler?search_path=public&sslmode=disable
//
// 문자열을 직접 이어 붙이지 않고 net/url 로 만든다. 비밀번호에 @ · / · 공백이 있어도 알아서 이스케이프된다.
func (d DB) DSN() string {
	query := url.Values{}
	query.Set("sslmode", d.SSLMode)
	query.Set("search_path", d.Schema)

	dsn := url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(d.Username, d.Password),
		Host:     net.JoinHostPort(d.Host, strconv.Itoa(d.Port)),
		Path:     "/" + d.Name,
		RawQuery: query.Encode(),
	}

	return dsn.String()
}
