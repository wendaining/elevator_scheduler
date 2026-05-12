# /// script
# requires-python = ">=3.12"
# dependencies = [
#     "matplotlib",
#     "numpy",
# ]
# ///
"""多调度器性能对比 — 读取多个 db 文件，生成对比图表和 analysis.md 报告。

用法:
    # 自动扫描 data/ 下所有带调度器名称的 db 文件
    uv run scripts/compare_charts.py

    # 指定文件
    uv run scripts/compare_charts.py data/requests_*_fcfs_*.db data/requests_*_scan_*.db
"""

import sqlite3
import sys
import os
import glob
import re
from pathlib import Path

import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
import numpy as np

from plot_fonts import configure_cjk_fonts

configure_cjk_fonts()

PROJECT_ROOT = Path(__file__).resolve().parent.parent
OUTPUT_DIR = str(PROJECT_ROOT / "analysis")
ANALYSIS_MD = f"{OUTPUT_DIR}/analysis.md"
SCHEDULER_COLORS = {
    "fcfs": "#3498DB",
    "scan": "#2ECC71",
    "look": "#E67E22",
    "first-available": "#9B59B6",
    "nearest-idle": "#E74C3C",
}

RE_DB_FILENAME = re.compile(r"requests_.*_(fcfs|scan|look|first-available|nearest-idle)_.*\.db$")


def find_db_files() -> dict[str, str]:
    """扫描 data/，返回 {scheduler_name: path} 映射（每个调度器取最新的 db 文件）。"""
    result = {}
    data_dir = str(PROJECT_ROOT / "data")
    for path in glob.glob(f"{data_dir}/requests_*.db"):
        m = RE_DB_FILENAME.match(os.path.basename(path))
        if m:
            name = m.group(1)
            if name not in result or os.path.getmtime(path) > os.path.getmtime(result[name]):
                result[name] = path
    return result


def load_stats(db_path: str) -> dict:
    conn = sqlite3.connect(db_path)
    conn.row_factory = sqlite3.Row
    rows = conn.execute("SELECT * FROM completed_requests").fetchall()
    conn.close()

    if not rows:
        return {"count": 0}

    wait = [r["assigned_tick"] - r["created_tick"] for r in rows]
    service = [r["completed_tick"] - r["assigned_tick"] for r in rows]
    total = [r["completed_tick"] - r["created_tick"] for r in rows]

    def pct(vals, p):
        return float(np.percentile(vals, p))

    return {
        "count": len(rows),
        "wait_avg": float(np.mean(wait)),
        "wait_p95": pct(wait, 95),
        "service_avg": float(np.mean(service)),
        "service_p95": pct(service, 95),
        "total_avg": float(np.mean(total)),
        "total_median": float(np.median(total)),
        "total_p95": pct(total, 95),
        "total_max": max(total),
    }


def chart_comparison(data: dict[str, dict], output_path: str):
    """分组柱状图：各调度器的平均等待时间和总耗时对比。"""
    schedulers = list(data.keys())
    if not schedulers:
        print("没有数据可对比")
        return

    x = np.arange(len(schedulers))
    width = 0.35

    wait_avgs = [data[s]["wait_avg"] for s in schedulers]
    total_avgs = [data[s]["total_avg"] for s in schedulers]
    colors = [SCHEDULER_COLORS.get(s, "#95A5A6") for s in schedulers]

    fig, axes = plt.subplots(1, 2, figsize=(16, 6))

    # 左图 — 平均等待 + 平均总耗时对比
    ax = axes[0]
    bars1 = ax.bar(x - width / 2, wait_avgs, width, label="平均等待时间",
                   color="#5DADE2", edgecolor="white")
    bars2 = ax.bar(x + width / 2, total_avgs, width, label="平均总耗时",
                   color="#E74C3C", edgecolor="white")

    for bar, val in zip(bars1, wait_avgs):
        ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height() + 0.5,
                f"{val:.1f}", ha="center", fontsize=9)
    for bar, val in zip(bars2, total_avgs):
        ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height() + 0.5,
                f"{val:.1f}", ha="center", fontsize=9)

    ax.set_title("各调度器平均等待时间 vs 平均总耗时", fontsize=13, fontweight="bold")
    ax.set_xticks(x)
    ax.set_xticklabels(schedulers, fontsize=10)
    ax.set_ylabel("tick 数")
    ax.legend(fontsize=10)
    ax.set_ylim(0, max(total_avgs) * 1.2)

    # 右图 — P95 对比
    ax = axes[1]
    bars3 = ax.bar(x - width / 2, [data[s]["wait_p95"] for s in schedulers], width,
                   label="等待时间 P95", color="#5DADE2", edgecolor="white")
    bars4 = ax.bar(x + width / 2, [data[s]["total_p95"] for s in schedulers], width,
                   label="总耗时 P95", color="#E74C3C", edgecolor="white")

    for bar, val in zip(bars3, [data[s]["wait_p95"] for s in schedulers]):
        ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height() + 1,
                f"{val:.1f}", ha="center", fontsize=9)
    for bar, val in zip(bars4, [data[s]["total_p95"] for s in schedulers]):
        ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height() + 1,
                f"{val:.1f}", ha="center", fontsize=9)

    ax.set_title("各调度器 P95 等待时间 vs P95 总耗时", fontsize=13, fontweight="bold")
    ax.set_xticks(x)
    ax.set_xticklabels(schedulers, fontsize=10)
    ax.set_ylabel("tick 数")
    ax.legend(fontsize=10)
    ax.set_ylim(0, max(data[s]["total_p95"] for s in schedulers) * 1.2)

    plt.tight_layout()
    fig.savefig(output_path, dpi=150)
    plt.close(fig)
    print(f"  -> {output_path}")


