package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"trpc.group/trpc-go/trpc-go/log"

	gamev1 "github.com/FrontTechLeo/trpc-deploy-experiment/gen/go/trpc/gametool/game/v1"
	"github.com/FrontTechLeo/trpc-deploy-experiment/pkg/mq"
)

type GameService struct {
	gamev1.UnimplementedGameService
	db  *sql.DB
	rdb *redis.Client
	mq  *mq.RabbitMQ
}

type CharacterAttributes struct {
	Coins int64 `json:"coins"`
	HP    int32 `json:"hp"`
	MP    int32 `json:"mp"`
}

func NewGameService(db *sql.DB, rdb *redis.Client, mqClient *mq.RabbitMQ) *GameService {
	svc := &GameService{
		db:  db,
		rdb: rdb,
		mq:  mqClient,
	}
	if mqClient != nil {
		svc.startConsumers()
	}
	return svc
}

func (s *GameService) startConsumers() {
	// 消费 Saga 订单事件
	err := s.mq.Consume(
		mq.QueueSagaPayment,
		mq.ExchangeSagaOrder,
		"order.*",
		func(msg []byte) error {
			var evt mq.OrderEvent
			if err := json.Unmarshal(msg, &evt); err != nil {
				return err
			}
			log.Infof("[Saga Consumer] 收到订单流转事件: OrderNo=%s, UserID=%d, Status=%s, Amount=%.2f, Reason=%s",
				evt.OrderNo, evt.UserID, evt.Status, evt.Amount, evt.Reason)
			return nil
		},
	)
	if err != nil {
		log.Warnf("[Saga Consumer] 初始化 Saga 消费队列失败: %v", err)
	}

	// 消费游戏业务遥测事件
	err = s.mq.Consume(
		mq.QueueTelemetry,
		mq.ExchangeGameEvents,
		mq.RoutingKeyGameTelemetry,
		func(msg []byte) error {
			var evt mq.TelemetryEvent
			if err := json.Unmarshal(msg, &evt); err != nil {
				return err
			}
			log.Infof("[Telemetry Consumer] 收到业务遥测埋点: UserID=%d, EventType=%s, Payload=%s",
				evt.UserID, evt.EventType, evt.Payload)
			return nil
		},
	)
	if err != nil {
		log.Warnf("[Telemetry Consumer] 初始化遥测消费队列失败: %v", err)
	}
}

// CreateCharacter 创建游戏角色
func (s *GameService) CreateCharacter(ctx context.Context, req *gamev1.CreateCharacterRequest) (*gamev1.CreateCharacterResponse, error) {
	if req.UserId == 0 || req.CharacterName == "" {
		return &gamev1.CreateCharacterResponse{
			Code:    400,
			Message: "用户ID与角色名不能为空",
		}, nil
	}

	// 检查角色是否存在
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM game_characters WHERE user_id = ?", req.UserId).Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("query character exists failed: %w", err)
	}
	if count > 0 {
		return &gamev1.CreateCharacterResponse{
			Code:    409,
			Message: "该用户已有角色，不能重复创建",
		}, nil
	}

	coins := req.InitialCoins
	if coins <= 0 {
		coins = 1000 // 默认初始金币 1000
	}

	attrs := CharacterAttributes{
		Coins: coins,
		HP:    100,
		MP:    100,
	}
	attrsJSON, _ := json.Marshal(attrs)

	res, err := s.db.ExecContext(ctx,
		"INSERT INTO game_characters (user_id, character_name, level, attributes, created_at, updated_at) VALUES (?, ?, 1, ?, NOW(), NOW())",
		req.UserId, req.CharacterName, string(attrsJSON),
	)
	if err != nil {
		return nil, fmt.Errorf("insert character failed: %w", err)
	}

	charID, _ := res.LastInsertId()

	charInfo := &gamev1.CharacterInfo{
		Id:             uint64(charID),
		UserId:         req.UserId,
		CharacterName:  req.CharacterName,
		Level:          1,
		Coins:          coins,
		AttributesJson: string(attrsJSON),
	}

	// 缓存到 Redis
	if s.rdb != nil {
		cacheKey := fmt.Sprintf("game:char:%d", req.UserId)
		data, _ := json.Marshal(charInfo)
		_ = s.rdb.Set(ctx, cacheKey, string(data), time.Hour*24).Err()
	}

	return &gamev1.CreateCharacterResponse{
		Code:      0,
		Message:   "角色创建成功",
		Character: charInfo,
	}, nil
}

