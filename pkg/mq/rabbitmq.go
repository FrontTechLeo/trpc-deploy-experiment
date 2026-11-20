package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"trpc.group/trpc-go/trpc-go/log"
)

// 常量定义：Exchange 与 RoutingKeys
const (
	ExchangeGameEvents = "gametool.events.topic"
	ExchangeSagaOrder  = "gametool.saga.order"

	RoutingKeyOrderCreated     = "order.created"
	RoutingKeyOrderPaid        = "order.paid"
	RoutingKeyOrderCompensated = "order.compensated"
	RoutingKeyGameTelemetry    = "game.telemetry"

	QueueSagaPayment = "gametool.saga.payment.queue"
	QueueTelemetry   = "gametool.telemetry.queue"
)

// OrderEvent Saga 事务事件结构
type OrderEvent struct {
	OrderNo   string  `json:"order_no"`
	UserID    uint64  `json:"user_id"`
	Amount    float64 `json:"amount"`
	Status    string  `json:"status"` // CREATED, PAID, FAILED, COMPENSATED
	Reason    string  `json:"reason,omitempty"`
	Timestamp int64   `json:"timestamp"`
}

// TelemetryEvent 业务遥测事件结构
type TelemetryEvent struct {
	UserID    uint64 `json:"user_id"`
	EventType string `json:"event_type"`
	Payload   string `json:"payload"`
	Timestamp int64  `json:"timestamp"`
}

// RabbitMQ 客户端封装
type RabbitMQ struct {
	conn    *amqp.Connection
	ch      *amqp.Channel
	url     string
	mu      sync.Mutex
	isClose bool
}

// InitRabbitMQ 初始化 RabbitMQ 客户端并创建默认交换机
func InitRabbitMQ(url string) (*RabbitMQ, error) {
	if url == "" {
		url = "amqp://guest:guest@127.0.0.1:5672/"
	}
	conn, err := amqp.Dial(url)
	if err != nil {
		return nil, fmt.Errorf("connect rabbitmq failed: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("open channel failed: %w", err)
	}

	client := &RabbitMQ{
		conn: conn,
		ch:   ch,
		url:  url,
	}

	// 声明默认 Topic 交换机
	if err := client.declareExchanges(); err != nil {
		client.Close()
		return nil, err
	}

	return client, nil
}

func (r *RabbitMQ) declareExchanges() error {
	// 声明 Saga 订单事件交换机
	err := r.ch.ExchangeDeclare(
		ExchangeSagaOrder,
		"topic",
		true,  // durable
		false, // auto-deleted
		false, // internal
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare saga exchange failed: %w", err)
	}

	// 声明游戏遥测埋点交换机
	err = r.ch.ExchangeDeclare(
		ExchangeGameEvents,
		"topic",
		true,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare telemetry exchange failed: %w", err)
	}

	return nil
}

// PublishJSON 发布 JSON 消息
func (r *RabbitMQ) PublishJSON(ctx context.Context, exchange, routingKey string, body interface{}) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("json marshal failed: %w", err)
	}

	return r.ch.PublishWithContext(
		ctx,
		exchange,
		routingKey,
		false, // mandatory
		false, // immediate
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Timestamp:    time.Now(),
			Body:         data,
		},
	)
}

// Consume 注册队列消费者
func (r *RabbitMQ) Consume(queue, exchange, routingKey string, handler func(msg []byte) error) error {
	// 声明队列
	q, err := r.ch.QueueDeclare(
		queue,
		true,  // durable
		false, // auto-delete
		false, // exclusive
		false, // no-wait
		nil,
	)
	if err != nil {
		return fmt.Errorf("declare queue failed: %w", err)
	}

	// 绑定交换机
	err = r.ch.QueueBind(
		q.Name,
		routingKey,
		exchange,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("bind queue failed: %w", err)
	}

	deliveries, err := r.ch.Consume(
		q.Name,
		"",    // consumer tag
		false, // auto-ack (手动确认，确保可靠性)
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("start consume failed: %w", err)
	}

	go func() {
		for d := range deliveries {
			if err := handler(d.Body); err != nil {
				log.Errorf("[RabbitMQ] handle message error: %v, body: %s", err, string(d.Body))
				// 消费失败重新入队或进入死信
				_ = d.Nack(false, false)
			} else {
				_ = d.Ack(false)
			}
		}
	}()

	return nil
}

// Close 关闭连接
func (r *RabbitMQ) Close() {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.isClose {
		return
	}
	r.isClose = true
	if r.ch != nil {
		_ = r.ch.Close()
	}
	if r.conn != nil {
		_ = r.conn.Close()
	}
}
