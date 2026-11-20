package clickhouse

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL string
	client  *http.Client
}

type GameEventRecord struct {
	EventID     string  `json:"event_id,omitempty"`
	EventTime   string  `json:"event_time,omitempty"`
	UserID      uint64  `json:"user_id"`
	CharacterID uint64  `json:"character_id"`
	EventType   string  `json:"event_type"`
	Channel     string  `json:"channel,omitempty"`
	DeviceType  string  `json:"device_type,omitempty"`
	AppVersion  string  `json:"app_version,omitempty"`
	Amount      float64 `json:"amount,omitempty"`
	CoinsDelta  int64   `json:"coins_delta,omitempty"`
	Level       uint32  `json:"level,omitempty"`
	DurationSec uint32  `json:"duration_sec,omitempty"`
	Payload     string  `json:"payload,omitempty"`
}

func InitClickHouse(baseURL string) (*Client, error) {
	if baseURL == "" {
		baseURL = "http://127.0.0.1:8123"
	}
	resp, err := http.Get(baseURL + "/ping")
	if err != nil {
		return nil, fmt.Errorf("ping clickhouse failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("clickhouse ping unexpected status: %d", resp.StatusCode)
	}
	return &Client{
		baseURL: baseURL,
		client:  &http.Client{Timeout: 5 * time.Second},
	}, nil
}

// InsertEvent 写入单条埋点事件 (使用 JSONEachRow 格式)
func (c *Client) InsertEvent(ctx context.Context, event GameEventRecord) error {
	return c.InsertBatch(ctx, []GameEventRecord{event})
}

// InsertBatch 批量写入埋点事件
func (c *Client) InsertBatch(ctx context.Context, events []GameEventRecord) error {
	if len(events) == 0 {
		return nil
	}
	var buf bytes.Buffer
	for _, e := range events {
		if e.EventTime == "" {
			e.EventTime = time.Now().Format("2006-01-02 15:04:05.000")
		}
		if e.Payload == "" {
			e.Payload = "{}"
		}
		data, err := json.Marshal(e)
		if err != nil {
			return err
		}
		buf.Write(data)
		buf.WriteString("\n")
	}

	qURL := c.baseURL + "/?query=" + url.QueryEscape("INSERT INTO analytics.game_events FORMAT JSONEachRow")
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, qURL, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("clickhouse insert failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("clickhouse insert error (status %d): %s", resp.StatusCode, string(body))
	}
	return nil
}

// Query 执行 ClickHouse SQL 查询并返回格式化结果
func (c *Client) Query(ctx context.Context, query string) (string, error) {
	qURL := c.baseURL + "/?query=" + url.QueryEscape(query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, qURL, nil)
	if err != nil {
		return "", err
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return string(data), err
}
