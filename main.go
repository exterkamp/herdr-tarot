// Command tarot is a little oracle for herdr: one face-down card a day. Turn it over to see what it says; it
// stays yours until midnight. It runs as a herdr plugin pane, or on its own in a terminal.
package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/aymanbagabas/go-osc52/v2"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	card := flag.String("card", "", "show this card (e.g. 19-sun) instead of today's; nothing is kept")
	reversed := flag.Bool("reversed", false, "with -card, show it reversed")
	printFrame := flag.Bool("print", false, "print one frame to stdout instead of running (for checking the layout)")
	width := flag.Int("w", 53, "with -print, the width")
	height := flag.Int("h", 40, "with -print, the height")
	frame := flag.Int("frame", -1, "with -print, show this frame of the flip (0 is face down)")
	flag.Parse()

	now := time.Now()
	m := model{dir: stateDir(), now: now}
	switch {
	case *card != "":
		if cardByFile(*card) < 0 {
			fmt.Fprintln(os.Stderr, "tarot: no card named", *card)
			os.Exit(2)
		}
		m.draw, m.preview = Draw{Date: day(now), Card: *card, Reversed: *reversed}, true
	default:
		if d, ok := today(m.dir, now); ok {
			m.draw, m.revealed, m.again = d, true, true
		} else {
			// Chosen now so the layout doesn't jump, but only kept once it's turned over.
			m.draw = pick(now, rand.New(rand.NewSource(now.UnixNano())))
		}
	}
	if err := m.load(); err != nil {
		fmt.Fprintln(os.Stderr, "tarot:", err)
		os.Exit(1)
	}
	if *printFrame {
		m.width, m.height = *width, *height
		m.revealed = *frame < 0 || *frame > len(flip)
		if *frame > 0 && *frame <= len(flip) {
			m.frame = *frame
		}
		fmt.Println(m.View())
		return
	}
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "tarot:", err)
		os.Exit(1)
	}
}

type model struct {
	dir     string
	now     time.Time
	draw    Draw
	card    Card
	art     picture
	preview bool // shown with -card: nothing is kept

	revealed bool
	again    bool // already drawn earlier today
	frame    int  // of the flip, 1…len(flip); 0 when still
	unturn   bool // the flip is running backwards: the card going face down for a new draw
	scroll   int
	note     string // shown in place of the hint for a moment, after copying

	width, height int
}

type (
	tick     struct{}
	noteGone struct{}
)

// The flip: the back narrows to an edge, then the face widens out of it.
var flip = []struct {
	face    bool
	squeeze float64
}{{false, .8}, {false, .55}, {false, .3}, {false, .08}, {true, .08}, {true, .3}, {true, .55}, {true, .8}, {true, 1}}

func (m *model) load() error {
	m.card = deck[cardByFile(m.draw.Card)]
	art, err := loadArt(m.draw.Card)
	if err != nil {
		return err
	}
	if m.draw.Reversed {
		art = art.rotated()
	}
	m.art = art
	return nil
}

func (m model) Init() tea.Cmd { return nil }

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case noteGone:
		m.note = ""
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tick:
		if m.frame == 0 {
			return m, nil
		}
		if m.unturn {
			m.frame--
			if m.frame == 0 {
				// Face down again, with a different card under it, kept once it's turned like the first.
				m.unturn, m.revealed, m.again = false, false, false
				m.draw = redraw(m.draw, m.now, rand.New(rand.NewSource(time.Now().UnixNano())))
				_ = m.load()
				return m, nil
			}
			return m, nextFrame()
		}
		m.frame++
		if m.frame > len(flip) {
			m.frame, m.revealed = 0, true
			return m, nil
		}
		return m, nextFrame()
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case " ", "enter", "f":
			if !m.revealed && m.frame == 0 {
				if !m.preview {
					// Kept the moment it's turned, so closing mid-flip doesn't give a second chance.
					// Dated when it's turned, not when the pane opened: it may have been left open past midnight.
					m.draw.Date = day(time.Now())
					keep(m.dir, m.draw)
				}
				m.frame = 1
				return m, nextFrame()
			}
		case "c", "y":
			if m.revealed && m.frame == 0 {
				m.note = "copied the card and its reading"
				return m, tea.Batch(copyText(m.share()), tea.Tick(2*time.Second, func(time.Time) tea.Msg { return noteGone{} }))
			}
		case "n":
			if m.revealed && m.frame == 0 {
				m.frame, m.unturn = len(flip)-1, true
				return m, nextFrame()
			}
		case "j", "down":
			m.scroll++
		case "k", "up":
			m.scroll = max(0, m.scroll-1)
		}
	}
	return m, nil
}

func nextFrame() tea.Cmd {
	return tea.Tick(55*time.Millisecond, func(time.Time) tea.Msg { return tick{} })
}

const (
	bold   = "\x1b[1m"
	dim    = "\x1b[2m"
	italic = "\x1b[3m"
)

