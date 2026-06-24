package tui

import (
	"image/color"
	"strings"

	"charm.land/lipgloss/v2"
)

// Synthwave palette — shared brand with lazyjsonl.
var (
	cBg      = lipgloss.Color("#130A25")
	cBar     = lipgloss.Color("#1E1140")
	cFg      = lipgloss.Color("#F9CDF6")
	cBright  = lipgloss.Color("#F4EDFF")
	cViolet  = lipgloss.Color("#9658FF")
	cCyan    = lipgloss.Color("#54DDFF")
	cMagenta = lipgloss.Color("#FF40B9")
	cYellow  = lipgloss.Color("#FFC102")
	cGreen   = lipgloss.Color("#56FF65")
	cRed     = lipgloss.Color("#FF4146")
	cMuted   = lipgloss.Color("#8A6FB8")
	cIdle    = lipgloss.Color("#54368E")
	cOrange  = lipgloss.Color("#FF9E64")
)

var (
	styleApp   = lipgloss.NewStyle().Foreground(cViolet).Bold(true)
	styleMuted = lipgloss.NewStyle().Foreground(cMuted)
	styleText  = lipgloss.NewStyle().Foreground(cFg)
	styleKey   = lipgloss.NewStyle().Foreground(cCyan).Bold(true)
	styleTopic = lipgloss.NewStyle().Foreground(cBright).Bold(true)
	styleOK    = lipgloss.NewStyle().Foreground(cGreen).Bold(true)
	styleWarn  = lipgloss.NewStyle().Foreground(cYellow).Bold(true)
)

// phaseStops returns the gradient color stops for a phase ("focus"/"short"/"long").
func phaseStops(phase string) []color.Color {
	switch phase {
	case "short":
		return []color.Color{cCyan, cViolet}
	case "long":
		return []color.Color{cGreen, cCyan}
	default: // focus
		return []color.Color{cViolet, cMagenta, cYellow}
	}
}

func phaseColor(phase string) color.Color {
	switch phase {
	case "short":
		return cCyan
	case "long":
		return cGreen
	default:
		return cMagenta
	}
}

func phaseLabel(phase string) string {
	switch phase {
	case "short":
		return "SHORT BREAK"
	case "long":
		return "LONG BREAK"
	default:
		return "FOCUS"
	}
}

// gradientText renders s with a synthwave ramp sliding by `frame` (shimmer).
func gradientText(s string, frame int) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	ramp := lipgloss.Blend1D(len(r)*3, cMagenta, cViolet, cCyan, cBright, cCyan, cViolet, cMagenta)
	var b strings.Builder
	for i, ch := range r {
		c := ramp[(i+frame)%len(ramp)]
		b.WriteString(lipgloss.NewStyle().Foreground(c).Bold(true).Render(string(ch)))
	}
	return b.String()
}

// keyHint renders "key desc" pairs for footers.
func keyHint(key, desc string) string {
	return styleKey.Render(key) + styleMuted.Render(" "+desc+"  ")
}

// goalRing renders a single ring glyph filling with daily-goal progress
// (yellow until met, green once the goal is reached).
func goalRing(done, goal int) string {
	if goal <= 0 {
		goal = 4
	}
	glyphs := []string{"○", "◔", "◑", "◕", "●"}
	i := int(float64(done) / float64(goal) * 4)
	if i > 4 {
		i = 4
	}
	col := cYellow
	if done >= goal {
		col = cGreen
	}
	return lipgloss.NewStyle().Foreground(col).Render(glyphs[i])
}

// miniBar renders a w-wide gradient progress bar (violet→magenta).
func miniBar(v, total, w int) string {
	if total <= 0 {
		total = 1
	}
	fill := v * w / total
	if fill > w {
		fill = w
	}
	ramp := lipgloss.Blend1D(max(2, w), cViolet, cMagenta)
	var b strings.Builder
	for i := 0; i < w; i++ {
		if i < fill {
			b.WriteString(lipgloss.NewStyle().Foreground(ramp[i]).Render("▰"))
		} else {
			b.WriteString(styleMuted.Render("▱"))
		}
	}
	return b.String()
}

// --- big block-digit clock ---------------------------------------------------

// glyphs are 5 rows tall; each digit is 4 cols, ":" is 2.
var glyphs = map[rune][5]string{
	'0': {"████", "█  █", "█  █", "█  █", "████"},
	'1': {"  █ ", " ██ ", "  █ ", "  █ ", "████"},
	'2': {"████", "   █", "████", "█   ", "████"},
	'3': {"████", "   █", "████", "   █", "████"},
	'4': {"█  █", "█  █", "████", "   █", "   █"},
	'5': {"████", "█   ", "████", "   █", "████"},
	'6': {"████", "█   ", "████", "█  █", "████"},
	'7': {"████", "   █", "  █ ", " █  ", " █  "},
	'8': {"████", "█  █", "████", "█  █", "████"},
	'9': {"████", "█  █", "████", "   █", "████"},
	':': {"  ", "██", "  ", "██", "  "},
}

// bigTime renders "MM:SS" as 5 rows of gradient-colored block glyphs. Only the
// filled cells are colored (spaces stay blank), so the digits glow.
func bigTime(s string, stops ...color.Color) string {
	var rows [5]string
	for _, ch := range s {
		g, ok := glyphs[ch]
		if !ok {
			continue
		}
		for r := 0; r < 5; r++ {
			rows[r] += g[r] + " "
		}
	}
	width := len([]rune(rows[0]))
	if width < 2 {
		return s
	}
	ramp := lipgloss.Blend1D(width, stops...)
	var b strings.Builder
	for r := 0; r < 5; r++ {
		for i, ch := range []rune(rows[r]) {
			if ch == '█' {
				b.WriteString(lipgloss.NewStyle().Foreground(ramp[i]).Render("█"))
			} else {
				b.WriteByte(' ')
			}
		}
		if r < 4 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
