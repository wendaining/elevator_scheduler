#!/usr/bin/env bash
# 电梯调度算法 - 性能对比脚本
# 对五种调度算法分别执行完全相同的请求负载，各自写入独立 db 文件，
# 方便后续比较不同算法的完成时间。
#
# 使用前请先启动后端服务：
#   go run ./cmd/server

set -euo pipefail

BASE="http://localhost:8080"
SCHEDULERS=("fcfs" "scan" "look" "first-available" "nearest-idle")

now() { date '+%H:%M:%S'; }
say() { echo "[$(now)] $*"; }

# ── 前置检查 ──────────────────────────────────────────

say "检查服务状态..."
if ! curl -s -o /dev/null -w '' "$BASE/api/health"; then
    echo "错误: 服务未启动，请先执行: go run ./cmd/server"
    exit 1
fi
say "服务正常"

# ── 生成测试请求负载 ─────────────────────────────────

# 定义一组固定的请求序列，对每个调度器完全一致。
# 覆盖场景：早高峰底部上行、高层下行、中层混合、同层多请求、cabin 请求。
generate_workload() {
    local total=0

    # 第 1 波 — 早高峰：低层大量上行
    for i in 1 1 1 1 2 2 2 3 3 3 4 4 5 5; do
        curl -s -o /dev/null -X POST "$BASE/api/request" \
            -H 'Content-Type: application/json' \
            -d "{\"floor\":$i,\"direction\":\"up\",\"kind\":\"hall\",\"elevatorId\":0}"
        ((total++))
    done

    # 第 2 波 — 高层下行
    for i in 18 18 17 17 16 15 14 14 13 12 11 10 9 9 8 8; do
        curl -s -o /dev/null -X POST "$BASE/api/request" \
            -H 'Content-Type: application/json' \
            -d "{\"floor\":$i,\"direction\":\"down\",\"kind\":\"hall\",\"elevatorId\":0}"
        ((total++))
    done

    # 第 3 波 — 中层混合双向
    for floor_dir in "5:up" "6:down" "7:up" "8:down" "9:up" "10:down" \
                      "11:up" "12:down" "13:up" "14:down" \
                      "15:up" "16:down" "3:down" "4:up" "17:up" "19:down"; do
        f="${floor_dir%%:*}"
        d="${floor_dir##*:}"
        curl -s -o /dev/null -X POST "$BASE/api/request" \
            -H 'Content-Type: application/json' \
            -d "{\"floor\":$f,\"direction\":\"$d\",\"kind\":\"hall\",\"elevatorId\":0}"
        ((total++))
    done

    # 第 4 波 — 同层多请求（压力点）
    for i in 2 3 5 7 9 11 13 15 17 19; do
        for _ in 1 2 3; do
            curl -s -o /dev/null -X POST "$BASE/api/request" \
                -H 'Content-Type: application/json' \
                -d "{\"floor\":$i,\"direction\":\"up\",\"kind\":\"hall\",\"elevatorId\":0}"
            ((total++))
        done
    done

    # 第 5 波 — cabin 请求（每部电梯内乘客按目标楼层）
    for eid in 1 2 3 4 5; do
        for f in 5 8 12 15 19; do
            curl -s -o /dev/null -X POST "$BASE/api/request" \
                -H 'Content-Type: application/json' \
                -d "{\"floor\":$f,\"direction\":\"idle\",\"kind\":\"cabin\",\"elevatorId\":$eid}"
            ((total++))
        done
    done

    echo "$total"
}

# ── 等待所有请求完成 ─────────────────────────────────

wait_all_done() {
    local max_wait=${1:-120}
    local waited=0
    while [ $waited -lt $max_wait ]; do
        local pending
        pending=$(curl -s "$BASE/api/state" | python3 -c "
import json,sys
d = json.load(sys.stdin)
reqs = d.get('Requests', {})
# Requests is null/None when empty in Go JSON output (nil map → null)
if reqs is None:
    print(0)
else:
    print(len(reqs))
" 2>/dev/null || echo "?")
        if [ "$pending" = "0" ]; then
            local tick
            tick=$(curl -s "$BASE/api/state" | python3 -c "import json,sys; print(json.load(sys.stdin)['currentTick'])" 2>/dev/null)
            say "  全部完成，耗时 ${tick} ticks, 等待 ${waited}s"
            return 0
        fi
        if [ "$pending" = "?" ]; then
            say "  状态查询异常，继续等待..."
        else
            echo -n "."
        fi
        sleep 2
        waited=$((waited + 2))
    done
    say "  等待超时 (${max_wait}s)，可能仍有未完成请求"
    return 1
}

# ── 主流程 ────────────────────────────────────────────

echo ""
say "══════════════════════════════════════════════"
say "  五种调度算法性能对比"
say "  每个算法执行相同请求负载，结果写入独立 db"
say "══════════════════════════════════════════════"
echo ""

DB_FILES=()

for sched in "${SCHEDULERS[@]}"; do
    echo ""
    say "━━━ 调度器: $sched ━━━"

    # 切换到目标调度器（会重启系统并创建新 db）
    say "  切换调度器..."
    curl -s -o /dev/null -X POST "$BASE/api/scheduler" \
        -H 'Content-Type: application/json' \
        -d "{\"name\":\"$sched\"}"

    # 等待系统稳定
    sleep 1

    # 获取新 db 文件名
    NEW_DB=$(ls -t data/requests_*.db 2>/dev/null | head -1)
    if [ -n "$NEW_DB" ]; then
        say "  新数据库: $NEW_DB"
        DB_FILES+=("$sched:$NEW_DB")
    fi

    # 提交所有请求
    say "  提交测试请求..."
    REQ_COUNT=$(generate_workload)
    say "  已提交 $REQ_COUNT 个请求"

    # 等待全部完成
    wait_all_done 180

    # 查看最终 tick
    FINAL_TICK=$(curl -s "$BASE/api/state" | python3 -c "import json,sys; print(json.load(sys.stdin)['currentTick'])" 2>/dev/null)
    say "  最终 tick: $FINAL_TICK"
done

# ── 汇总报告 ──────────────────────────────────────────

echo ""
say "══════════════════════════════════════════════"
say "  对比报告"
say "══════════════════════════════════════════════"
echo ""

printf "%-20s %-10s %s\n" "Scheduler" "Requests" "DB File"
printf "%s\n" "--------------------------------------------------------------"

for entry in "${DB_FILES[@]}"; do
    sched="${entry%%:*}"
    db="${entry#*:}"
    count=$(sqlite3 "$db" "SELECT COUNT(*) FROM completed_requests;" 2>/dev/null || echo "?")
    printf "%-20s %-10s %s\n" "$sched" "$count" "$db"
done

echo ""
echo "────────────────────────────────────────────────────────────"
echo "  详细数据查询示例:"
echo "  sqlite3 <db文件> 'SELECT AVG(completed_tick - created_tick) FROM completed_requests;'"
echo "  sqlite3 <db文件> 'SELECT AVG(assigned_tick - created_tick) FROM completed_requests;'"
echo "────────────────────────────────────────────────────────────"
