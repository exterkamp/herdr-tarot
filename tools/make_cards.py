"""Turn the 1909 public-domain scans into the deck's 40x60 pixel art (cards/*.png): one shared 64-colour palette,
with Floyd-Steinberg dithering to carry the detail and shading that 40x60 pixels and 64 colours can't hold outright."""
import os, sys
from PIL import Image, ImageEnhance
SCANS, OUT = sys.argv[1], sys.argv[2]
SIZE = (40, 60)
COLOURS = 64
def prep(im):
    # Crop the printed title and numeral (the program shows the name as text), then brighten the faded inks.
    w, h = im.size
    im = im.crop((int(w * 0.03), int(h * 0.065), w - int(w * 0.03), h - int(h * 0.115)))
    im = ImageEnhance.Color(im).enhance(1.45)
    return ImageEnhance.Contrast(im).enhance(1.35)
files = sorted(f for f in os.listdir(SCANS) if f.endswith('.jpeg'))
cards = {f: prep(Image.open(os.path.join(SCANS, f)).convert('RGB')).resize(SIZE, Image.LANCZOS) for f in files}
# One palette for the whole deck, so the cards look like they belong together.
sample = Image.new('RGB', (len(files) * SIZE[0], SIZE[1]))
for i, f in enumerate(files):
    sample.paste(cards[f], (i * SIZE[0], 0))
pal = sample.quantize(colors=COLOURS, method=Image.MEDIANCUT)
os.makedirs(OUT, exist_ok=True)
for f in files:
    name = f[:-5].lower().replace(' ', '-')
    cards[f].quantize(palette=pal, dither=Image.FLOYDSTEINBERG).save(os.path.join(OUT, name + '.png'), optimize=True)
print(len(files), 'cards')
