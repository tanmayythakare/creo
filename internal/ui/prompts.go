package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var reader = bufio.NewReader(os.Stdin)

// Ask prompts for a string input with an optional default value
func Ask(question, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("  %s: ", question)
	} else {
		fmt.Printf("  %s: ", question)
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return defaultVal
	}
	return input
}

// Confirm asks a yes/no question, returns true for yes
func Confirm(question string, defaultYes bool) bool {
	hint := "[Y/n]"
	if !defaultYes {
		hint = "[y/N]"
	}

	fmt.Printf("  %s %s ", question, Dim.Render(hint))
	input, _ := reader.ReadString('\n')
	input = strings.ToLower(strings.TrimSpace(input))

	if input == "" {
		return defaultYes
	}
	return input == "y" || input == "yes"
}

// Success prints a styled success line
func Success(msg string) {
	fmt.Println("  " + SymCheck + " " + msg)
}

// Warn prints a styled warning line
func Warn(msg string) {
	fmt.Println("  " + Warning.Render("⚠") + " " + msg)
}

// Error prints a styled error line
func Error(msg string) {
	fmt.Println("  " + SymCross + " " + msg)
}

// Info prints a plain info line
func Info(msg string) {
	fmt.Println(Dim.Render("  " + msg))
}

// Section prints a bold section header
func Section(title string) {
	fmt.Println()
	fmt.Println(Accent.Render("── " + title))
}
