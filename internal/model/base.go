package model

import (
	"time"
)

// GORM이 CreatedAt, UpdatedAt을 자동으로 관리
// CreatedBy, UpdatedBy는 Repository에서 명시적으로 설정
type BaseEntity struct {
	CreatedAt time.Time `gorm:"column:created_at;not null"` // GORM이 자동 관리
	UpdatedAt time.Time `gorm:"column:updated_at;not null"` // GORM이 자동 관리
	CreatedBy *int64    `gorm:"column:created_by"`
	UpdatedBy *int64    `gorm:"column:updated_by"`
}