// GetCharacter 获取角色详情
func (s *GameService) GetCharacter(ctx context.Context, req *gamev1.GetCharacterRequest) (*gamev1.GetCharacterResponse, error) {
	if req.UserId == 0 {
		return &gamev1.GetCharacterResponse{
			Code:    400,
			Message: "用户ID不能为空",
		}, nil
	}

	// 先查缓存
	cacheKey := fmt.Sprintf("game:char:%d", req.UserId)
	if s.rdb != nil {
		val, err := s.rdb.Get(ctx, cacheKey).Result()
		if err == nil && val != "" {
			var charInfo gamev1.CharacterInfo
			if err := json.Unmarshal([]byte(val), &charInfo); err == nil {
				return &gamev1.GetCharacterResponse{
					Code:      0,
					Message:   "查询成功(来自缓存)",
					Character: &charInfo,
				}, nil
			}
		}
	}

	var (
		id        uint64
		charName  string
		level     int32
		attrsJSON string
	)
	err := s.db.QueryRowContext(ctx,
		"SELECT id, character_name, level, attributes FROM game_characters WHERE user_id = ?",
		req.UserId,
	).Scan(&id, &charName, &level, &attrsJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &gamev1.GetCharacterResponse{
				Code:    404,
				Message: "角色不存在",
			}, nil
		}
		return nil, fmt.Errorf("query character failed: %w", err)
	}

	var attrs CharacterAttributes
	_ = json.Unmarshal([]byte(attrsJSON), &attrs)

	charInfo := &gamev1.CharacterInfo{
		Id:             id,
		UserId:         req.UserId,
		CharacterName:  charName,
		Level:          level,
		Coins:          attrs.Coins,
		AttributesJson: attrsJSON,
	}

	// 回写缓存
	if s.rdb != nil {
		data, _ := json.Marshal(charInfo)
		_ = s.rdb.Set(ctx, cacheKey, string(data), time.Hour*24).Err()
	}

	return &gamev1.GetCharacterResponse{
		Code:      0,
		Message:   "查询成功",
		Character: charInfo,
	}, nil
}

// UpdateCharacterLevel 角色等级升级
func (s *GameService) UpdateCharacterLevel(ctx context.Context, req *gamev1.UpdateCharacterLevelRequest) (*gamev1.UpdateCharacterLevelResponse, error) {
	if req.UserId == 0 || req.NewLevel <= 0 {
		return &gamev1.UpdateCharacterLevelResponse{
			Code:    400,
			Message: "用户ID与新等级参数无效",
		}, nil
	}

	res, err := s.db.ExecContext(ctx,
		"UPDATE game_characters SET level = ?, updated_at = NOW() WHERE user_id = ?",
		req.NewLevel, req.UserId,
	)
	if err != nil {
		return nil, fmt.Errorf("update level failed: %w", err)
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return &gamev1.UpdateCharacterLevelResponse{
			Code:    404,
			Message: "用户角色不存在，升级失败",
		}, nil
	}

	// 清除缓存
	if s.rdb != nil {
		cacheKey := fmt.Sprintf("game:char:%d", req.UserId)
		_ = s.rdb.Del(ctx, cacheKey).Err()
	}

	return &gamev1.UpdateCharacterLevelResponse{
		Code:    0,
		Message: fmt.Sprintf("等级已成功提升至 Lv.%d", req.NewLevel),
	}, nil
}

// CreateOrder 创建业务订单 (Saga 事务起点)
func (s *GameService) CreateOrder(ctx context.Context, req *gamev1.CreateOrderRequest) (*gamev1.CreateOrderResponse, error) {
	if req.UserId == 0 || req.Amount <= 0 {
		return &gamev1.CreateOrderResponse{
			Code:    400,
			Message: "用户ID与订单金额必须大于0",
		}, nil
	}

	channel := req.Channel
	if channel == "" {
		channel = "web"
	}

	orderNo := fmt.Sprintf("ord_%s", uuid.New().String()[:12])

	_, err := s.db.ExecContext(ctx,
		"INSERT INTO orders (user_id, order_no, amount, status, channel, created_at, updated_at) VALUES (?, ?, ?, 0, ?, NOW(), NOW())",
		req.UserId, orderNo, req.Amount, channel,
	)
	if err != nil {
		return nil, fmt.Errorf("insert order failed: %w", err)
	}

	// 发布订单创建事件至 RabbitMQ
	if s.mq != nil {
		_ = s.mq.PublishJSON(ctx, mq.ExchangeSagaOrder, mq.RoutingKeyOrderCreated, mq.OrderEvent{
			OrderNo:   orderNo,
			UserID:    req.UserId,
			Amount:    req.Amount,
			Status:    "CREATED",
			Timestamp: time.Now().Unix(),
		})
	}

	return &gamev1.CreateOrderResponse{
		Code:    0,
		Message: "订单创建成功(待支付)",
		OrderNo: orderNo,
	}, nil
}

