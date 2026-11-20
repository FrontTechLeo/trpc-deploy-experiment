#!/usr/bin/env bash
# ==============================================================================
# test-phase5.sh
# 阶段五全套组件端到端集成测试:
# 1. TiDB 分布式数据库集群状态与数据查询
# 2. ClickHouse 实时 OLAP 引擎连通性与埋点聚合
# 3. MinIO S3 存储桶与 Parquet 列式数据验证
# 4. DuckDB Serverless 离线数仓报表与跨源联邦查询
# 5. 可观测监控全家桶健康巡检 (Prometheus, Jaeger, Loki, Grafana)
# ==============================================================================

set -euo pipefail

PASS=0
FAIL=0

check() {
  local desc="$1"
  shift
  echo -n "[CHECK] ${desc}... "
  if "$@" >/dev/null 2>&1; then
    echo "✓ PASS"
    PASS=$((PASS + 1))
  else
    echo "✗ FAIL"
    FAIL=$((FAIL + 1))
  fi
}

echo "=================================================================="
echo "          阶段五：分布式数据库、湖仓分析与监控套件集成测试          "
echo "=================================================================="

# 1. 容器存活检测
echo "--- 1. 核心容器存活状态 ---"
check "TiDB PD 服务容器" docker inspect -f '{{.State.Running}}' local-tidb-pd
check "TiDB TiKV 服务容器" docker inspect -f '{{.State.Running}}' local-tidb-tikv
check "TiDB Server 服务容器" docker inspect -f '{{.State.Running}}' local-tidb-server
check "ClickHouse OLAP 容器" docker inspect -f '{{.State.Running}}' local-clickhouse
check "MinIO 对象存储容器" docker inspect -f '{{.State.Running}}' local-minio
check "Prometheus 容器" docker inspect -f '{{.State.Running}}' prometheus
check "Jaeger 链路追踪容器" docker inspect -f '{{.State.Running}}' jaeger
check "Loki 日志聚合容器" docker inspect -f '{{.State.Running}}' loki
check "Grafana 可视化容器" docker inspect -f '{{.State.Running}}' grafana

# 2. TiDB 集群可用性与 HTAP 功能测试
echo ""
echo "--- 2. TiDB 分布式数据库可用性 ---"
check "TiDB SQL 端口连通与版本读取" docker exec -i local-mysql mysql -h local-tidb-server -P 4000 -u root -e "SELECT tidb_version();"
check "TiDB 玩家结算库表与数据读取" docker exec -i local-mysql mysql -h local-tidb-server -P 4000 -u root -e "SELECT count(*) FROM gametool_analytics.player_daily_settlement;"

# 3. ClickHouse 实时 OLAP 引擎测试
echo ""
echo "--- 3. ClickHouse 实时数仓服务 ---"
check "ClickHouse HTTP 接口健康状态" curl -sf "http://127.0.0.1:8123/ping"
check "ClickHouse 游戏事件表实时查询" curl -sf "http://127.0.0.1:8123/?query=SELECT+count(*)+FROM+analytics.game_events"

# 4. DuckDB 与 MinIO 湖仓一体离线报表测试
echo ""
echo "--- 4. DuckDB + MinIO S3 湖仓离线分析与跨源联邦 ---"
check "MinIO S3 数据桶存在性" docker exec -i local-minio test -d /data/gametool-lakehouse
check "DuckDB 执行湖仓一体归档与联邦查询脚本" bash "$(dirname "$0")/lakehouse-sync.sh"

# 5. 可观测监控体系健康检测
echo ""
echo "--- 5. 可观测监控体系 (Prometheus / Jaeger / Loki / Grafana) ---"
check "Prometheus 服务就绪 (Port 9090)" curl -sf "http://127.0.0.1:9090/-/healthy"
check "Jaeger Web UI (Port 16686)" curl -sf "http://127.0.0.1:16686/"
check "Loki 日志服务就绪 (Port 3100)" curl -sf "http://127.0.0.1:3100/ready"
check "Grafana API 健康状态 (Port 3002)" curl -sf "http://127.0.0.1:3002/api/health"
check "Grafana Prometheus 数据源配置" curl -sf -u admin:admin "http://127.0.0.1:3002/api/datasources/name/Prometheus"
check "Grafana Loki 数据源配置" curl -sf -u admin:admin "http://127.0.0.1:3002/api/datasources/name/Loki"

echo ""
echo "=================================================================="
echo "测试结果: ${PASS} 项通过, ${FAIL} 项失败"
echo "=================================================================="

if [ "${FAIL}" -eq 0 ]; then
  echo "🎉 阶段五所有组件测试 100% 全部通过！"
  exit 0
else
  echo "❌ 存在失败项，请检查相关服务配置！"
  exit 1
fi
