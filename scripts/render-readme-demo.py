#!/usr/bin/env python3
"""Regenerates docs/images/demo.gif, the animated terminal demo used in README.md.

The GIF shows three steps of examples/feriados-brasil:
  1. the curl request to BrasilAPI and the JSON that comes back;
  2. the report script (relatorio.rp);
  3. the same request piped into rptgo, and the report it prints.

The commands really run: curl hits BrasilAPI and rptgo is built from this checkout, so the
frames show real output. If the request fails (offline), the saved copy
examples/feriados-brasil/dados-2026.json stands in for the response and a warning is printed.

Needs: Go, curl and Python 3 with Pillow.
"""
import json
import os
import re
import subprocess
import sys
import tempfile
import textwrap
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parent.parent
EXAMPLE = ROOT / "examples" / "feriados-brasil"
OUT = ROOT / "docs" / "images" / "demo.gif"
URL = "https://brasilapi.com.br/api/feriados/v1/2026"

COLS, ROWS = 100, 46
FONT_PATHS = [
    "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
    "/usr/share/fonts/dejavu/DejaVuSansMono.ttf",
    "/Library/Fonts/Menlo.ttc",
]
FONT_SIZE = 13
PAD = 18
TITLE_H = 34

BG = (24, 26, 33)
BAR = (38, 41, 52)
FG = (216, 222, 233)
DIM = (110, 118, 135)
PROMPT = (126, 217, 140)
KEYWORD = (110, 168, 254)
STRING = (240, 198, 116)
CURSOR = (216, 222, 233)

KEYWORDS = set(
    "input width length sort before after detail header footer report end print skip "
    "if then else in col using where set to and or not centered".split()
)
FUNCTIONS = set("count total avg min max index mdy dow".split())


def load_font():
    for p in FONT_PATHS:
        if os.path.exists(p):
            return ImageFont.truetype(p, FONT_SIZE)
    sys.exit("no monospace font found; edit FONT_PATHS in " + __file__)


FONT = load_font()
CELL_W = FONT.getlength("M")
ASC, DESC = FONT.getmetrics()
CELL_H = ASC + DESC + 3
WIDTH = int(PAD * 2 + CELL_W * COLS)
HEIGHT = int(TITLE_H + PAD * 2 + CELL_H * ROWS)


def run(cmd, **kw):
    return subprocess.run(cmd, check=True, capture_output=True, **kw)


def fetch_json():
    try:
        return run(["curl", "-sf", "-m", "20", URL]).stdout
    except (subprocess.CalledProcessError, FileNotFoundError):
        print("warning: request failed, using dados-2026.json", file=sys.stderr)
        return (EXAMPLE / "dados-2026.json").read_bytes()


def build_rptgo(tmp):
    exe = Path(tmp) / "rptgo"
    run(["go", "build", "-o", str(exe), "./cmd/rptgo"], cwd=ROOT)
    return exe


# A row is a list of (text, color) segments.
def plain(text, color=FG):
    return [(text, color)]


def wrap(text, color=FG):
    """Hard-wraps one logical line at COLS, like a terminal does."""
    if text == "":
        return [[]]
    return [plain(text[i : i + COLS], color) for i in range(0, len(text), COLS)]


def wrap_segments(segs):
    """Hard-wraps a row made of colored segments at COLS characters."""
    rows, cur, used = [], [], 0
    for text, color in segs:
        while text:
            room = COLS - used
            cur.append((text[:room], color))
            used += len(text[:room])
            text = text[room:]
            if used == COLS:
                rows.append(cur)
                cur, used = [], 0
    if cur or not rows:
        rows.append(cur)
    return rows


def highlight_rp(lines):
    rows, in_comment = [], False
    for line in lines:
        segs = []
        pos = 0
        while pos < len(line):
            if in_comment:
                end = line.find("*/", pos)
                stop = len(line) if end < 0 else end + 2
                segs.append((line[pos:stop], DIM))
                pos = stop
                in_comment = end < 0
                continue
            if line.startswith("/*", pos):
                in_comment = True
                continue
            m = re.match(r"'[^']*'?", line[pos:])
            if m and line[pos] == "'":
                segs.append((m.group(0), STRING))
                pos += len(m.group(0))
                continue
            m = re.match(r"[A-Za-z_]+", line[pos:])
            if m:
                w = m.group(0)
                col = KEYWORD if w in KEYWORDS or w in FUNCTIONS else FG
                segs.append((w, col))
                pos += len(w)
                continue
            segs.append((line[pos], FG))
            pos += 1
        rows.extend(wrap_segments(segs))
    return rows


