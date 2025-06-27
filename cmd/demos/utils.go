package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
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

// think writes text with a typewriter effect and thinking emoji
func think(text string) {
	fmt.Print("💭 ")
	typewrite(text, 25*time.Millisecond)
}

// explain writes explanatory text with a light bulb and typewriter effect
func explain(text string) {
	fmt.Print("💡 ")
	typewrite(text, 20*time.Millisecond)
}

// demonstrate shows what's happening with a pointer emoji
func demonstrate(text string) {
	fmt.Print("👉 ")
	typewrite(text, 15*time.Millisecond)
}

// observe comments on what we're seeing
func observe(text string) {
	fmt.Print("👀 ")
	typewrite(text, 30*time.Millisecond)
}

// prettyJSON formats JSON with indentation and syntax-like formatting
func prettyJSON(data interface{}) string {
	jsonBytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting JSON: %v", err)
	}
	
	// Add some basic syntax highlighting with colors (if terminal supports it)
	jsonStr := string(jsonBytes)
	
	// Simple formatting improvements
	jsonStr = strings.ReplaceAll(jsonStr, `"`, `"`)
	jsonStr = strings.ReplaceAll(jsonStr, `{`, `{\n`)
	jsonStr = strings.ReplaceAll(jsonStr, `}`, `\n}`)
	
	return jsonStr
}

// showJSON displays formatted JSON with a sparkle effect
func showJSON(data interface{}) {
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

// pause creates a brief thinking pause
func pause() {
	time.Sleep(800 * time.Millisecond)
}

// shortPause creates a quick breath
func shortPause() {
	time.Sleep(300 * time.Millisecond)
}