package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/FrontTechLeo/trpc-deploy-experiment/pkg/database"
	authv1 "github.com/FrontTechLeo/trpc-deploy-experiment/gen/go/trpc/gametool/auth/v1"
)

func TestAuthService_Lifecycle(t *testing.T) {
	db, err := database.InitMySQL("root:root@tcp(127.0.0.1:3306)/gametool?parseTime=true&charset=utf8mb4")
	if err != nil {
		t.Fatalf("InitMySQL failed: %v", err)
	}
	defer db.Close()

	rdb, err := database.InitRedis("127.0.0.1:6379", "", 0)
	if err != nil {
		t.Fatalf("InitRedis failed: %v", err)
	}
	defer rdb.Close()

	authSvr := NewAuthService(db, rdb)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testUser := fmt.Sprintf("test_user_%d", time.Now().UnixNano())
	testPwd := "SecretPassword123!"

	// 1. 测试注册
	regRsp, err := authSvr.Register(ctx, &authv1.RegisterRequest{
		Username: testUser,
		Password: testPwd,
		Email:    testUser + "@example.com",
	})
	if err != nil {
		t.Fatalf("Register error: %v", err)
	}
	if regRsp.Code != 0 || regRsp.UserId == 0 {
		t.Fatalf("Register unexpected response: code=%d msg=%s", regRsp.Code, regRsp.Message)
	}
	t.Logf("User registered: id=%d", regRsp.UserId)

	// 2. 测试重复注册冲突
	dupRsp, err := authSvr.Register(ctx, &authv1.RegisterRequest{
		Username: testUser,
		Password: testPwd,
	})
	if err != nil || dupRsp.Code != 409 {
		t.Fatalf("Duplicate register expected 409, got code=%d err=%v", dupRsp.Code, err)
	}

	// 3. 测试密码错误登录
	wrongPwdRsp, err := authSvr.Login(ctx, &authv1.LoginRequest{
		Username: testUser,
		Password: "wrong_password",
	})
	if err != nil || wrongPwdRsp.Code != 401 {
		t.Fatalf("Wrong password login expected 401, got code=%d err=%v", wrongPwdRsp.Code, err)
	}

	// 4. 测试正确登录
	loginRsp, err := authSvr.Login(ctx, &authv1.LoginRequest{
		Username: testUser,
		Password: testPwd,
	})
	if err != nil {
		t.Fatalf("Login error: %v", err)
	}
	if loginRsp.Code != 0 || loginRsp.Token == "" {
		t.Fatalf("Login unexpected response: code=%d msg=%s token=%s", loginRsp.Code, loginRsp.Message, loginRsp.Token)
	}
	t.Logf("User logged in: token=%s user_id=%d", loginRsp.Token, loginRsp.UserId)

	// 5. 测试 Token 校验
	valRsp, err := authSvr.ValidateToken(ctx, &authv1.ValidateTokenRequest{
		Token: loginRsp.Token,
	})
	if err != nil {
		t.Fatalf("ValidateToken error: %v", err)
	}
	if !valRsp.Valid || valRsp.UserId != regRsp.UserId || valRsp.Username != testUser {
		t.Fatalf("ValidateToken mismatch: valid=%v user_id=%d username=%s", valRsp.Valid, valRsp.UserId, valRsp.Username)
	}
	t.Logf("Token validated successfully: user=%s id=%d", valRsp.Username, valRsp.UserId)
}
