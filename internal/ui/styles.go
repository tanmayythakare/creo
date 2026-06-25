package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

var (
	// Colors
	Accent       = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
	SuccessStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Bold(true)
	Failure      = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	Warning      = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Bold(true)
	Dim          = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	Bold         = lipgloss.NewStyle().Bold(true)

	// Symbols
	SymCheck = SuccessStyle.Render("✓")
	SymCross = Failure.Render("✗")
	SymDot   = SuccessStyle.Render("●")
	SymEmpty = Dim.Render("○")
	SymArrow = Dim.Render("→")

	// Header style
	Header = lipgloss.NewStyle().
		Foreground(lipgloss.Color("99")).
		Bold(true)

	// Section divider
	Divider = Dim.Render("──────────────────────────────────")
)

// SectionHeader prints "creo — <cmd>" style header
func SectionHeader(cmd string) string {
	return "  " + Accent.Render("creo") + Dim.Render(" — "+cmd)
}

// Truncate cleanly cuts a UTF-8 string to maxLen using visual cell width, appends "..." if exceeded
func Truncate(s string, maxLen int) string {
	if runewidth.StringWidth(s) <= maxLen {
		return s
	}

	var res strings.Builder
	width := 0
	limit := maxLen
	if maxLen > 3 {
		limit = maxLen - 3
	}

	for _, r := range s {
		w := runewidth.RuneWidth(r)
		if width+w > limit {
			break
		}
		res.WriteRune(r)
		width += w
	}

	if maxLen > 3 {
		res.WriteString("...")
	}
	return res.String()
}
