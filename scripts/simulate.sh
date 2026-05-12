#!/usr/bin/env bash
# 电梯调度系统 - 用户操作模拟脚本
# 通过 HTTP API 模拟真实用户操作，覆盖主要流程和边界情况。
#
# 使用前请先启动后端服务：
#   go run ./cmd/server
#
# 运行时可以实时打开 http://localhost:8080 观察电梯状态变化。

set -euo pipefail

BASE="http://localhost:8080"

# ── 工具函数 ──────────────────────────────────────────

now() { date '+%H:%M:%S'; }
say() { echo "[$(now)] $*"; }

# 发送 POST JSON 请求，检查 HTTP 状态码
post() {
    local url="$1" body="$2" expect="${3:-201}"
    local http_code
    http_code=$(curl -s -o /tmp/resp_body -w '%{http_code}' \
        -X POST "$url" \
        -H 'Content-Type: application/json' \
        -d "$body")
    if [ "$http_code" != "$expect" ]; then
        say "  → HTTP $http_code (预期 $expect) — $(cat /tmp/resp_body)"
    else
        say "  ✓ $(echo "$body" | head -c 80)... → $http_code"
    fi
}

get() {
    local url="$1"
    curl -s "$url" | python3 -m json.tool 2>/dev/null | head -30 || curl -s "$url"
}

# ── 前置检查 ──────────────────────────────────────────

say "检查服务状态..."
if ! curl -s -o /dev/null -w '' "$BASE/api/health"; then
    echo "错误: 服务未启动，请先执行: go run ./cmd/server"
    exit 1
fi
say "服务正常运行"

