package clickhouse

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestClickHouse_InsertAndQuery(t *testing.T) {
	client, err := InitClickHouse("http://127.0.0.1:8123")
	if err != nil {
		t.Fatalf("InitClickHouse failed: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testEvents := []GameEventRecord{
		{
			UserID:      88801,
			CharacterID: 101,
			EventType:   "player_login",
			Channel:     "web",
			DeviceType:  "PC",
			AppVersion:  "v1.0.0",
			Payload:     `{"ip":"192.168.1.50","city":"Beijing"}`,
		},
		{
			UserID:      88801,
			CharacterID: 101,
			EventType:   "create_character",
			Channel:     "web",
			Level:       1,
			CoinsDelta:  1000,
			Payload:     `{"character_name":"TestPaladin"}`,
		},
		{
			UserID:      88801,
			CharacterID: 101,
			EventType:   "level_up",
			Channel:     "web",
			Level:       15,
			Payload:     `{"old_level":14,"new_level":15}`,
		},
		{
			UserID:      88801,
			CharacterID: 101,
			EventType:   "order_pay",
			Channel:     "alipay",
			Amount:      99.00,
			CoinsDelta:  990,
			Payload:     `{"order_no":"ord_test_01","item":"GoldPack"}`,
		},
		{
			UserID:      88801,
			CharacterID: 101,
			EventType:   "dungeon_pass",
			Channel:     "web",
			DurationSec: 185,
			Level:       15,
			Payload:     `{"dungeon_id":201,"stars":3}`,
		},
	}

	err = client.InsertBatch(ctx, testEvents)
	if err != nil {
		t.Fatalf("InsertBatch failed: %v", err)
	}

	// 查询验证
	res, err := client.Query(ctx, "SELECT count(*) FROM analytics.game_events WHERE user_id = 88801 FORMAT TabSeparated")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}
	t.Logf("Rows count for user 88801: %s", strings.TrimSpace(res))
	if strings.TrimSpace(res) == "0" {
		t.Fatalf("Expected at least 5 rows, got 0")
	}

	// 聚合分析查询
	agg, err := client.Query(ctx, "SELECT event_type, count(*), sum(amount) FROM analytics.game_events WHERE user_id = 88801 GROUP BY event_type FORMAT TabSeparated")
	if err != nil {
		t.Fatalf("Aggregate query failed: %v", err)
	}
	t.Logf("Aggregate results:\n%s", agg)
}
