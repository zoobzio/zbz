package utils

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ANSI color codes for syntax highlighting
const (
	colorReset     = "\033[0m"
	colorRed       = "\033[31m"
	colorGreen     = "\033[32m"
	colorYellow    = "\033[33m"
	colorBlue      = "\033[34m"
	colorMagenta   = "\033[35m"
	colorCyan      = "\033[36m"
	colorWhite     = "\033[37m"
	colorBright    = "\033[1m"
	colorDim       = "\033[2m"
)

// typewrite simulates AI-style text generation with a typewriter effect
func typewrite(text string, delay time.Duration) {
	for _, char := range text {
		fmt.Print(string(char))
		if char == '.' || char == '!' || char == '?' {
			time.Sleep(delay * 3) // Longer pause at sentence end
		} else if char == ',' || char == ';' {
			time.Sleep(delay * 2) // Medium pause at punctuation
		} else {
			time.Sleep(delay)
		}
	}
	fmt.Println()
}

// Think writes text with a typewriter effect and thinking emoji
func Think(text string) {
	fmt.Print("💭 ")
	typewrite(text, 25*time.Millisecond)
}

// Explain writes explanatory text with a light bulb and typewriter effect
func Explain(text string) {
	fmt.Print("💡 ")
	typewrite(text, 20*time.Millisecond)
}

// Demonstrate shows what's happening with a pointer emoji
func Demonstrate(text string) {
	fmt.Print("👉 ")
	typewrite(text, 15*time.Millisecond)
}

// Observe comments on what we're seeing
func Observe(text string) {
	fmt.Print("👀 ")
	typewrite(text, 30*time.Millisecond)
}

// prettyJSON formats JSON with indentation and syntax highlighting
func prettyJSON(data interface{}) string {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting JSON: %v", err)
	}
	
	jsonStr := string(jsonBytes)
	
	// Apply syntax highlighting with regex patterns
	// Strings (values) - cyan
	stringValueRe := regexp.MustCompile(`:\s*"([^"]*)"`)
	jsonStr = stringValueRe.ReplaceAllString(jsonStr, fmt.Sprintf(": %s\"$1\"%s", colorCyan, colorReset))
	
	// Keys - bright blue
	keyRe := regexp.MustCompile(`"([^"]+)":`)
	jsonStr = keyRe.ReplaceAllString(jsonStr, fmt.Sprintf("%s\"$1\"%s:", colorBright+colorBlue, colorReset))
	
	// Numbers - yellow
	numberRe := regexp.MustCompile(`:\s*(\d+\.?\d*)`)
	jsonStr = numberRe.ReplaceAllString(jsonStr, fmt.Sprintf(": %s$1%s", colorYellow, colorReset))
	
	// Booleans - magenta
	boolRe := regexp.MustCompile(`:\s*(true|false)`)
	jsonStr = boolRe.ReplaceAllString(jsonStr, fmt.Sprintf(": %s$1%s", colorMagenta, colorReset))
	
	// null values - dim
	nullRe := regexp.MustCompile(`:\s*(null)`)
	jsonStr = nullRe.ReplaceAllString(jsonStr, fmt.Sprintf(": %s$1%s", colorDim, colorReset))
	
	// Brackets and braces - green
	jsonStr = strings.ReplaceAll(jsonStr, "{", colorGreen+"{"+colorReset)
	jsonStr = strings.ReplaceAll(jsonStr, "}", colorGreen+"}"+colorReset)
	jsonStr = strings.ReplaceAll(jsonStr, "[", colorGreen+"["+colorReset)
	jsonStr = strings.ReplaceAll(jsonStr, "]", colorGreen+"]"+colorReset)
	
	return jsonStr
}

// ShowJSON displays formatted JSON with a sparkle effect
func ShowJSON(data interface{}) {
	fmt.Println("✨ Here's what that looks like as JSON:")
	time.Sleep(300 * time.Millisecond)
	
	formatted := prettyJSON(data)
	
	// Print JSON with a slight delay to make it feel more dynamic
	lines := strings.Split(formatted, "\n")
	for _, line := range lines {
		fmt.Println("  " + line)
		time.Sleep(50 * time.Millisecond)
	}
	fmt.Println()
}

// Pause creates a brief thinking pause
func Pause() {
	time.Sleep(800 * time.Millisecond)
}

// ShortPause creates a quick breath
func ShortPause() {
	time.Sleep(300 * time.Millisecond)
}