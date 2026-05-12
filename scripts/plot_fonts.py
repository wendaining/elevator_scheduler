"""Matplotlib 字体配置工具。

脚本生成中文图表时需要 CJK 字体，但不能依赖某台机器上的固定字体路径。
这里按常见字体目录扫描可用字体，再用字体家族名配置 matplotlib。
"""

from pathlib import Path

import matplotlib.font_manager as fm
import matplotlib.pyplot as plt


CJK_FONT_FAMILIES = [
    "Noto Sans CJK SC",
    "Noto Serif CJK SC",
    "Source Han Sans SC",
    "Source Han Serif SC",
    "Microsoft YaHei",
    "SimHei",
    "SimSun",
    "WenQuanYi Micro Hei",
    "Droid Sans Fallback",
    "PingFang SC",
    "Hiragino Sans GB",
]

CJK_FONT_NAME_TOKENS = [
    "noto",
    "sourcehan",
    "source-han",
    "simsun",
    "simhei",
    "msyh",
    "microsoftyahei",
    "pingfang",
    "hiragino",
    "droidsansfallback",
    "droid",
    "wenquanyi",
    "wqy",
]

FONT_DIRS = [
    Path.home() / ".local/share/fonts",
    Path("/usr/share/fonts"),
    Path("/usr/local/share/fonts"),
    Path("/System/Library/Fonts"),
    Path("/Library/Fonts"),
    Path("C:/Windows/Fonts"),
]


def configure_cjk_fonts() -> None:
    """配置 matplotlib 中文字体和负号显示。

    优先使用系统中已安装的常见中文字体。找不到时保留 DejaVu Sans，
    matplotlib 会继续使用默认字体，只是中文可能显示为方框。
    """

    register_cjk_font_files()
    installed_families = {font.name for font in fm.fontManager.ttflist}
    available_families = [
        family for family in CJK_FONT_FAMILIES
        if family in installed_families
    ]

    plt.rcParams["font.sans-serif"] = available_families + ["DejaVu Sans"]
    plt.rcParams["axes.unicode_minus"] = False


def register_cjk_font_files() -> None:
    """从常见字体目录注册疑似 CJK 字体文件。"""

    for font_dir in FONT_DIRS:
        if not font_dir.exists():
            continue

        for font_path in iter_font_files(font_dir):
            filename = normalized_name(font_path.name)
            if not any(token in filename for token in CJK_FONT_NAME_TOKENS):
                continue
            try:
                fm.fontManager.addfont(font_path)
            except RuntimeError:
                continue


def iter_font_files(font_dir: Path):
    for suffix in ("*.ttf", "*.ttc", "*.otf"):
        yield from font_dir.rglob(suffix)


def normalized_name(name: str) -> str:
    return name.lower().replace(" ", "").replace("_", "").replace("-", "")
