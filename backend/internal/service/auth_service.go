package service

import (
	"errors"

	"ppk/backend/internal/config"
	"ppk/backend/internal/model"
	"ppk/backend/internal/pkg/auth"

	"gorm.io/gorm"
)

type AuthService struct {
	DB     *gorm.DB
	Config config.Config
}

type LoginResult struct {
	Token string      `json:"token"`
	User  interface{} `json:"user"`
	Role  string      `json:"role"`
}

func (s *AuthService) MerchantLogin(account, password string) (*LoginResult, error) {
	var user model.MerchantUser
	if err := s.DB.Where("account = ? AND status = ?", account, model.StatusEnabled).First(&user).Error; err != nil {
		return nil, errors.New("账号或密码错误")
	}
	if !auth.CheckPassword(user.PasswordHash, password) {
		return nil, errors.New("账号或密码错误")
	}
	token, err := auth.GenerateToken(user.ID, "merchant", s.Config.JWTSecret)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Token: token, User: user, Role: "merchant"}, nil
}

func (s *AuthService) AdminLogin(account, password string) (*LoginResult, error) {
	var user model.AdminUser
	if err := s.DB.Where("account = ? AND status = ?", account, model.StatusEnabled).First(&user).Error; err != nil {
		return nil, errors.New("账号或密码错误")
	}
	if !auth.CheckPassword(user.PasswordHash, password) {
		return nil, errors.New("账号或密码错误")
	}
	token, err := auth.GenerateToken(user.ID, "admin", s.Config.JWTSecret)
	if err != nil {
		return nil, err
	}
	return &LoginResult{Token: token, User: user, Role: "admin"}, nil
}
