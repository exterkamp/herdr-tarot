# tarot

A little tarot oracle for [herdr](https://herdr.dev). Open it and one card lies face down;
turn it over to see what it says. That card is yours for the day: open it again before midnight and the same card
is waiting. About a third come up reversed, drawn upside down with the reversed reading.

```
go build -o tarot .
herdr plugin link .
```

Then bind it in `~/.config/herdr/config.toml`:

```toml
[[keys.command]]
key = "prefix+o"
type = "plugin_action"
command = "exterkamp.tarot.open"
description = "draw a tarot card"
```

On its own, `./tarot` runs in any truecolor terminal. `./tarot -card 19-sun` (add `-reversed` to turn it) shows a
card without keeping it.

| key | |
|---|---|
| space, enter | turn the card over |
| j / k | scroll, when the pane is short |
| q, esc | close |

The day's card is kept in `$HERDR_PLUGIN_STATE_DIR/today.json` (outside herdr, `~/.local/state/herdr-tarot`).

## The cards

The art is Pamela Colman Smith's, from the 1909 deck she illustrated for A. E. Waite, published by William
Rider & Son. It is in the public domain. The scans are from the 1909 "Roses & Lilies" printing on
[Wikimedia Commons](https://commons.wikimedia.org/wiki/Category:Rider-Waite_tarot_deck_(Roses_%26_Lilies)).
`tools/make_cards.py` crops each scan's title and numeral (the name is shown as text instead), reduces it to 40×60
pixels and maps the whole deck onto one shared 32-colour palette:

```
python3 tools/make_cards.py <folder of scans> cards
```

The terminal draws two pixels to a character cell with `▀`. `tools/ansi2png.py` turns a frame from
`./tarot -print -w 53 -h 40` into a PNG, for checking the layout without a terminal.

The readings in `meanings.go` were written for this program, after the cards' traditional meanings.

## Licence

The code and readings are Apache-2.0. The card art is in the public domain.
