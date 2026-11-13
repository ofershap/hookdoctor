#!/usr/bin/env python3
"""Generate hookdoctor hero visual (GitHub dark palette, no-cringe flat design)."""
from PIL import Image, ImageDraw, ImageFont

BG = (13, 17, 23)
FG = (230, 237, 243)
MUTED = (139, 148, 158)
GO = (0, 173, 216)
RED = (248, 81, 73)
GREEN = (63, 185, 80)
BORDER = (48, 54, 61)

SANS = "/usr/share/fonts/truetype/noto/NotoSans-Bold.ttf"
SANS_R = "/usr/share/fonts/truetype/noto/NotoSans-Regular.ttf"
MONO = "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf"


def font(path, size):
    return ImageFont.truetype(path, size)


def hero():
    W, H = 1280, 640
    img = Image.new("RGB", (W, H), BG)
    d = ImageDraw.Draw(img)
    for gx in range(0, W, 64):
        d.line([gx, 0, gx, H], fill=(20, 25, 32))
    for gy in range(0, H, 64):
        d.line([0, gy, W, gy], fill=(20, 25, 32))

    # stethoscope-style diagnostic card on the right
    x, y, w, h = 830, 150, 390, 350
    d.rounded_rectangle([x, y, x + w, y + h], radius=14, outline=BORDER, width=4, fill=(22, 27, 34))
    d.text((x + 26, y + 22), "hookdoctor check", font=font(MONO, 28), fill=FG)
    rows = [
        ("PreToolUse  ok", GREEN),
        ("SessionStart  FAIL", RED),
        ("  cause: not on PATH", MUTED),
        ("  fix: absolute path", GO),
        ("Stop  ok", GREEN),
    ]
    ly = y + 78
    for text, color in rows:
        d.text((x + 26, ly), text, font=font(MONO, 24), fill=color)
        ly += 52

    d.text((90, 220), "hookdoctor", font=font(SANS, 110), fill=FG)
    d.text((94, 370), "Why didn't my Claude Code", font=font(SANS_R, 44), fill=MUTED)
    d.text((94, 430), "hook fire?", font=font(SANS_R, 44), fill=MUTED)
    img.save("assets/hero.png")


hero()
print("wrote assets/hero.png")