func (m model) View() string {
	w, h := max(m.width, 20), max(m.height, 10)
	head := fg(gold) + truncate("  ☾ tarot · "+m.now.Format("Monday 2 January"), w) + reset
	hint := m.hint()

	// Wide enough: the card on the left, its reading beside it. Otherwise the reading goes underneath.
	var body []string
	side := w >= 42+3+28
	if side {
		cols, rows := cardSize(w-3-28, h-2)
		textW := min(w-cols-3-2, 56)
		text := m.reading(textW)
		card := m.cardLines(cols, rows)
		margin := strings.Repeat(" ", max(0, (w-cols-3-textW)/2))
		top := max(0, (len(card)-len(text))/2)
		for i := 0; i < max(len(card), top+len(text)); i++ {
			line := margin + strings.Repeat(" ", cols)
			if i < len(card) {
				line = margin + card[i]
			}
			if t := i - top; t >= 0 && t < len(text) {
				line += "   " + text[t]
			}
			body = append(body, line)
		}
	} else {
		textW := min(w-2, 64)
		text := m.reading(textW)
		cols, rows := cardSize(w, h-2-1-len(text))
		for _, l := range m.cardLines(cols, rows) {
			body = append(body, strings.Repeat(" ", (w-cols)/2)+l)
		}
		body = append(body, "")
		for _, l := range text {
			body = append(body, strings.Repeat(" ", max(0, (w-textW)/2))+l)
		}
	}

	room := h - 2
	if len(body) < room {
		// Sit in the middle of the pane.
		pad := (room - len(body)) / 2
		body = append(make([]string, pad), body...)
	}
	scroll := clamp(m.scroll, 0, max(0, len(body)-room))
	body = body[scroll:min(len(body), scroll+room)]
	for len(body) < room {
		body = append(body, "")
	}
	return head + "\n" + strings.Join(body, "\n") + "\n" + hint
}

func (m model) cardLines(cols, rows int) []string {
	switch {
	case m.frame > 0 && flip[m.frame-1].face:
		return face(m.art, cols, rows, flip[m.frame-1].squeeze)
	case m.frame > 0:
		return back(cols, rows, flip[m.frame-1].squeeze)
	case m.revealed:
		return face(m.art, cols, rows, 1)
	default:
		return back(cols, rows, 1)
	}
}

// reading is the text under (or beside) the card: always the same number of lines, so nothing moves when the
// card turns over.
func (m model) reading(w int) []string {
	meaning, way := m.card.Upright, "upright"
	if m.draw.Reversed {
		meaning, way = m.card.Reversed, "reversed"
	}
	lines := []string{
		bold + fg(starGold) + truncate(m.card.Name, w) + reset,
		dim + way + reset,
		italic + fg(gold) + truncate(meaning.Keywords, w) + reset,
		"",
	}
	lines = append(lines, wrap(meaning.Text, w)...)
	if m.revealed {
		return lines
	}
	// Face down, the same space holds an invitation instead.
	out := []string{bold + fg(starGold) + truncate("Your card is face down.", w) + reset}
	for _, l := range wrap("Yours until midnight, or until you draw again.", w) {
		out = append(out, dim+l+reset)
	}
	for len(out) < len(lines) {
		out = append(out, "")
	}
	return out
}

func (m model) hint() string {
	var s string
	switch {
	case m.preview && !m.revealed:
		s = "space turns it over · q closes · preview"
	case m.note != "":
		s = m.note
	case m.preview:
		s = "c copy · n redraw · q close · preview"
	case !m.revealed:
		s = "space turns it over · q closes"
	case m.again:
		s = "today's card · c copy · n redraw · q close"
	default:
		s = "yours for today · c copy · n redraw · q close"
	}
	return dim + truncate("  "+s, m.width) + reset
}

// share is the card and its reading as plain text, for pasting anywhere.
func (m model) share() string {
	meaning, way := m.card.Upright, "upright"
	if m.draw.Reversed {
		meaning, way = m.card.Reversed, "reversed"
	}
	return fmt.Sprintf("☾ My tarot card for %s\n%s (%s)\n%s\n%s\n",
		m.now.Format("Monday 2 January"), m.card.Name, way, meaning.Keywords, meaning.Text)
}

// copyText puts text on the clipboard with OSC 52, which herdr passes on to the terminal it's attached from, even
// over ssh or mosh.
func copyText(text string) tea.Cmd {
	return func() tea.Msg {
		_, _ = osc52.New(text).WriteTo(os.Stdout)
		return nil
	}
}

func truncate(s string, w int) string {
	if utf8.RuneCountInString(s) <= w {
		return s
	}
	r := []rune(s)
	return string(r[:max(0, w-1)]) + "…"
}

// cardSize fits the card, frame and all, into cols×rows: the art keeps its 2:3 shape at two pixels a row.
func cardSize(cols, rows int) (int, int) {
	inner := min(40, min(cols-2, (rows-2)*4/3))
	inner = max(12, inner/4*4)
	return inner + 2, inner*3/4 + 2
}

func wrap(s string, w int) []string {
	var lines []string
	line := ""
	for _, word := range strings.Fields(s) {
		if line != "" && utf8.RuneCountInString(line)+1+utf8.RuneCountInString(word) > w {
			lines = append(lines, line)
			line = ""
		}
		if line != "" {
			line += " "
		}
		line += word
	}
	if line != "" {
		lines = append(lines, line)
	}
	return lines
}

func clamp(v, lo, hi int) int { return max(lo, min(hi, v)) }
