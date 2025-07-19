// event_cycle_check.go: Static event cycle detection for event-driven architecture
// Usage: go run scripts/event_cycle_check.go
// Scans app/app.go and handler files for event handler registrations and emissions, builds the event flow graph, and detects cycles.
package main

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Edge represents a directed edge in the event flow graph
// from: event type consumed by handler
// to: event type emitted by handler
// handler: handler function name (for reporting)
type Edge struct {
	From    string
	To      string
	Handler string
}

func main() {
	// 1. Parse app/app.go for bus.Subscribe lines
	appFile, err := os.Open("app/app.go")
	if err != nil {
		fmt.Println("Error: could not open app/app.go:", err)
		os.Exit(1)
	}
	defer appFile.Close()

	subscribeRe := regexp.MustCompile(`bus\.Subscribe\("([A-Za-z0-9]+)"\s*,\s*([a-zA-Z0-9_.]+)\(`)
	handlerToEvent := make(map[string]string) // handler func -> event type consumed
	eventToHandlers := make(map[string][]string) // event type -> handler funcs

	scanner := bufio.NewScanner(appFile)
	for scanner.Scan() {
		line := scanner.Text()
		matches := subscribeRe.FindStringSubmatch(line)
		if len(matches) == 3 {
			eventType := matches[1]
			handlerFunc := matches[2]
			handlerToEvent[handlerFunc] = eventType
			eventToHandlers[eventType] = append(eventToHandlers[eventType], handlerFunc)
		}
	}

	// 2. Parse handler files for bus.Publish lines
	// (for simplicity, scan pkg/handler/ recursively for bus.Publish)
	publishRe := regexp.MustCompile(`bus\.Publish\(.*?([A-Za-z0-9]+Event)\{`)
	edges := []Edge{}

	err = walkDir("pkg/handler", func(path string) {
		f, err := os.Open(path)
		if err != nil {
			return
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		var currentHandler string
		for scanner.Scan() {
			line := scanner.Text()
			// Try to detect handler function name
			if strings.HasPrefix(line, "func ") && strings.Contains(line, "(") && strings.Contains(line, ")") {
				fn := strings.Fields(line)[1]
				if idx := strings.Index(fn, "("); idx > 0 {
					fn = fn[:idx]
				}
				currentHandler = fn
			}
			// Detect bus.Publish
			matches := publishRe.FindStringSubmatch(line)
			if len(matches) == 2 && currentHandler != "" {
				emittedEvent := matches[1]
				// Find which event this handler consumes
				consumedEvent := handlerToEvent[currentHandler]
				if consumedEvent != "" {
					edges = append(edges, Edge{From: consumedEvent, To: emittedEvent, Handler: currentHandler})
				}
			}
		}
	})
	if err != nil {
		fmt.Println("Error walking handler files:", err)
		os.Exit(1)
	}

	// 3. Build event flow graph and detect cycles
	graph := make(map[string][]string)
	for _, edge := range edges {
		graph[edge.From] = append(graph[edge.From], edge.To)
	}

	visited := make(map[string]bool)
	stack := make(map[string]bool)
	var hasCycle bool
	var dfs func(string) bool
	dfs = func(node string) bool {
		if stack[node] {
			fmt.Println("Cycle detected at:", node)
			hasCycle = true
			return true
		}
		if visited[node] {
			return false
		}
		visited[node] = true
		stack[node] = true
		for _, neighbor := range graph[node] {
			if dfs(neighbor) {
				return true
			}
		}
		stack[node] = false
		return false
	}

	fmt.Println("\nEvent Flow Graph:")
	for from, tos := range graph {
		fmt.Printf("  %s -> %v\n", from, tos)
	}

	fmt.Println("\nCycle Detection:")
	for node := range graph {
		if !visited[node] {
			dfs(node)
		}
	}
	if hasCycle {
		fmt.Println("\n❌ Event cycle(s) detected! Review your event flow.")
		os.Exit(1)
	} else {
		fmt.Println("\n✅ No event cycles detected.")
	}
}

// walkDir recursively walks a directory and calls fn for each .go file
func walkDir(dir string, fn func(path string)) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			if err := walkDir(dir+"/"+entry.Name(), fn); err != nil {
				return err
			}
		} else if strings.HasSuffix(entry.Name(), ".go") {
			fn(dir+"/"+entry.Name())
		}
	}
	return nil
}