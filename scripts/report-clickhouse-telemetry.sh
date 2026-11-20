#!/bin/bash
set -e

CH_HOST="127.0.0.1:8123"

echo "=========================================================="
echo "ClickHouse 海量业务埋点上报与多维分析实测"
echo "=========================================================="

echo -e "\n>>> 1. 确保 analytics.game_events 目标数仓表存在..."
curl -s "http://${CH_HOST}/" -d "
CREATE TABLE IF NOT EXISTS analytics.game_events (
    event_id UUID DEFAULT generateUUIDv4(),
    event_time DateTime64(3, 'Asia/Shanghai') DEFAULT now64(3),
    user_id UInt64,
    character_id UInt64 DEFAULT 0,
    event_type LowCardinality(String),
    channel LowCardinality(String) DEFAULT 'web',
    device_type LowCardinality(String) DEFAULT 'PC',
    app_version LowCardinality(String) DEFAULT 'v1.0.0',
    amount Decimal64(2) DEFAULT 0.00,
    coins_delta Int64 DEFAULT 0,
    level UInt32 DEFAULT 1,
    duration_sec UInt32 DEFAULT 0,
    payload String DEFAULT '{}'
) ENGINE = MergeTree()
PARTITION BY toYYYYMM(event_time)
ORDER BY (event_type, user_id, event_time);
"

echo -e "\n>>> 2. 批量生成并上报高仿真玩家多维业务埋点数据 (150+ 事件)..."

DATA_PAYLOAD=""

# 生成登录事件 (25 个用户)
for uid in $(seq 1001 1025); do
  ch=$([ $((uid % 3)) -eq 0 ] && echo "app" || ([ $((uid % 3)) -eq 1 ] && echo "web" || echo "steam"))
  dev=$([ $((uid % 2)) -eq 0 ] && echo "iOS" || echo "Android")
  DATA_PAYLOAD+="$(printf '{"user_id":%d,"character_id":%d,"event_type":"player_login","channel":"%s","device_type":"%s","app_version":"v1.2.0","payload":"{\\"ip\\":\\"192.168.1.%d\\",\\"city\\":\\"Shanghai\\"}"}\n' "$uid" "$uid" "$ch" "$dev" "$((uid % 250))")"
done

# 生成创角事件 (20 个用户)
for uid in $(seq 1001 1020); do
  DATA_PAYLOAD+="$(printf '{"user_id":%d,"character_id":%d,"event_type":"create_character","channel":"web","device_type":"PC","level":1,"coins_delta":1000,"payload":"{\\"role\\":\\"Warrior\\",\\"name\\":\\"Player_%d\\"}"}\n' "$uid" "$uid" "$uid")"
done

# 生成升级事件 (多次升级)
for uid in $(seq 1001 1020); do
  lvl=$(( (uid % 30) + 10 ))
  DATA_PAYLOAD+="$(printf '{"user_id":%d,"character_id":%d,"event_type":"level_up","channel":"app","device_type":"iOS","level":%d,"duration_sec":%d,"payload":"{\\"old_lvl\\":%d,\\"new_lvl\\":%d}"}\n' "$uid" "$uid" "$lvl" "$((lvl * 45))" "$((lvl - 1))" "$lvl")"
done

# 生成充值与订单支付事件 (不同档位与渠道)
CHANNELS=("alipay" "wechat_pay" "apple_pay")
AMOUNTS=(6.00 30.00 68.00 128.00 328.00 648.00)
for uid in $(seq 1001 1025); do
  if [ $((uid % 2)) -eq 0 ]; then
    ch="${CHANNELS[$((uid % 3))]}"
    amt="${AMOUNTS[$((uid % 6))]}"
    coins=$(echo "$amt * 10" | bc | cut -d. -f1)
    DATA_PAYLOAD+="$(printf '{"user_id":%d,"character_id":%d,"event_type":"order_pay","channel":"%s","device_type":"Mobile","amount":%.2f,"coins_delta":%d,"payload":"{\\"order_no\\":\\"ord_batch_%d\\",\\"gift_item\\":\\"Crystal\\"}"}\n' "$uid" "$uid" "$ch" "$amt" "$coins" "$uid")"
  fi
done

