# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "matplotlib",
#     "numpy",
# ]
# ///
"""单数据库分析 — 统计指标 + 图表，输出到 analysis/ 目录。

用法:
    uv run scripts/analyze.py <db文件路径>

示例:
    uv run scripts/analyze.py data/requests_5e_20f_scan_1778562726.db
"""

import sqlite3
import sys
import os
from pathlib import Path
from collections import defaultdict

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
import numpy as np

from plot_fonts import configure_cjk_fonts

configure_cjk_fonts()

PROJECT_ROOT = Path(__file__).resolve().parent.parent
OUTPUT_DIR = str(PROJECT_ROOT / "analysis")


def load_data(db_path: str) -> list[dict]:
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    rows = conn.execute("SELECT * FROM completed_requests ORDER BY id").fetchall()
    conn.close()
    return [dict(r) for r in rows]


def compute_stats(rows: list[dict]) -> dict:
    wait_times = [r["assigned_tick"] - r["created_tick"] for r in rows]
    service_times = [r["completed_tick"] - r["assigned_tick"] for r in rows]
    total_times = [r["completed_tick"] - r["created_tick"] for r in rows]

    def pct(vals, p):
        return float(np.percentile(vals, p))

    stats = {
        "count": len(rows),
        "wait": {
            "avg": float(np.mean(wait_times)),
            "median": float(np.median(wait_times)),
            "p95": pct(wait_times, 95),
            "max": max(wait_times),
        },
        "service": {
            "avg": float(np.mean(service_times)),
            "median": float(np.median(service_times)),
            "p95": pct(service_times, 95),
            "max": max(service_times),
        },
        "total": {
            "avg": float(np.mean(total_times)),
            "median": float(np.median(total_times)),
            "p95": pct(total_times, 95),
            "max": max(total_times),
        },
    }

    # 按 kind 分组
    for kind in ("hall", "cabin"):
        subset = [r for r in rows if r["kind"] == kind]
        if subset:
            tt = [r["completed_tick"] - r["created_tick"] for r in subset]
            stats[f"{kind}_count"] = len(subset)
            stats[f"{kind}_avg_total"] = float(np.mean(tt))

    return stats, wait_times, service_times, total_times


def print_stats(stats: dict):
    print()
    print("=" * 60)
    print("  请求统计")
    print("=" * 60)
    print(f"  总请求数: {stats['count']}")
    if "hall_count" in stats:
        print(f"    hall 请求: {stats['hall_count']}, 平均总耗时: {stats['hall_avg_total']:.1f} tick")
    if "cabin_count" in stats:
        print(f"    cabin 请求: {stats['cabin_count']}, 平均总耗时: {stats['cabin_avg_total']:.1f} tick")
    print()
    print(f"  {'':>12} {'等待时间':>10} {'服务时间':>10} {'总耗时':>10}")
    print(f"  {'平均':>12} {stats['wait']['avg']:>10.1f} {stats['service']['avg']:>10.1f} {stats['total']['avg']:>10.1f}")
    print(f"  {'中位数':>12} {stats['wait']['median']:>10.1f} {stats['service']['median']:>10.1f} {stats['total']['median']:>10.1f}")
    print(f"  {'P95':>12} {stats['wait']['p95']:>10.1f} {stats['service']['p95']:>10.1f} {stats['total']['p95']:>10.1f}")
    print(f"  {'最大':>12} {stats['wait']['max']:>10.1f} {stats['service']['max']:>10.1f} {stats['total']['max']:>10.1f}")
    print("=" * 60)


def chart_histogram(total_times: list[float], db_name: str, output_path: str):
    fig, axes = plt.subplots(1, 3, figsize=(18, 5))
    fig.suptitle(f"请求耗时分布 — {db_name}", fontsize=13, fontweight="bold")

    for ax, title, data in [
        (axes[0], "等待时间 (分配 - 创建)", [r["assigned_tick"] - r["created_tick"] for r in _rows_cache]),
        (axes[1], "服务时间 (完成 - 分配)", [r["completed_tick"] - r["assigned_tick"] for r in _rows_cache]),
        (axes[2], "总耗时 (完成 - 创建)", total_times),
    ]:
        ax.hist(data, bins=30, color="#4A90D9", edgecolor="white", alpha=0.85)
        ax.axvline(np.mean(data), color="#E74C3C", linestyle="--", linewidth=1.5, label=f"均值={np.mean(data):.1f}")
        ax.set_title(title)
        ax.set_xlabel("tick 数")
        ax.legend(fontsize=8)

    plt.tight_layout()
    fig.savefig(output_path, dpi=150)
    plt.close(fig)
    print(f"  -> {output_path}")


def chart_by_floor(rows: list[dict], db_name: str, output_path: str):
    floors = defaultdict(list)
    for r in rows:
        if r["kind"] == "hall":
            floors[r["floor"]].append(r["completed_tick"] - r["created_tick"])

    if not floors:
        return

    floor_nums = sorted(floors.keys())
    avg_times = [float(np.mean(floors[f])) for f in floor_nums]

    fig, ax = plt.subplots(figsize=(14, 5))
    colors = ["#E74C3C" if t == max(avg_times) else "#4A90D9" for t in avg_times]
    bars = ax.bar(floor_nums, avg_times, color=colors, edgecolor="white")

    for bar, val in zip(bars, avg_times):
        ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height() + 1,
                f"{val:.0f}", ha="center", fontsize=7)

    ax.set_title(f"各楼层平均总耗时 (hall 请求) — {db_name}", fontsize=12, fontweight="bold")
    ax.set_xlabel("楼层")
    ax.set_ylabel("平均总耗时 (tick)")
    ax.set_xticks(floor_nums)
    ax.axhline(np.mean(avg_times), color="#E74C3C", linestyle="--", linewidth=1,
               label=f"全局均值={np.mean(avg_times):.1f}")
    ax.legend(fontsize=9)
    plt.tight_layout()
    fig.savefig(output_path, dpi=150)
    plt.close(fig)
    print(f"  -> {output_path}")


# 全局缓存用于直方图内部引用
_rows_cache = []


def main():
    if len(sys.argv) < 2:
        print(f"用法: python3 {sys.argv[0]} <db文件路径>")
        sys.exit(1)

    db_path = sys.argv[1]
    if not os.path.exists(db_path):
        print(f"文件不存在: {db_path}")
        sys.exit(1)

    db_name = Path(db_path).stem
    os.makedirs(OUTPUT_DIR, exist_ok=True)

    rows = load_data(db_path)
    if not rows:
        print("数据库为空，无数据可分析")
        sys.exit(0)

    global _rows_cache
    _rows_cache = rows

    stats, wait_times, service_times, total_times = compute_stats(rows)
    print_stats(stats)

    chart_histogram(total_times, db_name, f"{OUTPUT_DIR}/{db_name}_hist.png")
    chart_by_floor(rows, db_name, f"{OUTPUT_DIR}/{db_name}_by_floor.png")

    print()
    print("完成。图表已保存到 analysis/ 目录。")


if __name__ == "__main__":
    main()
