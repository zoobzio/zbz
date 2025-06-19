package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

var levelColors = map[string]string{
	"debug":  "\033[34m",   // Blue
	"info":   "\033[32m",   // Green
	"warn":   "\033[33m",   // Yellow
	"error":  "\033[31m",   // Red
	"dpanic": "\033[35m",   // Magenta
	"panic":  "\033[35m",   // Magenta
	"fatal":  "\033[1;31m", // Bold Red
}

const (
	mediumGray = "\033[38;5;246m"
	reset      = "\033[0m"
)

// TODO this needs a lot of work
func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			// Not JSON, just print as-is
			fmt.Println(line)
			continue
		}

		// Extract and color the level
		level := strings.ToLower(fmt.Sprintf("%v", m["level"]))
		color := levelColors[level]
		if color == "" {
			color = reset
		}
		// Build header (customize as needed)
		time := fmt.Sprintf("%v", m["ts"])
		msg := fmt.Sprintf("%v", m["msg"])
		header := fmt.Sprintf("%s[%s] %s: %s%s", color, level, time, msg, reset)

		fmt.Println(header)

		// Print the rest of the fields, pretty and gray
		delete(m, "level")
		delete(m, "ts")
		delete(m, "msg")
		if len(m) > 0 {
			pretty, _ := json.MarshalIndent(m, "  │ ", "  ")
			fmt.Print(mediumGray)
			fmt.Println(string(pretty))
			fmt.Print(reset)
		}
	}
}
