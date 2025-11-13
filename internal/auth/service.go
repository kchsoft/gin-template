package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/member"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/model"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/shared/database"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/shared/logger"
	"github.com/changhyeonkim/pray-together/go-api-server/internal/shared/token"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
	"strconv"
)

type AuthService struct {
	db               *gorm.DB
	memberRepository *member.MemberRepository
	tokenManager     token.Manager
}

func NewAuthService(db *gorm.DB, memberRepository *member.MemberRepository, tokenManager token.Manager) *AuthService {
	return &AuthService{
		db:               db,
		memberRepository: memberRepository,
		tokenManager:     tokenManager,
	}
}

func (a *AuthService) Login(ctx context.Context, request *LoginRequest) (*LoginResponse, error) {
	log := logger.FromContext(ctx)

	// 1. Find member by email
	member, err := a.memberRepository.FindByEmail(ctx, a.db, request.Email)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("이메일을 찾을 수 없습니다: email=%s %w", logger.MaskEmail(request.Email), ErrInCorrectEmailPassword) // Security: don't reveal if email exists
		}
		return nil, fmt.Errorf("로그인 실패: email=%s %w", logger.MaskEmail(request.Email), err)
	}

	// 2. Validate password
	if err := bcrypt.CompareHashAndPassword([]byte(member.Password), []byte(request.Password)); err != nil {
		return nil, fmt.Errorf("로그인 실패: email=%s %w", logger.MaskEmail(request.Email), ErrInCorrectEmailPassword)
	}

	// 3. Generate JWT tokens
	memberID := strconv.FormatUint(uint64(member.ID), 10)
	accessToken, err := a.tokenManager.GenerateAccessToken(memberID, member.Email)
	if err != nil {
		return nil, fmt.Errorf("AccessToken 생성 실패: memberID=%d %w", memberID, err)
	}

	refreshToken, err := a.tokenManager.GenerateRefreshToken(memberID, member.Email)
	if err != nil {
		return nil, fmt.Errorf("RefreshToken 생성 실패: memberID=%d %w", memberID, err)
	}

	log.Info("로그인 성공", "email", logger.MaskEmail(request.Email))

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

func (a *AuthService) Signup(ctx context.Context, request *SignupRequest) error {
	log := logger.FromContext(ctx)
	return database.WithTransaction(ctx, a.db, func(tx *gorm.DB) error {
		exists, err := a.memberRepository.IsExist(ctx, tx, request.Email)
		if err != nil {
			return fmt.Errorf("회원 존재 확인 오류: email=%s %w", logger.MaskEmail(request.Email), err)
		}
		if exists {
			return fmt.Errorf("이미 존재하는 회원입니다: email=%s %w", logger.MaskEmail(request.Email), member.ErrMemberAlreadyExists)
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("비밀번호 해싱 실패: %w", err)
		}

		member := model.NewMember(request.Name, request.Email, request.PhoneNumber, string(hashedPassword))
		if err := a.memberRepository.Create(ctx, tx, member); err != nil {
			return fmt.Errorf("회원 계정 생성 실패: %w", err)
		}

		log.Info("Member created successfully", "email", logger.MaskEmail(request.Email))
		return nil
	})
}
