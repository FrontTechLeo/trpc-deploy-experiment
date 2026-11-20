package main

import (
	"os"

	authv1 "github.com/FrontTechLeo/trpc-deploy-experiment/gen/go/trpc/gametool/auth/v1"
	"github.com/FrontTechLeo/trpc-deploy-experiment/pkg/database"
	"github.com/FrontTechLeo/trpc-deploy-experiment/services/auth-service/internal/service"
	trpc "trpc.group/trpc-go/trpc-go"
	"trpc.group/trpc-go/trpc-go/log"

	_ "trpc.group/trpc-go/trpc-naming-polarismesh"
)

func main() {
	s := trpc.NewServer()

	// 初始化数据库与缓存
	dsn := os.Getenv("MYSQL_DSN")
	if dsn == "" {
		dsn = "root:root@tcp(127.0.0.1:3306)/gametool?parseTime=true&charset=utf8mb4"
	}
	db, err := database.InitMySQL(dsn)
	if err != nil {
		log.Fatalf("init mysql failed: %v", err)
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "127.0.0.1:6379"
	}
	rdb, err := database.InitRedis(redisAddr, "", 0)
	if err != nil {
		log.Fatalf("init redis failed: %v", err)
	}

	// 注册 AuthService 业务实现
	authSvr := service.NewAuthService(db, rdb)
	authv1.RegisterAuthServiceService(s.Service("trpc.gametool.auth.v1.AuthService"), authSvr)

	log.Info("AuthService starting, ready to serve and register with PolarisMesh...")
	if err := s.Serve(); err != nil {
		log.Fatalf("server serve failed: %v", err)
	}
}
