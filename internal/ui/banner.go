package ui

import (
	"fmt"
	"math/rand"
	"runtime/debug"
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

var Version = getDynamicVersion()

func PrintBanner() {
	accent := lipgloss.NewStyle().
		Foreground(lipgloss.Color("99")).
		Bold(true)

	dim := lipgloss.NewStyle().
		Foreground(lipgloss.Color("8"))

	fmt.Println(accent.Render(asciiArt))
	fmt.Println()
	fmt.Println(dim.Render("  Welcome to Creo"))
	fmt.Println(dim.Render("  " + Version))
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

	totalFrames := 12
	// Pre-print 10 empty lines so we have space to move up without clipping
	for i := 0; i < 10; i++ {
		fmt.Println()
	}

	for frame := 0; frame < totalFrames; frame++ {
		// Move cursor up 10 lines (6 ascii + 1 empty + 2 text + 1 empty)
		fmt.Print("\033[10A")

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
		fmt.Println(glitchStyle.Render("  Welcome to Creo"))
		fmt.Println(glitchStyle.Render("  " + Version))
		fmt.Println()

		time.Sleep(110 * time.Millisecond)
	}

	// Final snap to clean banner
	fmt.Print("\033[10A")
	// Clear from cursor to end of screen to erase glitch artifacts cleanly
	fmt.Print("\033[J")
	PrintBanner()
}

func getDynamicVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "v0.3.1"
	}
	
	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	
	var revision string
	var modified bool
	for _, setting := range info.Settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			modified = setting.Value == "true"
		}
	}
	
	if revision != "" {
		short := revision
		if len(short) > 7 {
			short = short[:7]
		}
		suffix := ""
		if modified {
			suffix = " (dirty)"
		}
		return fmt.Sprintf("v0.3.1-dev-%s%s", short, suffix)
	}
	
	return "v0.3.1"
}
