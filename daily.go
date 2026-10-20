package main

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

// Draw is the day's card: drawn once, kept until midnight.
type Draw struct {
	Date     string `json:"date"`
	Card     string `json:"card"`
	Reversed bool   `json:"reversed"`
}

func day(t time.Time) string { return t.Format("2006-01-02") }

func stateDir() string {
	if d := os.Getenv("HERDR_PLUGIN_STATE_DIR"); d != "" {
		return d
	}
	if d := os.Getenv("XDG_STATE_HOME"); d != "" {
		return filepath.Join(d, "herdr-tarot")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "herdr-tarot")
}

// today is the card already drawn today, if there is one.
func today(dir string, now time.Time) (Draw, bool) {
	data, err := os.ReadFile(filepath.Join(dir, "today.json"))
	if err != nil {
		return Draw{}, false
	}
	var d Draw
	if json.Unmarshal(data, &d) != nil || d.Date != day(now) || cardByFile(d.Card) < 0 {
		return Draw{}, false
	}
	return d, true
}

// pick chooses a card for the day. About a third come up reversed.
func pick(now time.Time, rng *rand.Rand) Draw {
	return Draw{Date: day(now), Card: deck[rng.Intn(len(deck))].File, Reversed: rng.Intn(3) == 0}
}

// redraw picks a new card for the day, never the one it replaces.
func redraw(old Draw, now time.Time, rng *rand.Rand) Draw {
	for {
		if d := pick(now, rng); d.Card != old.Card {
			return d
		}
	}
}

// keep saves the day's card, so opening the oracle again shows the same one.
func keep(dir string, d Draw) {
	if data, err := json.Marshal(d); err == nil {
		_ = os.MkdirAll(dir, 0o755)
		tmp := filepath.Join(dir, "today.json.tmp")
		if os.WriteFile(tmp, data, 0o644) == nil {
			_ = os.Rename(tmp, filepath.Join(dir, "today.json"))
		}
	}
}

func cardByFile(file string) int {
	for i, c := range deck {
		if c.File == file {
			return i
		}
	}
	return -1
}