# 记录开始前的 db 文件列表，便于最后对比
say "运行前 data/ 中的 db 文件:"
ls -la data/*.db 2>/dev/null || echo "  (无)"

# ── 1. 基础查询 ───────────────────────────────────────

say ""
say "═══ 1. 基础信息查询 ═══"

say "GET /api/config →"
curl -s "$BASE/api/config" | python3 -m json.tool

say "GET /api/state (初始状态) →"
get "$BASE/api/state"

# ── 2. 早高峰模拟（大量 hall 请求）───────────────────

say ""
say "═══ 2. 早高峰 — 各楼层按下按钮 ═══"

say "1 楼 3 位乘客按上行"
post "$BASE/api/request" '{"floor":1,"direction":"up","kind":"hall","elevatorId":0}'
post "$BASE/api/request" '{"floor":1,"direction":"up","kind":"hall","elevatorId":0}'
post "$BASE/api/request" '{"floor":1,"direction":"up","kind":"hall","elevatorId":0}'

say "不同楼层乘客按下按钮"
post "$BASE/api/request" '{"floor":3,"direction":"up","kind":"hall","elevatorId":0}'
post "$BASE/api/request" '{"floor":5,"direction":"down","kind":"hall","elevatorId":0}'
post "$BASE/api/request" '{"floor":8,"direction":"down","kind":"hall","elevatorId":0}'
post "$BASE/api/request" '{"floor":10,"direction":"up","kind":"hall","elevatorId":0}'
post "$BASE/api/request" '{"floor":12,"direction":"down","kind":"hall","elevatorId":0}'
post "$BASE/api/request" '{"floor":15,"direction":"up","kind":"hall","elevatorId":0}'
post "$BASE/api/request" '{"floor":18,"direction":"down","kind":"hall","elevatorId":0}'

say "等待电梯运行 (4s)..."
sleep 4

say "查看状态:"
get "$BASE/api/state"

# ── 3. 乘客进入电梯，按目标楼层 (cabin 请求) ─────────

say ""
say "═══ 3. 乘客进入电梯 — cabin 请求 ═══"

say "电梯 #1 内的乘客按 7 楼"
post "$BASE/api/request" '{"floor":7,"direction":"idle","kind":"cabin","elevatorId":1}'
say "电梯 #2 内的乘客按 14 楼"
post "$BASE/api/request" '{"floor":14,"direction":"idle","kind":"cabin","elevatorId":2}'
say "电梯 #3 内的乘客按 20 楼（顶层）"
post "$BASE/api/request" '{"floor":20,"direction":"idle","kind":"cabin","elevatorId":3}'

say "等待电梯运行 (5s)..."
sleep 5
get "$BASE/api/state"

# ── 4. 边界情况测试 ───────────────────────────────────

say ""
say "═══ 4. 边界情况 — 预期应被拒绝 ═══"

say "1 楼按下行 → 应 400"
post "$BASE/api/request" '{"floor":1,"direction":"down","kind":"hall","elevatorId":0}' 400

say "顶层(20F)按上行 → 应 400"
post "$BASE/api/request" '{"floor":20,"direction":"up","kind":"hall","elevatorId":0}' 400

say "hall 请求方向 idle → 应 400"
post "$BASE/api/request" '{"floor":10,"direction":"idle","kind":"hall","elevatorId":0}' 400

say "cabin 请求方向 up → 应 400"
post "$BASE/api/request" '{"floor":10,"direction":"up","kind":"cabin","elevatorId":1}' 400

say "楼层超出范围(0) → 应 400"
post "$BASE/api/request" '{"floor":0,"direction":"up","kind":"hall","elevatorId":0}' 400

say "楼层超出范围(99) → 应 400"
post "$BASE/api/request" '{"floor":99,"direction":"up","kind":"hall","elevatorId":0}' 400

say "cabin 请求电梯ID无效(99) → 应 400"
post "$BASE/api/request" '{"floor":5,"direction":"idle","kind":"cabin","elevatorId":99}' 400

say "cabin 请求电梯ID为0 → 应 400"
post "$BASE/api/request" '{"floor":5,"direction":"idle","kind":"cabin","elevatorId":0}' 400

say "请求体为空 → 应 400"
post "$BASE/api/request" '{}' 400

say "非法方向 → 应 400"
post "$BASE/api/request" '{"floor":5,"direction":"left","kind":"hall","elevatorId":0}' 400

say "非法 kind → 应 400"
post "$BASE/api/request" '{"floor":5,"direction":"up","kind":"outside","elevatorId":0}' 400

say "多余字段 → 应 400"
post "$BASE/api/request" '{"floor":5,"direction":"up","kind":"hall","elevatorId":0,"extra":123}' 400

say "GET 请求到 POST 端点 → 应 405"
http_code=$(curl -s -o /tmp/resp_body -w '%{http_code}' "$BASE/api/request")
say "  → HTTP $http_code (预期 405)"

# ── 5. 同一楼层多个同方向请求（压力） ────────────────

say ""
say "═══ 5. 同楼层同方向密集请求 ═══"

for i in $(seq 1 5); do
    post "$BASE/api/request" '{"floor":4,"direction":"up","kind":"hall","elevatorId":0}'
done

say "等待处理 (4s)..."
sleep 4
get "$BASE/api/state"

# ── 6. 切换调度算法（会产生新的 db 文件） ────────────

say ""
say "═══ 6. 切换调度算法 ═══"
say "注意: 通过 API 切换调度器会重建系统，创建新数据库文件"

say "当前 data/ 中的 db 文件:"
ls -la data/*.db 2>/dev/null

say "切换到 LOOK..."
post "$BASE/api/scheduler" '{"name":"look"}' 200

say "切换到后添加请求..."
post "$BASE/api/request" '{"floor":2,"direction":"up","kind":"hall","elevatorId":0}'
post "$BASE/api/request" '{"floor":9,"direction":"down","kind":"hall","elevatorId":0}'
post "$BASE/api/request" '{"floor":16,"direction":"up","kind":"hall","elevatorId":0}'
post "$BASE/api/request" '{"floor":19,"direction":"down","kind":"hall","elevatorId":0}'
sleep 5

say "切换到 FCFS..."
post "$BASE/api/scheduler" '{"name":"fcfs"}' 200
post "$BASE/api/request" '{"floor":3,"direction":"up","kind":"hall","elevatorId":0}'
post "$BASE/api/request" '{"floor":11,"direction":"down","kind":"hall","elevatorId":0}'
sleep 5

say "切换到 nearest-idle..."
post "$BASE/api/scheduler" '{"name":"nearest-idle"}' 200
post "$BASE/api/request" '{"floor":6,"direction":"up","kind":"hall","elevatorId":0}'
post "$BASE/api/request" '{"floor":13,"direction":"down","kind":"hall","elevatorId":0}'
sleep 5

say "切换到 first-available..."
post "$BASE/api/scheduler" '{"name":"first-available"}' 200
post "$BASE/api/request" '{"floor":8,"direction":"up","kind":"hall","elevatorId":0}'
post "$BASE/api/request" '{"floor":17,"direction":"down","kind":"hall","elevatorId":0}'
sleep 5

say "切换回 scan..."
post "$BASE/api/scheduler" '{"name":"scan"}' 200

say "切换到无效调度器 → 应 400"
post "$BASE/api/scheduler" '{"name":"invalid-scheduler"}' 400

# ── 7. 修改楼层数和电梯数 ─────────────────────────────

say ""
say "═══ 7. 修改系统参数 ═══"

say "修改楼层数为 30..."
post "$BASE/api/floor-count" '{"floorCount":30}' 200
say "在 25 楼按上行（新楼层范围）"
post "$BASE/api/request" '{"floor":25,"direction":"up","kind":"hall","elevatorId":0}'
say "在 29 楼按下行"
post "$BASE/api/request" '{"floor":29,"direction":"down","kind":"hall","elevatorId":0}'
sleep 3

say "修改电梯数为 3..."
post "$BASE/api/elevator-count" '{"elevatorCount":3}' 200
sleep 3

say "恢复为 20 层 5 部电梯..."
post "$BASE/api/floor-count" '{"floorCount":20}' 200
sleep 1
post "$BASE/api/elevator-count" '{"elevatorCount":5}' 200
sleep 1

say "添加最后的请求验证正常运行..."
post "$BASE/api/request" '{"floor":5,"direction":"up","kind":"hall","elevatorId":0}'
post "$BASE/api/request" '{"floor":10,"direction":"down","kind":"hall","elevatorId":0}'
sleep 4

# ── 8. 最终状态 ───────────────────────────────────────

say ""
say "═══ 8. 最终系统状态 ═══"
say "GET /api/state →"
get "$BASE/api/state"

# ── 9. 检查数据库文件 ─────────────────────────────────

say ""
say "═══ 9. 数据库文件检查 ═══"
say "data/ 目录中的所有 db 文件:"
ls -la data/*.db

say ""
echo "────────────────────────────────────────────────"
echo "  模拟完成。可通过以下方式验证数据:"
echo "  sqlite3 data/requests.db '.tables'"
echo "  sqlite3 data/requests.db 'SELECT COUNT(*) FROM completed_requests;'"
echo "  sqlite3 data/requests.db 'SELECT * FROM completed_requests LIMIT 5;'"
echo "────────────────────────────────────────────────"
