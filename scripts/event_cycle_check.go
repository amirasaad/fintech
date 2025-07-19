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

// Helper to normalize handler function names (strip package prefix)
func normalizeHandlerName(name string) string {
	if idx := strings.LastIndex(name, "."); idx != -1 {
		return name[idx+1:]
	}
	return name
}

func eventCycleCheck() int {
	// 1. Parse app/app.go for bus.Subscribe lines
	appFile, err := os.Open("app/app.go")
	if err != nil {
		fmt.Println("Error: could not open app/app.go:", err)
		return 1
	}
	defer appFile.Close() //nolint:errcheck

	subscribeRe := regexp.MustCompile(`bus\.Subscribe\("([A-Za-z0-9]+)"\s*,\s*([a-zA-Z0-9_.]+)\(`)
	handlerToEvent := make(map[string]string)    // handler func -> event type consumed
	eventToHandlers := make(map[string][]string) // event type -> handler funcs

	scanner := bufio.NewScanner(appFile)
	for scanner.Scan() {
		line := scanner.Text()
		matches := subscribeRe.FindStringSubmatch(line)
		if len(matches) == 3 {
			eventType := matches[1]
			handlerFunc := normalizeHandlerName(matches[2])
			handlerToEvent[handlerFunc] = eventType
			eventToHandlers[eventType] = append(eventToHandlers[eventType], handlerFunc)
		}
	}

	// 2. Parse handler files for bus.Publish lines
	// (for simplicity, scan pkg/handler/ recursively for bus.Publish)
	// Improved regex to match more event emission patterns
	publishRe := regexp.MustCompile(`bus\.Publish\([^,]+,\s*(?:[a-zA-Z0-9_]+\.)*([A-Za-z0-9]+Event)\s*\{`)
	handlerEmits := make(map[string][]string) // handler func -> []emitted event types

	err = walkDir("pkg/handler", func(path string) {
		f, err := os.Open(path)
		if err != nil {
			return
		}
		defer func() {
			if err := f.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "failed to close file: %v\n", err)
			}
		}()
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
				currentHandler = normalizeHandlerName(fn)
			}
			// Detect bus.Publish
			matches := publishRe.FindStringSubmatch(line)
			if len(matches) == 2 && currentHandler != "" {
				emittedEvent := matches[1]
				handlerEmits[currentHandler] = append(handlerEmits[currentHandler], emittedEvent)
			}
		}
	})
	if err != nil {
		fmt.Println("Error walking handler files:", err)
		return 1
	}

	// 3. Build event-to-event graph and detect cycles
	graph := make(map[string][]string)
	for eventType, handlers := range eventToHandlers {
		for _, handler := range handlers {
			emittedEvents := handlerEmits[handler]
			graph[eventType] = append(graph[eventType], emittedEvents...)
		}
	}

	visited := make(map[string]bool)
	stack := make(map[string]bool)
	var hasCycle bool
	var path []string
	var dfs func(string) bool
	dfs = func(node string) bool {
		if stack[node] {
			fmt.Println("Cycle detected:", append(path, node))
			hasCycle = true
			return true
		}
		if visited[node] {
			return false
		}
		visited[node] = true
		stack[node] = true
		path = append(path, node)
		for _, neighbor := range graph[node] {
			if dfs(neighbor) {
				return true
			}
		}
		stack[node] = false
		path = path[:len(path)-1]
		return false
	}

	fmt.Println("\nEvent Flow Graph:")
	for from, tos := range graph {
		fmt.Printf("  %s -> %v\n", from, tos)
	}

	fmt.Println("\nCycle Detection:")
	hasCycle = false
	visited = make(map[string]bool)
	stack = make(map[string]bool)
	path = nil
	for node := range graph {
		if !visited[node] {
			dfs(node)
		}
	}
	if hasCycle {
		fmt.Println("\n❌ Event cycle(s) detected! Review your event flow.")
		return 1
	}
	// Only print success if no cycles were detected
	fmt.Println("\n✅ No event cycles detected.")
	return 0
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
			fn(dir + "/" + entry.Name())
		}
	}
	return nil
}

func main() {
	os.Exit(eventCycleCheck())
}