// ProcessPayment 处理订单支付与长事务 Saga / 补偿回滚
func (s *GameService) ProcessPayment(ctx context.Context, req *gamev1.ProcessPaymentRequest) (*gamev1.ProcessPaymentResponse, error) {
	if req.OrderNo == "" || req.UserId == 0 {
		return &gamev1.ProcessPaymentResponse{
			Code:    400,
			Message: "订单号与用户ID不能为空",
			Status:  "INVALID_PARAM",
		}, nil
	}

	// 查询订单
	var (
		amount float64
		status int
	)
	err := s.db.QueryRowContext(ctx,
		"SELECT amount, status FROM orders WHERE order_no = ? AND user_id = ?",
		req.OrderNo, req.UserId,
	).Scan(&amount, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return &gamev1.ProcessPaymentResponse{
				Code:    404,
				Message: "订单不存在",
				Status:  "NOT_FOUND",
			}, nil
		}
		return nil, fmt.Errorf("query order failed: %w", err)
	}

	if status != 0 {
		return &gamev1.ProcessPaymentResponse{
			Code:    400,
			Message: fmt.Sprintf("订单已被处理 (当前状态: %d)", status),
			Status:  "ALREADY_PROCESSED",
		}, nil
	}

	// 模拟 Saga 长事务故障与补偿回滚分支
	if req.SimulateFailure {
		// 1. 更新订单为失败/补偿状态 (status = 2)
		_, _ = s.db.ExecContext(ctx,
			"UPDATE orders SET status = 2, updated_at = NOW() WHERE order_no = ?",
			req.OrderNo,
		)

		// 2. 发送 Saga 补偿事件 order.compensated 到 RabbitMQ
		if s.mq != nil {
			_ = s.mq.PublishJSON(ctx, mq.ExchangeSagaOrder, mq.RoutingKeyOrderCompensated, mq.OrderEvent{
				OrderNo:   req.OrderNo,
				UserID:    req.UserId,
				Amount:    amount,
				Status:    "COMPENSATED",
				Reason:    "下游服务库存扣减超时，触发 Saga 自动补偿回滚",
				Timestamp: time.Now().Unix(),
			})
		}

		return &gamev1.ProcessPaymentResponse{
			Code:    500,
			Message: "支付失败：下游发货超时，已自动触发 Saga 补偿机制，订单回滚完成",
			Status:  "COMPENSATED",
		}, nil
	}

	// 正常 Saga 事务提交分支
	// 1. 更新订单状态为支付成功 (status = 1)
	_, err = s.db.ExecContext(ctx,
		"UPDATE orders SET status = 1, updated_at = NOW() WHERE order_no = ?",
		req.OrderNo,
	)
	if err != nil {
		return nil, fmt.Errorf("update order paid failed: %w", err)
	}

	// 2. 奖励角色对应金币 (如 1元 = 10金币)
	rewardCoins := int64(amount * 10)
	var attrsJSON string
	if err := s.db.QueryRowContext(ctx, "SELECT attributes FROM game_characters WHERE user_id = ?", req.UserId).Scan(&attrsJSON); err == nil {
		var attrs CharacterAttributes
		_ = json.Unmarshal([]byte(attrsJSON), &attrs)
		attrs.Coins += rewardCoins
		newAttrsJSON, _ := json.Marshal(attrs)
		_, _ = s.db.ExecContext(ctx,
			"UPDATE game_characters SET attributes = ?, updated_at = NOW() WHERE user_id = ?",
			string(newAttrsJSON), req.UserId,
		)
		// 清除角色缓存
		if s.rdb != nil {
			_ = s.rdb.Del(ctx, fmt.Sprintf("game:char:%d", req.UserId)).Err()
		}
	}

	// 3. 发布订单支付成功事件到 RabbitMQ
	if s.mq != nil {
		_ = s.mq.PublishJSON(ctx, mq.ExchangeSagaOrder, mq.RoutingKeyOrderPaid, mq.OrderEvent{
			OrderNo:   req.OrderNo,
			UserID:    req.UserId,
			Amount:    amount,
			Status:    "PAID",
			Reason:    fmt.Sprintf("支付成功，已发放金币 %d", rewardCoins),
			Timestamp: time.Now().Unix(),
		})
	}

	return &gamev1.ProcessPaymentResponse{
		Code:    0,
		Message: fmt.Sprintf("支付成功！Saga 事务提交完成，已自动到账 %d 金币", rewardCoins),
		Status:  "PAID",
	}, nil
}

// ReportGameEvent 业务遥测与埋点上报
func (s *GameService) ReportGameEvent(ctx context.Context, req *gamev1.ReportGameEventRequest) (*gamev1.ReportGameEventResponse, error) {
	if req.UserId == 0 || req.EventType == "" {
		return &gamev1.ReportGameEventResponse{
			Code:    400,
			Message: "用户ID与事件类型不能为空",
		}, nil
	}

	payload := req.PayloadJson
	if payload == "" {
		payload = "{}"
	}

	// 1. 写入 MySQL events_log 基础埋点流水
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO events_log (user_id, event_type, payload, created_at) VALUES (?, ?, ?, NOW())",
		req.UserId, req.EventType, payload,
	)
	if err != nil {
		return nil, fmt.Errorf("insert event log failed: %w", err)
	}

	// 2. 投递异步遥测事件到 RabbitMQ Topic
	if s.mq != nil {
		_ = s.mq.PublishJSON(ctx, mq.ExchangeGameEvents, mq.RoutingKeyGameTelemetry, mq.TelemetryEvent{
			UserID:    req.UserId,
			EventType: req.EventType,
			Payload:   payload,
			Timestamp: time.Now().Unix(),
		})
	}

	return &gamev1.ReportGameEventResponse{
		Code:    0,
		Message: "业务埋点与遥测事件上报成功",
	}, nil
}
