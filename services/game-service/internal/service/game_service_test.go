package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	gamev1 "github.com/FrontTechLeo/trpc-deploy-experiment/gen/go/trpc/gametool/game/v1"
	"github.com/FrontTechLeo/trpc-deploy-experiment/pkg/clickhouse"
	"github.com/FrontTechLeo/trpc-deploy-experiment/pkg/database"
	"github.com/FrontTechLeo/trpc-deploy-experiment/pkg/mq"
)

func TestGameService_LifecycleAndSaga(t *testing.T) {
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

	mqClient, err := mq.InitRabbitMQ("amqp://guest:guest@127.0.0.1:5672/")
	if err != nil {
		t.Fatalf("InitRabbitMQ failed: %v", err)
	}
	defer mqClient.Close()

	chClient, _ := clickhouse.InitClickHouse("http://127.0.0.1:8123")

	gameSvr := NewGameService(db, rdb, mqClient, chClient)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	testUID := uint64(time.Now().UnixNano()%1000000 + 10000)
	charName := fmt.Sprintf("Hero_%d", testUID)

	// 1. 创建角色
	createRsp, err := gameSvr.CreateCharacter(ctx, &gamev1.CreateCharacterRequest{
		UserId:        testUID,
		CharacterName: charName,
		InitialCoins:  2000,
	})
	if err != nil {
		t.Fatalf("CreateCharacter error: %v", err)
	}
	if createRsp.Code != 0 || createRsp.Character == nil || createRsp.Character.Coins != 2000 {
		t.Fatalf("CreateCharacter unexpected response: code=%d msg=%s", createRsp.Code, createRsp.Message)
	}
	t.Logf("Character created: ID=%d Name=%s Coins=%d", createRsp.Character.Id, createRsp.Character.CharacterName, createRsp.Character.Coins)

	// 2. 获取角色
	getRsp, err := gameSvr.GetCharacter(ctx, &gamev1.GetCharacterRequest{
		UserId: testUID,
	})
	if err != nil || getRsp.Code != 0 || getRsp.Character.CharacterName != charName {
		t.Fatalf("GetCharacter failed: code=%d err=%v", getRsp.Code, err)
	}
	t.Logf("Character retrieved: level=%d coins=%d", getRsp.Character.Level, getRsp.Character.Coins)

	// 3. 升级角色
	upRsp, err := gameSvr.UpdateCharacterLevel(ctx, &gamev1.UpdateCharacterLevelRequest{
		UserId:   testUID,
		NewLevel: 15,
	})
	if err != nil || upRsp.Code != 0 {
		t.Fatalf("UpdateCharacterLevel failed: code=%d msg=%s err=%v", upRsp.Code, upRsp.Message, err)
	}
	t.Logf("Character leveled up: %s", upRsp.Message)

	// 4. 测试 Saga 补偿流程：创建订单 -> 模拟支付/发货故障 -> 触发 Saga 补偿回滚
	orderRsp1, err := gameSvr.CreateOrder(ctx, &gamev1.CreateOrderRequest{
		UserId:   testUID,
		ItemName: "DragonSword",
		Amount:   50.00,
		Channel:  "game_store",
	})
	if err != nil || orderRsp1.Code != 0 || orderRsp1.OrderNo == "" {
		t.Fatalf("CreateOrder 1 failed: code=%d err=%v", orderRsp1.Code, err)
	}
	t.Logf("Order 1 created for Saga compensation test: %s", orderRsp1.OrderNo)

	payRspCompensate, err := gameSvr.ProcessPayment(ctx, &gamev1.ProcessPaymentRequest{
		OrderNo:         orderRsp1.OrderNo,
		UserId:          testUID,
		Amount:          50.00,
		SimulateFailure: true,
	})
	if err != nil {
		t.Fatalf("ProcessPayment error: %v", err)
	}
	if payRspCompensate.Status != "COMPENSATED" {
		t.Fatalf("Expected COMPENSATED status, got: %s msg=%s", payRspCompensate.Status, payRspCompensate.Message)
	}
	t.Logf("Saga Compensation verified: %s, status=%s", payRspCompensate.Message, payRspCompensate.Status)

	// 5. 测试正常 Saga 流程：创建订单 -> 支付成功 -> 状态更新与金币发放
	orderRsp2, err := gameSvr.CreateOrder(ctx, &gamev1.CreateOrderRequest{
		UserId:   testUID,
		ItemName: "DiamondPack",
		Amount:   100.00,
		Channel:  "alipay",
	})
	if err != nil || orderRsp2.Code != 0 || orderRsp2.OrderNo == "" {
		t.Fatalf("CreateOrder 2 failed: code=%d err=%v", orderRsp2.Code, err)
	}

	payRspSuccess, err := gameSvr.ProcessPayment(ctx, &gamev1.ProcessPaymentRequest{
		OrderNo:         orderRsp2.OrderNo,
		UserId:          testUID,
		Amount:          100.00,
		SimulateFailure: false,
	})
	if err != nil || payRspSuccess.Code != 0 || payRspSuccess.Status != "PAID" {
		t.Fatalf("ProcessPayment success failed: status=%s msg=%s err=%v", payRspSuccess.Status, payRspSuccess.Message, err)
	}
	t.Logf("Saga Payment verified: %s, status=%s", payRspSuccess.Message, payRspSuccess.Status)

	// 验证金币增加 (2000 + 100*10 = 3000)
	getCharAfterPay, _ := gameSvr.GetCharacter(ctx, &gamev1.GetCharacterRequest{UserId: testUID})
	if getCharAfterPay.Character.Coins != 3000 {
		t.Fatalf("Coins mismatch after payment: expected 3000, got %d", getCharAfterPay.Character.Coins)
	}
	t.Logf("Character balance after payment verified: %d coins", getCharAfterPay.Character.Coins)

	// 6. 测试业务埋点遥测上报
	evtRsp, err := gameSvr.ReportGameEvent(ctx, &gamev1.ReportGameEventRequest{
		UserId:      testUID,
		EventType:   "dungeon_pass",
		PayloadJson: `{"dungeon_id": 101, "score": 9500, "stars": 3}`,
	})
	if err != nil || evtRsp.Code != 0 {
		t.Fatalf("ReportGameEvent failed: code=%d msg=%s err=%v", evtRsp.Code, evtRsp.Message, err)
	}
	t.Logf("Game event telemetry reported successfully: %s", evtRsp.Message)
}
