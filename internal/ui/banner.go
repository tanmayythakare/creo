package ui

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const asciiArt = `
 ██████╗██████╗ ███████╗ ██████╗
██╔════╝██╔══██╗██╔════╝██╔═══██╗
██║     ██████╔╝█████╗  ██║   ██║
██║     ██╔══██╗██╔══╝  ██║   ██║
╚██████╗██║  ██║███████╗╚██████╔╝
 ╚═════╝╚═╝  ╚═╝╚══════╝ ╚═════╝`

func PrintBanner() {
	accent := lipgloss.NewStyle().
		Foreground(lipgloss.Color("99")).
		Bold(true)

	dim := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	fmt.Println(accent.Render(asciiArt))
	fmt.Println()
	fmt.Println(dim.Render("  Welcome to Creo — kill startup friction"))
	fmt.Println(dim.Render("  v0.1.0"))
	fmt.Println()
}

func PrintBannerWithGlitch() {
	glitchChars := []rune{'#', '@', '!', '%', '&', '/', '?', '~', '^', '*', '$', 'a', 'x', '7', '0', '1', '2'}
	lines := strings.Split(strings.TrimPrefix(asciiArt, "\n"), "\n")

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Print escape sequence to hide cursor
	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h") // Restore cursor on exit

	accent := lipgloss.NewStyle().
		Foreground(lipgloss.Color("99")).
		Bold(true)

	glitchStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	totalFrames := 6
	for frame := 0; frame < totalFrames; frame++ {
		// Clear screen and reset cursor
		fmt.Print("\033[H\033[2J")

		// Glitch probability decays linearly
		prob := 0.85 * (1.0 - float64(frame)/float64(totalFrames))

		for _, line := range lines {
			var sb strings.Builder
			runes := []rune(line)
			for _, r := range runes {
				if r == ' ' {
					sb.WriteRune(' ')
				} else if rng.Float64() < prob {
					gChar := glitchChars[rng.Intn(len(glitchChars))]
					sb.WriteString(glitchStyle.Render(string(gChar)))
				} else {
					sb.WriteString(accent.Render(string(r)))
				}
			}
			fmt.Println(sb.String())
		}

		fmt.Println()
		fmt.Println(glitchStyle.Render("  Welcome to Creo — kill startup friction"))
		fmt.Println(glitchStyle.Render("  v0.1.0"))
		fmt.Println()

		time.Sleep(45 * time.Millisecond)
	}

	// Final snap to clean banner
	fmt.Print("\033[H\033[2J")
	PrintBanner()
}
