#!/usr/bin/env bash
set -e

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$DIR"

echo "=== [1/4] 编译微服务与网关二进制 ==="
go build -o bin/auth-service ./services/auth-service/cmd/server
go build -o bin/gateway ./services/gateway/cmd/server

echo "=== [2/4] 启动 auth-service 与 tRPC-Gateway ==="
./bin/auth-service -conf ./services/auth-service/conf/trpc_go.yaml > /tmp/auth-service.log 2>&1 &
AUTH_PID=$!

./bin/gateway -conf ./services/gateway/conf/trpc_go.yaml -router ./services/gateway/conf/router.yaml > /tmp/gateway.log 2>&1 &
GW_PID=$!

cleanup() {
    echo "清理测试进程 (PID: $AUTH_PID, $GW_PID)..."
    kill -9 $AUTH_PID $GW_PID 2>/dev/null || true
}
trap cleanup EXIT

# 等待服务初始化与北极星注册完成
sleep 2

echo "=== [3/4] 执行全链路端到端 HTTP -> Gateway -> tRPC -> MySQL/Redis 验收测试 ==="

TEST_USER="e2e_tester_$(date +%s)"
TEST_PWD="ProductionPassword_2026!"

# 1. 注册
echo ">> 步骤 1: POST /api/v1/auth/register"
REG_RESP=$(curl -s -X POST http://127.0.0.1:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d "{\"username\": \"$TEST_USER\", \"password\": \"$TEST_PWD\", \"email\": \"$TEST_USER@gametool.local\"}")
echo "注册响应: $REG_RESP"
USER_ID=$(echo "$REG_RESP" | grep -o '"user_id":"[^"]*' | cut -d'"' -f4)
if [ -z "$USER_ID" ]; then
    echo "❌ 注册失败: 未获取到 user_id"
    exit 1
fi
echo "✅ 用户注册成功! UserID: $USER_ID"

# 2. 登录
echo ">> 步骤 2: POST /api/v1/auth/login"
LOGIN_RESP=$(curl -s -X POST http://127.0.0.1:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d "{\"username\": \"$TEST_USER\", \"password\": \"$TEST_PWD\"}")
echo "登录响应: $LOGIN_RESP"
TOKEN=$(echo "$LOGIN_RESP" | grep -o '"token":"[^"]*' | cut -d'"' -f4)
if [ -z "$TOKEN" ]; then
    echo "❌ 登录失败: 未获取到 Token"
    exit 1
fi
echo "✅ 用户登录成功! 分布式 Session Token: $TOKEN"

# 3. 校验 Token
echo ">> 步骤 3: POST /api/v1/auth/validate"
VAL_RESP=$(curl -s -X POST http://127.0.0.1:8080/api/v1/auth/validate \
  -H "Content-Type: application/json" \
  -d "{\"token\": \"$TOKEN\"}")
echo "校验响应: $VAL_RESP"
if ! echo "$VAL_RESP" | grep -q '"valid":true'; then
    echo "❌ Token 校验失败: $VAL_RESP"
    exit 1
fi
echo "✅ Token 鉴权校验通过: 用户身份与 Session 状态一致!"

echo "=== [4/4] 阶段二全链路测试全部通过 100% 达成! ==="
