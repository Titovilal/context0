package cmd

import "fmt"

// ANSI escape codes — shared across all commands for consistent styling.
const (
	cReset  = "\033[0m"
	cBold   = "\033[1m"
	cDim    = "\033[2m"
	cRed    = "\033[38;5;167m"
	cYellow = "\033[38;5;220m"
	cGreen  = "\033[32m"
	cBlue   = "\033[34m"
	cWhite  = "\033[97m"
)

// styled helpers — keep output DRY across commands.

func stHeader(s string) string { return cWhite + cBold + s + cReset }
func stOk(s string) string     { return cGreen + s + cReset }
func stWarn(s string) string    { return cYellow + s + cReset }
func stErr(s string) string     { return cRed + s + cReset }
func stDim(s string) string     { return cDim + s + cReset }
func stBold(s string) string    { return cBold + s + cReset }
func stValue(s string) string   { return cWhite + s + cReset }

// stStep prints a styled step indicator: "  ▸ message"
func stStep(msg string) { fmt.Println(cDim + "  ▸ " + cReset + msg) }

// stDone prints a styled success line: "  ✓ message"
func stDone(msg string) { fmt.Println(stOk("  ✓ ") + msg) }

// stSkip prints a styled skip line: "  – message"
func stSkip(msg string) { fmt.Println(stDim("  – " + msg)) }

// stTitle prints a section title with spacing.
func stTitle(title string) {
	fmt.Println()
	fmt.Println(stHeader("  " + title))
}
