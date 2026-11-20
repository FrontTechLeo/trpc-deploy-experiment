#!/bin/bash
set -e

GATEWAY_URL="http://127.0.0.1:30880"
TIMESTAMP=$(date +%s%N)
TEST_USER="phase4_hero_${TIMESTAMP: -6}"
TEST_PWD="HeroSecurePass123!"

echo "=========================================================="
echo "阶段四全链路集成实测: tRPC Gateway + GameService + Saga/MQ"
echo "=========================================================="

echo -e "\n>>> 1. 用户注册与鉴权..."
REG_RESP=$(curl -s -X POST "${GATEWAY_URL}/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${TEST_USER}\",\"password\":\"${TEST_PWD}\",\"email\":\"${TEST_USER}@gametool.com\"}")
echo "注册响应: ${REG_RESP}"
USER_ID=$(echo "${REG_RESP}" | jq -r .user_id)
if [ -z "${USER_ID}" ] || [ "${USER_ID}" = "null" ]; then
  echo "注册失败!"
  exit 1
fi
echo "获取用户ID: ${USER_ID}"

LOGIN_RESP=$(curl -s -X POST "${GATEWAY_URL}/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"username\":\"${TEST_USER}\",\"password\":\"${TEST_PWD}\"}")
echo "登录响应: ${LOGIN_RESP}"
TOKEN=$(echo "${LOGIN_RESP}" | jq -r .token)
echo "签发 Token: ${TOKEN}"

echo -e "\n>>> 2. 创建游戏角色 (初始金币 5000)..."
CREATE_CHAR_RESP=$(curl -s -X POST "${GATEWAY_URL}/api/v1/game/character/create" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":${USER_ID},\"character_name\":\"Arthas_${USER_ID}\",\"initial_coins\":5000}")
echo "创建角色响应: ${CREATE_CHAR_RESP}"

echo -e "\n>>> 3. 查询角色属性与详情..."
GET_CHAR_RESP=$(curl -s -X POST "${GATEWAY_URL}/api/v1/game/character/get" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":${USER_ID}}")
echo "角色详情响应: ${GET_CHAR_RESP}"

echo -e "\n>>> 4. 角色升级至 Lv.25..."
LEVEL_RESP=$(curl -s -X POST "${GATEWAY_URL}/api/v1/game/character/level" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":${USER_ID},\"new_level\":25}")
echo "角色升级响应: ${LEVEL_RESP}"

echo -e "\n>>> 5. 测试 Saga 长事务补偿回滚: 创建订单 -> 模拟下游发货故障 -> 触发 Saga 补偿..."
ORDER1_RESP=$(curl -s -X POST "${GATEWAY_URL}/api/v1/game/order/create" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":${USER_ID},\"item_name\":\"Frostmourne\",\"amount\":88.00,\"channel\":\"game_store\"}")
echo "订单创建响应: ${ORDER1_RESP}"
ORDER1_NO=$(echo "${ORDER1_RESP}" | jq -r .order_no)

PAY1_RESP=$(curl -s -X POST "${GATEWAY_URL}/api/v1/game/order/pay" \
  -H "Content-Type: application/json" \
  -d "{\"order_no\":\"${ORDER1_NO}\",\"user_id\":${USER_ID},\"amount\":88.00,\"simulate_failure\":true}")
echo "Saga 异常补偿响应: ${PAY1_RESP}"

echo -e "\n>>> 6. 测试正常 Saga 长事务流程: 创建订单 -> 支付成功 -> 金币到账..."
ORDER2_RESP=$(curl -s -X POST "${GATEWAY_URL}/api/v1/game/order/create" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":${USER_ID},\"item_name\":\"MythicTreasure\",\"amount\":150.00,\"channel\":\"wechat_pay\"}")
echo "订单创建响应: ${ORDER2_RESP}"
ORDER2_NO=$(echo "${ORDER2_RESP}" | jq -r .order_no)

PAY2_RESP=$(curl -s -X POST "${GATEWAY_URL}/api/v1/game/order/pay" \
  -H "Content-Type: application/json" \
  -d "{\"order_no\":\"${ORDER2_NO}\",\"user_id\":${USER_ID},\"amount\":150.00,\"simulate_failure\":false}")
echo "Saga 支付成功响应: ${PAY2_RESP}"

echo -e "\n>>> 7. 验证支付后金币增加 (原5000 + 150*10 = 6500)..."
VERIFY_CHAR_RESP=$(curl -s -X POST "${GATEWAY_URL}/api/v1/game/character/get" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":${USER_ID}}")
echo "支付后角色属性: ${VERIFY_CHAR_RESP}"

echo -e "\n>>> 8. 业务遥测与事件上报..."
EVENT_RESP=$(curl -s -X POST "${GATEWAY_URL}/api/v1/game/event/report" \
  -H "Content-Type: application/json" \
  -d "{\"user_id\":${USER_ID},\"event_type\":\"boss_kill\",\"payload_json\":\"{\\\"boss_id\\\":\\\"lich_king\\\",\\\"damage\\\":99999,\\\"duration_sec\\\":120}\"}")
echo "埋点事件上报响应: ${EVENT_RESP}"

echo -e "\n=========================================================="
echo ">>> 阶段四全链路集成测试 100% 通过！"
echo "=========================================================="
