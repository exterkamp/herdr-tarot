"""Render a frame of tarot's output (truecolor SGR + text) to a PNG, to look at the layout without a terminal."""
import re, sys
from PIL import Image, ImageDraw, ImageFont
CW, CH = 9, 18
try:
    font = ImageFont.truetype('DejaVuSansMono.ttf', 14)
except OSError:
    font = ImageFont.truetype('/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf', 14)
lines = sys.stdin.read().split('\n')
cols = int(sys.argv[2])  # usage: ./tarot -print -w 53 | python3 tools/ansi2png.py out.png 53
img = Image.new('RGB', (cols * CW, len(lines) * CH), (12, 12, 16))
d = ImageDraw.Draw(img)
DEF_FG, DEF_BG = (220, 220, 220), (12, 12, 16)
for y, line in enumerate(lines):
    fg, bg, dim, x = DEF_FG, DEF_BG, False, 0
    for tok in re.findall(r'\x1b\[[0-9;]*m|.', line):
        if tok.startswith('\x1b['):
            p = [int(v) for v in tok[2:-1].split(';') if v] or [0]
            i = 0
            while i < len(p):
                if p[i] == 0: fg, bg, dim = DEF_FG, DEF_BG, False
                elif p[i] == 2: dim = True
                elif p[i] == 38: fg = tuple(p[i+2:i+5]); i += 4
                elif p[i] == 48: bg = tuple(p[i+2:i+5]); i += 4
                i += 1
            continue
        X, Y = x * CW, y * CH
        d.rectangle([X, Y, X + CW - 1, Y + CH - 1], fill=bg)
        f = tuple(v // 2 for v in fg) if dim else fg
        if tok == '▀':
            d.rectangle([X, Y, X + CW - 1, Y + CH // 2 - 1], fill=f)
        elif tok != ' ':
            d.text((X, Y + 1), tok, font=font, fill=f)
        x += 1
img.save(sys.argv[1])
