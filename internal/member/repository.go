package member

import (
	"context"

	"github.com/changhyeonkim/pray-together/go-api-server/internal/model"
	"gorm.io/gorm"
)

type MemberRepository struct{}

func NewMemberRepository() *MemberRepository {
	return &MemberRepository{}
}

func (m *MemberRepository) IsExist(ctx context.Context, db *gorm.DB, email string) (bool, error) {
	var count int64
	err := db.WithContext(ctx).
		Model(&model.Member{}).
		Where("email = ?", email).
		Count(&count).Error

	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (m *MemberRepository) Create(ctx context.Context, db *gorm.DB, member *model.Member) error {
	return db.WithContext(ctx).Create(member).Error
}

func (m *MemberRepository) FindByEmail(ctx context.Context, db *gorm.DB, email string) (*model.Member, error) {
	var member model.Member
	err := db.WithContext(ctx).Where("email = ?", email).First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

func (m *MemberRepository) FindByID(ctx context.Context, db *gorm.DB, ID uint32) (*model.Member, error) {
	var member model.Member
	err := db.WithContext(ctx).Where("id = ?", ID).First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}
