"""Turn the 1909 public-domain scans into the deck's 40x60 pixel art (cards/*.png), with one shared 32-color palette."""
import os, sys
from PIL import Image, ImageEnhance, ImageFilter
SCANS, OUT = sys.argv[1], sys.argv[2]
SIZE = (40, 60)

def prep(im):
    w, h = im.size
    # The printed numeral strip and title box are unreadable at this size; the plugin prints them as text.
    im = im.crop((int(w * 0.03), int(h * 0.065), w - int(w * 0.03), h - int(h * 0.115)))
    im = ImageEnhance.Color(im).enhance(1.45)
    return ImageEnhance.Contrast(im).enhance(1.35)

def pixel(im, pal):
    big = im.resize((SIZE[0] * 4, SIZE[1] * 4), Image.LANCZOS).filter(ImageFilter.MedianFilter(5))
    small = big.resize(SIZE, Image.BOX).filter(ImageFilter.UnsharpMask(radius=1, percent=60, threshold=2))
    return small.quantize(palette=pal, dither=Image.NONE)

files = sorted(f for f in os.listdir(SCANS) if f.endswith('.jpeg'))
sample = Image.new('RGB', (len(files) * SIZE[0], SIZE[1]))
for i, f in enumerate(files):
    sample.paste(prep(Image.open(os.path.join(SCANS, f)).convert('RGB')).resize(SIZE, Image.BOX), (i * SIZE[0], 0))
pal = sample.quantize(colors=32, method=Image.MEDIANCUT)
os.makedirs(OUT, exist_ok=True)
for f in files:
    card = pixel(prep(Image.open(os.path.join(SCANS, f)).convert('RGB')), pal)
    name = f[:-5].lower().replace(' ', '-')
    card.save(os.path.join(OUT, name + '.png'), optimize=True)
print(len(files), 'cards')