class Terminal:
    def __init__(self):
        self.rows = []
        self.frames = []  # (RGB image, duration ms)

    def clear(self):
        self.rows = []

    def visible(self):
        return self.rows[-ROWS:]

    def snapshot(self, ms, cursor=True):
        img = Image.new("RGB", (WIDTH, HEIGHT), BG)
        d = ImageDraw.Draw(img)
        d.rectangle([0, 0, WIDTH, TITLE_H], fill=BAR)
        for i, c in enumerate([(255, 95, 86), (255, 189, 46), (39, 201, 63)]):
            cx = PAD + 8 + i * 22
            d.ellipse([cx - 6, TITLE_H // 2 - 6, cx + 6, TITLE_H // 2 + 6], fill=c)
        d.text((WIDTH / 2, TITLE_H / 2), "rptgo", font=FONT, fill=DIM, anchor="mm")
        vis = self.visible()
        y0 = TITLE_H + PAD
        for r, segs in enumerate(vis):
            x = PAD
            y = y0 + r * CELL_H
            for text, color in segs:
                d.text((x, y), text, font=FONT, fill=color)
                x += CELL_W * len(text)
        if cursor and vis:
            x = PAD + CELL_W * sum(len(t) for t, _ in vis[-1])
            y = y0 + (len(vis) - 1) * CELL_H
            d.rectangle([x, y + 1, x + CELL_W - 1, y + CELL_H - 2], fill=CURSOR)
        self.frames.append((img, ms))

    def type_command(self, cmd, per_char=45):
        self.rows.append([("$ ", PROMPT)])
        self.snapshot(500)
        for ch in cmd:
            self.rows[-1] = self.rows[-1] + [(ch, FG)]
            self.snapshot(per_char)
        self.snapshot(450)
        self.rows.append([])  # Enter: the cursor drops to a new line

    def show(self, rows, hold):
        self.rows.extend(rows)
        self.snapshot(hold)

    def stream(self, rows, per_frame=3, ms=60, pause_when_full=2400):
        paused = False
        for i in range(0, len(rows), per_frame):
            self.rows.extend(rows[i : i + per_frame])
            self.snapshot(ms)
            if not paused and len(self.rows) >= ROWS - 1:
                self.snapshot(pause_when_full)
                paused = True


def save_gif(frames, path):
    # One shared palette keeps every frame small; identical neighbours are merged.
    # The palette comes from the script frame, which has every text color in use.
    sample = frames[len(frames) // 2][0]
    pal_img = sample.quantize(colors=48, method=Image.Quantize.MEDIANCUT)
    merged = []
    for img, ms in frames:
        q = img.quantize(palette=pal_img, dither=Image.Dither.NONE)
        if merged and merged[-1][0].tobytes() == q.tobytes():
            merged[-1][1] += ms
        else:
            merged.append([q, ms])
    merged[-1][1] += 4000  # rest on the finished report before looping
    path.parent.mkdir(parents=True, exist_ok=True)
    merged[0][0].save(
        path,
        save_all=True,
        append_images=[m[0] for m in merged[1:]],
        duration=[m[1] for m in merged],
        loop=0,
        disposal=1,
        optimize=True,
    )
    print(f"wrote {path.relative_to(ROOT)}: {len(merged)} frames, {path.stat().st_size // 1024} KiB")


def main():
    with tempfile.TemporaryDirectory() as tmp:
        exe = build_rptgo(tmp)
        body = fetch_json()
        script = (EXAMPLE / "relatorio.rp").read_text(encoding="utf-8").splitlines()
        report = run(
            [str(exe), "--input", "json", "relatorio.rp", "-"], cwd=EXAMPLE, input=body
        ).stdout.decode("utf-8").rstrip().splitlines()  # drop the blank padding to `length`

    json_rows = []
    for chunk in textwrap.wrap(body.decode("utf-8"), COLS, break_long_words=True, drop_whitespace=False):
        json_rows.append(plain(chunk, DIM))

    t = Terminal()
    # 1. the request and the JSON that comes back
    t.type_command(f"curl -s {URL}")
    t.snapshot(700)
    t.show(json_rows, 2600)
    # 2. the script
    t.clear()
    t.type_command("cat relatorio.rp")
    t.show(highlight_rp(script), 3600)
    # 3. the request piped into rptgo
    t.clear()
    t.type_command(f"curl -s {URL} | rptgo --input json relatorio.rp -")
    t.snapshot(700)
    rows, blanks = [], 0
    for line in report:
        # The text report pads each page with blank lines up to `length`; the demo shows
        # that gap as at most two blank lines so the screen is not wasted on page-break padding.
        blanks = blanks + 1 if not line.strip() else 0
        if blanks > 2:
            continue
        rows.extend(wrap(line))
    t.stream(rows)
    t.rows.append([("$ ", PROMPT)])
    t.snapshot(400)

    save_gif(t.frames, OUT)


if __name__ == "__main__":
    main()