def chart_request_count(data: dict[str, dict], output_path: str):
    """各调度器完成的请求数对比。"""
    schedulers = list(data.keys())
    x = np.arange(len(schedulers))
    counts = [data[s]["count"] for s in schedulers]
    colors = [SCHEDULER_COLORS.get(s, "#95A5A6") for s in schedulers]

    fig, ax = plt.subplots(figsize=(10, 5))
    bars = ax.bar(x, counts, color=colors, edgecolor="white", width=0.6)

    for bar, val in zip(bars, counts):
        ax.text(bar.get_x() + bar.get_width() / 2, bar.get_height() + 1,
                str(val), ha="center", fontsize=12, fontweight="bold")

    ax.set_title("各调度器完成请求数", fontsize=13, fontweight="bold")
    ax.set_xticks(x)
    ax.set_xticklabels(schedulers, fontsize=11)
    ax.set_ylabel("完成请求数")

    plt.tight_layout()
    fig.savefig(output_path, dpi=150)
    plt.close(fig)
    print(f"  -> {output_path}")


def write_analysis_md(data: dict[str, dict], chart_paths: list[str]):
    """生成 analysis.md 汇总报告。"""
    if not data:
        return

    lines = [
        "# 电梯调度算法性能对比",
        "",
        "## 汇总表",
        "",
        "| 调度器 | 请求数 | 平均等待 | 等待 P95 | 平均总耗时 | 总耗时 P95 | 最大总耗时 |",
        "|--------|--------|----------|----------|------------|------------|------------|",
    ]

    for name in data:
        d = data[name]
        lines.append(
            f"| {name} | {d['count']} | {d['wait_avg']:.1f} | {d['wait_p95']:.1f} "
            f"| {d['total_avg']:.1f} | {d['total_p95']:.1f} | {d['total_max']:.0f} |"
        )

    # 最佳和最差
    if len(data) >= 2:
        best_wait = min(data, key=lambda s: data[s]["wait_avg"])
        best_total = min(data, key=lambda s: data[s]["total_avg"])
        worst_wait = max(data, key=lambda s: data[s]["wait_avg"])
        worst_total = max(data, key=lambda s: data[s]["total_avg"])

        lines += [
            "",
            "## 关键发现",
            "",
            f"- **最短平均等待**: {best_wait} ({data[best_wait]['wait_avg']:.1f} tick)",
            f"- **最长平均等待**: {worst_wait} ({data[worst_wait]['wait_avg']:.1f} tick)",
            f"- **最短平均总耗时**: {best_total} ({data[best_total]['total_avg']:.1f} tick)",
            f"- **最长平均总耗时**: {worst_total} ({data[worst_total]['total_avg']:.1f} tick)",
        ]

    lines += [
        "",
        "## 图表",
        "",
    ]
    for p in chart_paths:
        fname = os.path.basename(p)
        lines.append(f"![{fname}]({fname})")
        lines.append("")

    with open(ANALYSIS_MD, "w", encoding="utf-8") as f:
        f.write("\n".join(lines))
    print(f"  -> {ANALYSIS_MD}")


def main():
    os.makedirs(OUTPUT_DIR, exist_ok=True)

    # 确定要分析的 db 文件
    args = sys.argv[1:]
    if args:
        db_map = {}
        for path in args:
            m = RE_DB_FILENAME.match(os.path.basename(path))
            if m:
                db_map[m.group(1)] = path
    else:
        db_map = find_db_files()

    if not db_map:
        print("未找到包含调度器名称的 db 文件。")
        print("请先运行 scripts/compare.sh 生成测试数据。")
        sys.exit(1)

    print(f"找到 {len(db_map)} 个调度器的数据:")
    for name, path in db_map.items():
        print(f"  {name}: {path}")

    # 加载统计
    data = {}
    for name, path in sorted(db_map.items()):
        print(f"\n分析 {name} ({path})...")
        stats = load_stats(path)
        if stats["count"] > 0:
            data[name] = stats
            print(f"  {stats['count']} 个请求, 平均总耗时: {stats['total_avg']:.1f} ticks")
        else:
            print(f"  无数据，跳过")

    if not data:
        print("所有数据库均为空")
        sys.exit(1)

    # 生成图表
    charts = []
    p1 = f"{OUTPUT_DIR}/compare_bar.png"
    p2 = f"{OUTPUT_DIR}/compare_count.png"
    chart_comparison(data, p1)
    charts.append(p1)
    chart_request_count(data, p2)
    charts.append(p2)

    # 生成 Markdown 报告
    write_analysis_md(data, charts)

    print(f"\n完成。报告和图表已保存到 {OUTPUT_DIR}/。")


if __name__ == "__main__":
    main()
