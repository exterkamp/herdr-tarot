package main

import (
	"embed"
	"fmt"
	"image"
	"image/png"
	"math"
	"strings"
)

//go:embed cards/*.png
var cardFiles embed.FS

type rgb struct{ r, g, b uint8 }

// picture is a card's art as plain pixels, row by row.
type picture struct {
	w, h int
	px   []rgb
}

func (p picture) at(x, y int) rgb { return p.px[y*p.w+x] }

func loadArt(file string) (picture, error) {
	f, err := cardFiles.Open("cards/" + file + ".png")
	if err != nil {
		return picture{}, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return picture{}, err
	}
	return fromImage(img), nil
}

func fromImage(img image.Image) picture {
	b := img.Bounds()
	p := picture{w: b.Dx(), h: b.Dy(), px: make([]rgb, b.Dx()*b.Dy())}
	for y := 0; y < p.h; y++ {
		for x := 0; x < p.w; x++ {
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			p.px[y*p.w+x] = rgb{uint8(r >> 8), uint8(g >> 8), uint8(bl >> 8)}
		}
	}
	return p
}

// rotated is the picture turned upside down, as a reversed card lies.
func (p picture) rotated() picture {
	out := picture{w: p.w, h: p.h, px: make([]rgb, len(p.px))}
	for i, c := range p.px {
		out.px[len(p.px)-1-i] = c
	}
	return out
}

// scaled resamples the picture to w×h: nearest pixel for small changes, otherwise by averaging the source area
// under each new pixel.
func (p picture) scaled(w, h int) picture {
	if w == p.w && h == p.h {
		return p
	}
	out := picture{w: w, h: h, px: make([]rgb, w*h)}
	sx, sy := float64(p.w)/float64(w), float64(p.h)/float64(h)
	if sx <= 4.0/3 && sy <= 4.0/3 {
		// A small change: pick the nearest pixel, because averaging would smear the dithering into mud.
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				out.px[y*w+x] = p.at(min(p.w-1, int((float64(x)+.5)*sx)), min(p.h-1, int((float64(y)+.5)*sy)))
			}
		}
		return out
	}
	for y := 0; y < h; y++ {
		y0, y1 := float64(y)*sy, float64(y+1)*sy
		for x := 0; x < w; x++ {
			x0, x1 := float64(x)*sx, float64(x+1)*sx
			var r, g, b, total float64
			for yy := int(y0); yy < int(math.Ceil(y1)) && yy < p.h; yy++ {
				wy := math.Min(y1, float64(yy+1)) - math.Max(y0, float64(yy))
				for xx := int(x0); xx < int(math.Ceil(x1)) && xx < p.w; xx++ {
					wt := wy * (math.Min(x1, float64(xx+1)) - math.Max(x0, float64(xx)))
					c := p.at(xx, yy)
					r, g, b, total = r+wt*float64(c.r), g+wt*float64(c.g), b+wt*float64(c.b), total+wt
				}
			}
			if total > 0 {
				out.px[y*w+x] = rgb{uint8(r/total + .5), uint8(g/total + .5), uint8(b/total + .5)}
			}
		}
	}
	return out
}

const reset = "\x1b[0m"

func fg(c rgb) string { return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", c.r, c.g, c.b) }
func bg(c rgb) string { return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", c.r, c.g, c.b) }

var (
	gold     = rgb{0xd8, 0xb0, 0x5a}
	night    = rgb{0x1c, 0x12, 0x33}
	lattice  = rgb{0x5e, 0x45, 0x8f}
	starGold = rgb{0xf2, 0xd2, 0x7a}
)

// The card is drawn inside a gold frame: cols wide (frame included) and rows tall.

// face draws the art with two pixels to a cell (▀: the top pixel is the ink, the bottom the paper), squeezed to
// squeeze of its width for the flip.
func face(p picture, cols, rows int, squeeze float64) []string {
	innerW, innerH := cols-2, rows-2
	w := squeezed(innerW, squeeze)
	art := p.scaled(innerW, innerH*2).scaled(w, innerH*2)
	lines := make([]string, 0, rows)
	lines = append(lines, frameEdge(w, true))
	for row := 0; row < innerH; row++ {
		var b strings.Builder
		b.WriteString(fg(gold) + bg(night) + "│")
		for x := 0; x < w; x++ {
			b.WriteString(fg(art.at(x, row*2)) + bg(art.at(x, row*2+1)) + "▀")
		}
		b.WriteString(fg(gold) + bg(night) + "│" + reset)
		lines = append(lines, b.String())
	}
	lines = append(lines, frameEdge(w, false))
	return centred(lines, w+2, cols)
}

// back draws the face-down card in plain characters: a lattice with a moon in the middle.
func back(cols, rows int, squeeze float64) []string {
	innerW, innerH := cols-2, rows-2
	w := squeezed(innerW, squeeze)
	// Every row the same width, so the medallion stays round.
	medallion := []string{
		`  .-"-.  `,
		` /  *  \ `,
		`|  * *  |`,
		` \  *  / `,
		`  '-.-'  `,
	}
	if innerH < 9 || w < 13 {
		medallion = []string{"*"}
	}
	top := (innerH - len(medallion)) / 2
	lines := []string{frameEdge(w, true)}
	for row := 0; row < innerH; row++ {
		cells := make([]string, w)
		for x := range cells {
			fx := x + (innerW-w)/2 // the middle of the lattice, as the card turns edge-on
			switch (fx + row) % 4 {
			case 0:
				cells[x] = fg(lattice) + "/"
			case 2:
				cells[x] = fg(lattice) + "\\"
			default:
				if (fx*7+row*3)%23 == 0 {
					cells[x] = fg(starGold) + "·"
				} else {
					cells[x] = " "
				}
			}
		}
		if m := row - top; m >= 0 && m < len(medallion) {
			text := medallion[m]
			start := (w - len(text)) / 2
			for i, ch := range text {
				if start+i >= 0 && start+i < w {
					cells[start+i] = fg(starGold) + string(ch)
				}
			}
			// A little clearing around the moon.
			for _, x := range []int{start - 1, start + len(text)} {
				if x >= 0 && x < w {
					cells[x] = " "
				}
			}
		}
		lines = append(lines, fg(gold)+bg(night)+"│"+strings.Join(cells, "")+fg(gold)+"│"+reset)
	}
	lines = append(lines, frameEdge(w, false))
	return centred(lines, w+2, cols)
}

func frameEdge(w int, top bool) string {
	l, r := "╭", "╮"
	if !top {
		l, r = "╰", "╯"
	}
	return fg(gold) + bg(night) + l + strings.Repeat("─", w) + r + reset
}

func squeezed(w int, squeeze float64) int {
	return max(0, min(w, int(float64(w)*squeeze+.5)))
}

// centred pads each line of width w so it sits in the middle of cols.
func centred(lines []string, w, cols int) []string {
	left := (cols - w) / 2
	right := cols - w - left
	for i, l := range lines {
		lines[i] = strings.Repeat(" ", left) + l + strings.Repeat(" ", right)
	}
	return lines
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