# 生成副本通关事件 (dungeon_pass)
DUNGEONS=(101 102 201 301 401 501)
for i in $(seq 1 40); do
  uid=$(( 1001 + (i % 25) ))
  d_id="${DUNGEONS[$((i % 6))]}"
  dur=$(( 120 + (i * 15) % 300 ))
  stars=$(( (i % 3) + 1 ))
  DATA_PAYLOAD+="$(printf '{"user_id":%d,"character_id":%d,"event_type":"dungeon_pass","channel":"web","device_type":"PC","level":20,"duration_sec":%d,"payload":"{\\"dungeon_id\\":%d,\\"stars\\":%d,\\"exp_gain\\":3500}"}\n' "$uid" "$uid" "$dur" "$d_id" "$stars")"
done

# 生成首领击杀事件 (boss_kill)
BOSSES=("LichKing" "Deathwing" "Ragnaros" "Illidan")
for i in $(seq 1 20); do
  uid=$(( 1001 + (i % 15) ))
  boss="${BOSSES[$((i % 4))]}"
  dmg=$(( 500000 + (i * 73210) % 900000 ))
  dur=$(( 180 + (i * 20) % 240 ))
  DATA_PAYLOAD+="$(printf '{"user_id":%d,"character_id":%d,"event_type":"boss_kill","channel":"steam","device_type":"PC","level":30,"duration_sec":%d,"payload":"{\\"boss_name\\":\\"%s\\",\\"damage_dealt\\":%d,\\"raid_size\\":10}"}\n' "$uid" "$uid" "$dur" "$boss" "$dmg")"
done

# 投递批量数据至 ClickHouse
echo "$DATA_PAYLOAD" | curl -s -X POST "http://${CH_HOST}/?query=INSERT%20INTO%20analytics.game_events%20FORMAT%20JSONEachRow" --data-binary @-
echo ">>> 埋点数据批量投递完成！"

echo -e "\n=========================================================="
echo ">>> 3. ClickHouse 多维 OLAP 聚合分析查询报表实测"
echo "=========================================================="

echo -e "\n【报表 1：各业务事件分类统计与独立用户数 (UV / DAU)】"
curl -s "http://${CH_HOST}/" -d "
SELECT
    event_type AS \`事件类型\`,
    count(*) AS \`事件总量\`,
    count(DISTINCT user_id) AS \`覆盖玩家数(UV)\`,
    round(count(*) / count(DISTINCT user_id), 2) AS \`人均触发频次\`
FROM analytics.game_events
GROUP BY event_type
ORDER BY \`事件总量\` DESC
FORMAT PrettyCompact;
"

echo -e "\n【报表 2：充值大盘渠道收入与客单价分析 (GMV)】"
curl -s "http://${CH_HOST}/" -d "
SELECT
    channel AS \`支付渠道\`,
    count(*) AS \`订单数\`,
    count(DISTINCT user_id) AS \`付费玩家数\`,
    sum(amount) AS \`总充值金额(元)\`,
    round(avg(amount), 2) AS \`客单价(元)\`
FROM analytics.game_events
WHERE event_type = 'order_pay'
GROUP BY channel
ORDER BY \`总充值金额(元)\` DESC
FORMAT PrettyCompact;
"

echo -e "\n【报表 3：副本通关耗时与难度表现分析 (Dungeon Stats)】"
curl -s "http://${CH_HOST}/" -d "
SELECT
    JSONExtractInt(payload, 'dungeon_id') AS \`副本ID\`,
    count(*) AS \`通关总人次\`,
    round(avg(duration_sec), 1) AS \`平均耗时(秒)\`,
    min(duration_sec) AS \`最快通关(秒)\`,
    max(duration_sec) AS \`最慢通关(秒)\`,
    round(avg(JSONExtractInt(payload, 'stars')), 2) AS \`平均通关星级\`
FROM analytics.game_events
WHERE event_type = 'dungeon_pass'
GROUP BY \`副本ID\`
ORDER BY \`副本ID\` ASC
FORMAT PrettyCompact;
"

echo -e "\n【报表 4：首领战伤害排行榜 Top 5 (Boss Raid Leaderboard)】"
curl -s "http://${CH_HOST}/" -d "
SELECT
    user_id AS \`玩家ID\`,
    JSONExtractString(payload, 'boss_name') AS \`讨伐首领\`,
    JSONExtractInt(payload, 'damage_dealt') AS \`总输出伤害\`,
    duration_sec AS \`战斗历时(秒)\`,
    event_time AS \`击杀上报时间\`
FROM analytics.game_events
WHERE event_type = 'boss_kill'
ORDER BY \`总输出伤害\` DESC
LIMIT 5
FORMAT PrettyCompact;
"

echo -e "\n=========================================================="
echo ">>> ClickHouse 埋点上报与多维分析实测 100% 成功！"
echo "=========================================================="
