#!/usr/bin/env bash
# ==============================================================================
# lakehouse-sync.sh
# 湖仓一体离线归档与 DuckDB 联邦分析脚本:
# 1. 将 ClickHouse 实时事件表归档导出至 MinIO S3 Lakehouse (Parquet 列式格式)
# 2. 调用 DuckDB Serverless 分析引擎直接读取 S3 Parquet 文件完成离线报表计算
# 3. 演示 DuckDB 跨源联邦查询: 联合 TiDB 分布式数据库与 MinIO S3 湖仓数据
# ==============================================================================

set -euo pipefail

CH_HTTP="http://127.0.0.1:8123"
S3_ENDPOINT="127.0.0.1:9000"
MINIO_INTERNAL="http://local-minio:9000"
S3_BUCKET="gametool-lakehouse"
DATE_STR=$(date +%Y%m%d)
PARQUET_FILE="events/game_events_${DATE_STR}.parquet"
S3_INTERNAL_URL="${MINIO_INTERNAL}/${S3_BUCKET}/${PARQUET_FILE}"
S3_DUCKDB_PATH="s3://${S3_BUCKET}/${PARQUET_FILE}"

echo "=================================================================="
echo "  [1/3] 同步 ClickHouse 数据至 MinIO S3 Lakehouse (Parquet 列式存储)"
echo "=================================================================="
echo "目标路径: ${S3_DUCKDB_PATH}"

# 确保 MinIO bucket 目录存在
docker exec -i local-minio mkdir -p "/data/${S3_BUCKET}/events" >/dev/null 2>&1 || true

# 通过 ClickHouse S3 表函数导出 Parquet (开启覆盖写入)
curl -s -f -X POST "${CH_HTTP}/" --data-binary "
INSERT INTO FUNCTION s3('${S3_INTERNAL_URL}', 'minioadmin', 'minioadmin', 'Parquet')
SETTINGS s3_truncate_on_insert = 1
SELECT 
  toString(event_id) AS event_id,
  toDateTime(event_time) AS event_time,
  user_id,
  character_id,
  toString(event_type) AS event_type,
  toString(channel) AS channel,
  toString(device_type) AS device_type,
  toString(app_version) AS app_version,
  toFloat64(amount) AS amount,
  coins_delta,
  level,
  duration_sec,
  payload
FROM analytics.game_events;
"

echo "✓ ClickHouse -> MinIO S3 Parquet 数据归档完成！"
echo ""

echo "=================================================================="
echo "  [2/3] DuckDB Serverless 引擎直接分析 MinIO S3 Parquet 湖仓数据"
echo "=================================================================="

duckdb -box -c "
INSTALL httpfs;
LOAD httpfs;
SET s3_endpoint='${S3_ENDPOINT}';
SET s3_access_key_id='minioadmin';
SET s3_secret_access_key='minioadmin';
SET s3_use_ssl=false;
SET s3_url_style='path';

-- 1. 事件概览与漏斗指标
SELECT 
  event_type AS '事件类型',
  count(*) AS '触发次数',
  count(DISTINCT user_id) AS '独立用户数(UV)',
  round(sum(amount), 2) AS '产生金额(¥)',
  round(avg(level), 1) AS '平均等级'
FROM '${S3_DUCKDB_PATH}'
GROUP BY event_type
ORDER BY count(*) DESC;

-- 2. 渠道与设备分布
SELECT 
  channel AS '推广渠道',
  device_type AS '设备类型',
  count(*) AS '上报总量',
  round(count(*) * 100.0 / sum(count(*)) OVER (), 2) || '%' AS '占比'
FROM '${S3_DUCKDB_PATH}'
GROUP BY channel, device_type
ORDER BY count(*) DESC;
"

echo ""
echo "=================================================================="
echo "  [3/3] DuckDB 跨源联邦查询: 联合 TiDB 分布式数据库 + MinIO 湖仓"
echo "=================================================================="

duckdb -box -c "
INSTALL httpfs;
LOAD httpfs;
INSTALL mysql;
LOAD mysql;

SET s3_endpoint='${S3_ENDPOINT}';
SET s3_access_key_id='minioadmin';
SET s3_secret_access_key='minioadmin';
SET s3_use_ssl=false;
SET s3_url_style='path';

ATTACH 'host=127.0.0.1 port=4000 user=root database=gametool_analytics' AS tidb (TYPE mysql);

-- 联邦 JOIN 查询：将 TiDB 玩家日结算表与 MinIO S3 Parquet 离线埋点数据动态关联
SELECT 
  t.user_id AS '用户ID',
  t.settle_date AS '结算日期',
  t.total_recharge AS 'TiDB充值总计(¥)',
  t.max_character_level AS 'TiDB最高等级',
  coalesce(s3.parquet_events_count, 0) AS 'S3埋点事件数',
  coalesce(round(s3.parquet_total_amount, 2), 0.00) AS 'S3埋点累计金额'
FROM tidb.player_daily_settlement t
LEFT JOIN (
  SELECT 
    user_id,
    count(*) AS parquet_events_count,
    sum(amount) AS parquet_total_amount
  FROM '${S3_DUCKDB_PATH}'
  GROUP BY user_id
) s3 ON t.user_id = s3.user_id
ORDER BY t.total_recharge DESC;
"

echo ""
echo "=================================================================="
echo "  ✓ DuckDB 湖仓一体分析与 TiDB 联邦查询全部执行成功！"
echo "=================================================================="
