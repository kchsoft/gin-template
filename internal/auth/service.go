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
			log.Warn("로그인 실패 - member email not found", "email", logger.MaskEmail(request.Email))
			return nil, fmt.Errorf("error %w", ErrInCorrectEmailPassword) // Security: don't reveal if email exists
		}
		log.Error("로그인 실패 - 알 수 없는 오류", "error", err)
		return nil, fmt.Errorf("로그인 실패: %w", err)
	}

	// 2. Validate password
	if err := bcrypt.CompareHashAndPassword([]byte(member.Password), []byte(request.Password)); err != nil {
		log.Warn("로그인 실패 - invalid password", "email", logger.MaskEmail(request.Email))
		return nil, fmt.Errorf("error %w", ErrInCorrectEmailPassword)
	}

	// 3. Generate JWT tokens
	memberID := strconv.FormatUint(uint64(member.ID), 10)
	accessToken, err := a.tokenManager.GenerateAccessToken(memberID, member.Email)
	if err != nil {
		log.Error("access token 생성 실패", "error", err)
		return nil, fmt.Errorf("generate access token: %w", err)
	}

	refreshToken, err := a.tokenManager.GenerateRefreshToken(memberID, member.Email)
	if err != nil {
		log.Error("refresh token 생성 실패", "error", err)
		return nil, fmt.Errorf("generate refresh token: %w", err)
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
			log.Error("Failed to check member existence", "error", err)
			return fmt.Errorf("check member existence: %w", err)
		}
		if exists {
			log.Warn("Member already exists", "email", logger.MaskEmail(request.Email))
			return fmt.Errorf("error %w", member.ErrMemberAlreadyExists)
		}

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(request.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Error("Failed to hash password", "error", err)
			return fmt.Errorf("hash password: %w", err)
		}

		member := model.NewMember(request.Name, request.Email, request.PhoneNumber, string(hashedPassword))
		if err := a.memberRepository.Create(ctx, tx, member); err != nil {
			log.Error("Failed to create member", "error", err)
			return fmt.Errorf("create member: %w", err)
		}

		log.Info("Member created successfully", "email", logger.MaskEmail(request.Email))
		return nil
	})
}
