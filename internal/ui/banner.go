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

// PrintBannerDissolve renders a reverse-dissolve animation where the Creo banner disintegrates and fades away.
func PrintBannerDissolve() {
	lines := strings.Split(strings.TrimPrefix(asciiArt, "\n"), "\n")
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	decayEarly := []rune{'░', '▒', '·', '∙'}
	decayLate := []rune{'·', '.', ' ', ' '}

	colorStages := []string{
		"99", "99",   // frames 0-1: bold purple (original)
		"97", "97",   // frames 2-3: slightly muted
		"97", "61",   // frames 4-5: desaturating
		"60", "60",   // frames 6-7: dark mauve
		"242", "240", // frames 8-9: medium gray
		"238", "236", // frames 10-11: dark gray
		"234", "234", // frames 12-13: near-invisible
		"234", "234", // frames 14-15: essentially gone
	}

	fmt.Print("\033[?25l")
	defer fmt.Print("\033[?25h")

	totalLines := 10
	for i := 0; i < totalLines; i++ {
		fmt.Println()
	}

	totalFrames := 16
	for frame := 0; frame < totalFrames; frame++ {
		fmt.Printf("\033[%dA", totalLines)

		var prob float64
		switch {
		case frame <= 5:
			prob = 0.15 * float64(frame+1) / 5.0
		case frame <= 11:
			prob = 0.3 + 0.5*float64(frame-6)/5.0
		default:
			prob = 0.85 + 0.15*float64(frame-12)/3.0
		}

		colorIdx := frame
		if colorIdx >= len(colorStages) {
			colorIdx = len(colorStages) - 1
		}
		charStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color(colorStages[colorIdx]))

		decayChars := decayEarly
		if frame >= 8 {
			decayChars = decayLate
		}

		for _, line := range lines {
			var sb strings.Builder
			for _, r := range []rune(line) {
				if r == ' ' {
					sb.WriteRune(' ')
				} else if rng.Float64() < prob {
					if frame >= 13 {
						sb.WriteRune(' ')
					} else {
						dc := decayChars[rng.Intn(len(decayChars))]
						sb.WriteString(charStyle.Render(string(dc)))
					}
				} else {
					sb.WriteString(charStyle.Render(string(r)))
				}
			}
			fmt.Println(sb.String())
		}

		fmt.Println()
		if frame < 12 {
			subtitle := "  Dropping Creo..."
			if frame >= 8 {
				subtitle = "  Dropping Creo" + strings.Repeat("·", frame-7)
			}
			fmt.Println(charStyle.Render(subtitle))
		} else {
			fmt.Println()
		}
		fmt.Println()
		fmt.Println()

		switch {
		case frame < 6:
			time.Sleep(110 * time.Millisecond)
		case frame < 12:
			time.Sleep(90 * time.Millisecond)
		default:
			time.Sleep(60 * time.Millisecond)
		}
	}

	// Final clear of the animated region
	fmt.Printf("\033[%dA", totalLines)
	fmt.Print("\033[J")
}


func getDynamicVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "v0.4.0"
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
		return fmt.Sprintf("v0.4.0-dev-%s%s", short, suffix)
	}
	
	return "v0.4.0"
}
