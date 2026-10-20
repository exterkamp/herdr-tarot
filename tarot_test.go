package main

import (
	"math/rand"
	"regexp"
	"testing"
	"time"
	"unicode/utf8"
)

func TestEveryCardHasArtAndMeanings(t *testing.T) {
	if len(deck) != 78 {
		t.Fatalf("%d cards, want 78", len(deck))
	}
	seen := map[string]bool{}
	for _, c := range deck {
		if seen[c.File] {
			t.Errorf("%s twice", c.File)
		}
		seen[c.File] = true
		art, err := loadArt(c.File)
		if err != nil || art.w != 40 || art.h != 60 {
			t.Errorf("%s: art %dx%d, %v", c.File, art.w, art.h, err)
		}
		for _, m := range []Meaning{c.Upright, c.Reversed} {
			if c.Name == "" || m.Keywords == "" || m.Text == "" {
				t.Errorf("%s is missing words", c.File)
			}
		}
	}
}

func TestTheCardIsKeptForTheDay(t *testing.T) {
	dir := t.TempDir()
	morning := time.Date(2026, 10, 19, 8, 0, 0, 0, time.Local)
	if _, ok := today(dir, morning); ok {
		t.Fatal("a card before any was drawn")
	}
	d := pick(morning, rand.New(rand.NewSource(1)))
	keep(dir, d)
	if got, ok := today(dir, morning.Add(15*time.Hour)); !ok || got != d {
		t.Errorf("that evening: %+v %v, want %+v", got, ok, d)
	}
	if _, ok := today(dir, morning.Add(17*time.Hour)); ok {
		t.Error("the card carried over past midnight")
	}
}

var sgr = regexp.MustCompile("\x1b\\[[0-9;]*m")

// Every frame fits the pane, from a phone to a wide monitor, face down, mid-flip and face up.
func TestFramesFitThePane(t *testing.T) {
	for _, size := range [][2]int{{53, 40}, {53, 24}, {40, 30}, {80, 24}, {100, 34}, {200, 60}} {
		for frame := 0; frame <= len(flip); frame++ {
			for _, rev := range []bool{false, true} {
				m := model{now: time.Now(), draw: Draw{Card: "21-world", Reversed: rev}, width: size[0], height: size[1]}
				if err := m.load(); err != nil {
					t.Fatal(err)
				}
				m.frame = frame
				m.revealed = frame == len(flip)
				if frame == len(flip) {
					m.frame = 0
				}
				lines := regexp.MustCompile("\n").Split(m.View(), -1)
				if len(lines) != size[1] && size[1] >= 30 {
					t.Errorf("%v frame %d: %d lines", size, frame, len(lines))
				}
				for _, l := range lines {
					if n := utf8.RuneCountInString(sgr.ReplaceAllString(l, "")); n > size[0] {
						t.Errorf("%v frame %d: a line is %d wide", size, frame, n)
						break
					}
				}
			}
		}
	}
}
