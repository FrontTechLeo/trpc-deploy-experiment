package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"trpc.group/trpc-go/trpc-go/log"

	authv1 "github.com/FrontTechLeo/trpc-deploy-experiment/gen/go/trpc/gametool/auth/v1"
)

type AuthService struct {
	authv1.UnimplementedAuthService
	db  *sql.DB
	rdb *redis.Client
}

func NewAuthService(db *sql.DB, rdb *redis.Client) *AuthService {
	return &AuthService{
		db:  db,
		rdb: rdb,
	}
}

func hashPassword(password string) string {
	h := sha256.New()
	h.Write([]byte("gametool_salt_" + password))
	return hex.EncodeToString(h.Sum(nil))
}

// Register 用户注册
func (s *AuthService) Register(ctx context.Context, req *authv1.RegisterRequest) (*authv1.RegisterResponse, error) {
	if req.Username == "" || req.Password == "" {
		return &authv1.RegisterResponse{
			Code:    400,
			Message: "用户名与密码不能为空",
		}, nil
	}

	// 检查用户名是否已存在
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(1) FROM users WHERE username = ?", req.Username).Scan(&count)
	if err != nil {
		log.Errorf("query user failed: %v", err)
		return &authv1.RegisterResponse{Code: 500, Message: "数据库查询失败"}, nil
	}
	if count > 0 {
		return &authv1.RegisterResponse{
			Code:    409,
			Message: "用户名已被注册",
		}, nil
	}

	pwdHash := hashPassword(req.Password)
	res, err := s.db.ExecContext(ctx, "INSERT INTO users (username, password_hash, email, status) VALUES (?, ?, ?, 1)",
		req.Username, pwdHash, req.Email)
	if err != nil {
		log.Errorf("insert user failed: %v", err)
		return &authv1.RegisterResponse{Code: 500, Message: "用户创建失败"}, nil
	}

	userID, _ := res.LastInsertId()
	log.Infof("user registered successfully: id=%d username=%s", userID, req.Username)

	return &authv1.RegisterResponse{
		Code:    0,
		Message: "注册成功",
		UserId:  uint64(userID),
	}, nil
}

// Login 用户登录并生成分布式 Token
func (s *AuthService) Login(ctx context.Context, req *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	if req.Username == "" || req.Password == "" {
		return &authv1.LoginResponse{
			Code:    400,
			Message: "用户名或密码不能为空",
		}, nil
	}

	var (
		userID  uint64
		pwdHash string
		status  int
	)
	err := s.db.QueryRowContext(ctx, "SELECT id, password_hash, status FROM users WHERE username = ?", req.Username).
		Scan(&userID, &pwdHash, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return &authv1.LoginResponse{Code: 401, Message: "用户名或密码错误"}, nil
	} else if err != nil {
		log.Errorf("query user on login failed: %v", err)
		return &authv1.LoginResponse{Code: 500, Message: "系统内部错误"}, nil
	}

	if status != 1 {
		return &authv1.LoginResponse{Code: 403, Message: "账号已被封禁"}, nil
	}

	if hashPassword(req.Password) != pwdHash {
		return &authv1.LoginResponse{Code: 401, Message: "用户名或密码错误"}, nil
	}

	// 生成分布式 Token 并存入 Redis
	token := "tk_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	sessionKey := "session:token:" + token
	sessionVal := fmt.Sprintf("%d:%s", userID, req.Username)

	if err := s.rdb.Set(ctx, sessionKey, sessionVal, 24*time.Hour).Err(); err != nil {
		log.Errorf("cache session failed: %v", err)
		return &authv1.LoginResponse{Code: 500, Message: "Token 生成失败"}, nil
	}

	log.Infof("user logged in successfully: id=%d username=%s", userID, req.Username)
	return &authv1.LoginResponse{
		Code:    0,
		Message: "登录成功",
		Token:   token,
		UserId:  userID,
	}, nil
}

// ValidateToken 验证 Token 并获取用户信息
func (s *AuthService) ValidateToken(ctx context.Context, req *authv1.ValidateTokenRequest) (*authv1.ValidateTokenResponse, error) {
	if req.Token == "" {
		return &authv1.ValidateTokenResponse{Valid: false}, nil
	}

	sessionKey := "session:token:" + req.Token
	val, err := s.rdb.Get(ctx, sessionKey).Result()
	if errors.Is(err, redis.Nil) {
		return &authv1.ValidateTokenResponse{Valid: false}, nil
	} else if err != nil {
		log.Errorf("get token from redis failed: %v", err)
		return &authv1.ValidateTokenResponse{Valid: false}, nil
	}

	parts := strings.SplitN(val, ":", 2)
	if len(parts) != 2 {
		return &authv1.ValidateTokenResponse{Valid: false}, nil
	}

	userID, _ := strconv.ParseUint(parts[0], 10, 64)
	return &authv1.ValidateTokenResponse{
		Valid:    true,
		UserId:   userID,
		Username: parts[1],
	}, nil
}
