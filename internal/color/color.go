package color

import (
	"fmt"
	"os"
	"strings"
)

// ANSI color codes
const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Bold    = "\033[1m"
)

// Color represents a colorizer that can be enabled or disabled
type Color struct {
	enabled bool
}

// New creates a new Color instance
func New(enabled bool) *Color {
	return &Color{enabled: enabled && shouldEnableColor()}
}

// shouldEnableColor determines if color should be enabled based on environment
func shouldEnableColor() bool {
	// Check NO_COLOR environment variable (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check TERM environment variable
	term := os.Getenv("TERM")
	if term == "dumb" || term == "" {
		return false
	}

	// Check if output is to a terminal
	// This is a simplified check - in a real implementation you might use
	// a package like github.com/mattn/go-isatty
	return true
}

// Add colors a string to indicate additions (green, like Terraform)
func (c *Color) Add(text string) string {
	if !c.enabled {
		return text
	}
	return Green + text + Reset
}

// Change colors a string to indicate modifications (yellow, like Terraform)
func (c *Color) Change(text string) string {
	if !c.enabled {
		return text
	}
	return Yellow + text + Reset
}

// Destroy colors a string to indicate deletions (red, like Terraform)
func (c *Color) Destroy(text string) string {
	if !c.enabled {
		return text
	}
	return Red + text + Reset
}

// Bold makes text bold
func (c *Color) Bold(text string) string {
	if !c.enabled {
		return text
	}
	return Bold + text + Reset
}

// Cyan colors text cyan (for headers and labels)
func (c *Color) Cyan(text string) string {
	if !c.enabled {
		return text
	}
	return Cyan + text + Reset
}

// Blue colors text blue
func (c *Color) Blue(text string) string {
	if !c.enabled {
		return text
	}
	return Blue + text + Reset
}

// PlanSymbol returns the appropriate symbol for plan actions
func (c *Color) PlanSymbol(action string) string {
	switch action {
	case "add", "create":
		return c.Add("+")
	case "change", "modify", "update":
		return c.Change("~")
	case "destroy", "drop", "delete":
		return c.Destroy("-")
	default:
		return " "
	}
}

// FormatPlanLine formats a line in Terraform plan style
func (c *Color) FormatPlanLine(symbol, objectType, name, action string) string {
	coloredSymbol := c.PlanSymbol(action)
	if name == "" {
		return fmt.Sprintf("  %s %s", coloredSymbol, objectType)
	}
	return fmt.Sprintf("  %s %s.%s", coloredSymbol, objectType, name)
}

// FormatSummaryLine formats summary counts with colors
func (c *Color) FormatSummaryLine(objectType string, added, modified, dropped int) string {
	// Always show all three categories, even if zero
	parts := []string{
		c.Add(fmt.Sprintf("%d to add", added)),
		c.Change(fmt.Sprintf("%d to modify", modified)),
		c.Destroy(fmt.Sprintf("%d to drop", dropped)),
	}
	
	return fmt.Sprintf("  %s: %s", objectType, strings.Join(parts, ", "))
}

// FormatPlanHeader formats the main plan header
func (c *Color) FormatPlanHeader(added, modified, dropped int) string {
	// Always show all three categories, even if zero
	parts := []string{
		c.Add(fmt.Sprintf("%d to add", added)),
		c.Change(fmt.Sprintf("%d to modify", modified)),
		c.Destroy(fmt.Sprintf("%d to drop", dropped)),
	}
	
	return fmt.Sprintf("Plan: %s.", strings.Join(parts, ", "))
}