package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var frames = []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}

var spinStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("99"))

// SpinUntil shows a spinner with label until done() returns true
func SpinUntil(label string, action func()) {
	done := make(chan struct{})
	exited := make(chan struct{})
	go func() {
		defer close(exited)
		i := 0
		for {
			select {
			case <-done:
				return
			default:
				fmt.Printf("\r  %s %s...",
					spinStyle.Render(frames[i%len(frames)]),
					label,
				)
				i++
				time.Sleep(80 * time.Millisecond)
			}
		}
	}()
	action()
	close(done)
	<-exited
	// Clear the line and print the success checkmark with a cleaned label
	var cleanLabel string
	if strings.HasPrefix(label, "relaunching ") {
		cleanLabel = strings.TrimPrefix(label, "relaunching ") + "   launched"
	} else {
		cleanLabel = strings.TrimPrefix(label, "launching ")
	}
	fmt.Printf("\r  %s %s        \n", SymCheck, cleanLabel)
}
