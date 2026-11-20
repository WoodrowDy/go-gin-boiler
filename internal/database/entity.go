package database

import (
	"time"

	"gorm.io/gorm"
)

// CoreHard 는 모든 엔티티가 갖는 공통 컬럼이다(id · 생성 · 수정 일시).
// 지우면 행이 실제로 사라지는(hard delete) 도메인이 쓴다.
//
// 도메인 엔티티에 이름 없이 넣는다(embedding). 그러면 필드가 바깥 구조체의 것처럼 펼쳐진다:
//
//	type Code struct {
//		database.CoreHard
//		Name string
//	}
//	code.ID, code.CreatedAt  ← 바로 접근
//
// TS 의 extends CoreHardEntity 자리지만 상속은 아니다. Go 에는 상속이 없고, 이것은 "포함" 이다.
//
// 영속성 전용이다. json · binding 태그를 두지 않는다 — 응답 계약은 각 도메인의 dto.go 가 갖는다.
//
// CreatedAt · UpdatedAt 은 이름 규칙으로 GORM 이 알아서 채운다. 컬럼 이름은 snake_case(created_at)가 된다.
type CoreHard struct {
	ID        int64     `gorm:"primaryKey;comment:데이터 ID(PK)"`
	CreatedAt time.Time `gorm:"not null;comment:생성 일시"`
	UpdatedAt time.Time `gorm:"not null;comment:수정 일시"`
}

// CoreSoft 는 소프트 삭제(deleted_at)를 더한 공통 컬럼이다.
//
// gorm.DeletedAt 타입 필드가 있으면 GORM 의 동작이 바뀐다:
//   - Delete 는 행을 지우지 않고 deleted_at 에 시각을 적는다.
//   - 조회 · 수정에는 deleted_at IS NULL 조건이 자동으로 붙는다.
//
// TypeORM 의 @DeleteDateColumn 과 같다.
type CoreSoft struct {
	CoreHard
	DeletedAt gorm.DeletedAt `gorm:"index;comment:삭제 일시"`
}
