// Package perf provides performance measurement and tracing for startup analysis.
// Enable with environment variable AUTO_WORKTREE_PERF=1 for human-readable output,
// or AUTO_WORKTREE_TRACE=<filename> to generate a trace file for `go tool trace`.
package perf

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime/trace"
	"sort"
	"strings"
	"sync"
	"time"
)

// Span represents a timed operation with hierarchical support.
type Span struct {
	Name      string
	StartTime time.Time
	EndTime   time.Time
	Parent    *Span
	Children  []*Span
	task      *trace.Task
	region    *trace.Region
	ctx       context.Context
}

// Duration returns the duration of this span.
func (s *Span) Duration() time.Duration {
	if s.EndTime.IsZero() {
		return time.Since(s.StartTime)
	}

	return s.EndTime.Sub(s.StartTime)
}

// Tracer manages performance measurement and trace output.
type Tracer struct {
	mu          sync.Mutex
	enabled     bool
	traceFile   *os.File
	rootSpans   []*Span
	activeSpans map[string]*Span
	startTime   time.Time
	output      io.Writer
}

var (
	// globalTracer is the singleton tracer instance
	globalTracer *Tracer
	once         sync.Once
)

// Init initializes the global tracer based on environment variables.
// Call this at the very beginning of main().
func Init() {
	once.Do(func() {
		globalTracer = &Tracer{
			activeSpans: make(map[string]*Span),
			startTime:   time.Now(),
			output:      os.Stderr,
		}

		// Check for perf mode (human-readable output)
		if os.Getenv("AUTO_WORKTREE_PERF") == "1" {
			globalTracer.enabled = true
		}

		// Check for trace file output (go tool trace compatible)
		if traceFile := os.Getenv("AUTO_WORKTREE_TRACE"); traceFile != "" {
			//nolint:gosec // G304: Path comes from trusted env variable, not user input
			f, err := os.Create(traceFile)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to create trace file: %v\n", err)
			} else {
				globalTracer.traceFile = f
				globalTracer.enabled = true

				err := trace.Start(f)
				if err != nil {
					//nolint:errcheck // Best-effort warning output
					fmt.Fprintf(os.Stderr, "Warning: failed to start trace: %v\n", err)
					//nolint:errcheck,gosec // Best-effort cleanup
					f.Close()

					globalTracer.traceFile = nil
				}
			}
		}
	})
}

// Enabled returns true if performance tracing is enabled.
func Enabled() bool {
	if globalTracer == nil {
		return false
	}

	return globalTracer.enabled
}

// StartSpan begins a new timed span. Returns a function to end the span.
// Usage:
//
//	end := perf.StartSpan("operation-name")
//	defer end()
func StartSpan(name string) func() {
	if globalTracer == nil || !globalTracer.enabled {
		return func() {}
	}

	return globalTracer.startSpan(name)
}

// StartSpanWithParent begins a new timed span as a child of the given parent.
func StartSpanWithParent(name string, parentName string) func() {
	if globalTracer == nil || !globalTracer.enabled {
		return func() {}
	}

	return globalTracer.startSpanWithParent(name, parentName)
}

func (t *Tracer) startSpan(name string) func() {
	t.mu.Lock()

	span := &Span{
		Name:      name,
		StartTime: time.Now(),
		ctx:       context.Background(),
	}

	// Create trace task if tracing is active
	if t.traceFile != nil {
		ctx, task := trace.NewTask(span.ctx, name)
		span.ctx = ctx
		span.task = task
		span.region = trace.StartRegion(ctx, name)
	}

	t.rootSpans = append(t.rootSpans, span)
	t.activeSpans[name] = span

	t.mu.Unlock()

	return func() {
		t.endSpan(name)
	}
}

func (t *Tracer) startSpanWithParent(name string, parentName string) func() {
	t.mu.Lock()

	span := &Span{
		Name:      name,
		StartTime: time.Now(),
		ctx:       context.Background(),
	}

	// Find parent span
	if parent, ok := t.activeSpans[parentName]; ok {
		span.Parent = parent
		span.ctx = parent.ctx
		parent.Children = append(parent.Children, span)
	} else {
		// No parent found, add as root
		t.rootSpans = append(t.rootSpans, span)
	}

	// Create trace region if tracing is active
	if t.traceFile != nil {
		span.region = trace.StartRegion(span.ctx, name)
	}

	t.activeSpans[name] = span

	t.mu.Unlock()

	return func() {
		t.endSpan(name)
	}
}

func (t *Tracer) endSpan(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	span, ok := t.activeSpans[name]
	if !ok {
		return
	}

	span.EndTime = time.Now()

	// End trace region/task
	if span.region != nil {
		span.region.End()
	}

	if span.task != nil {
		span.task.End()
	}

	delete(t.activeSpans, name)
}

// Shutdown finalizes tracing and outputs the summary.
// Call this at the end of main() using defer.
func Shutdown() {
	if globalTracer == nil || !globalTracer.enabled {
		return
	}

	globalTracer.shutdown()
}

func (t *Tracer) shutdown() {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Stop trace if active
	if t.traceFile != nil {
		trace.Stop()
		//nolint:errcheck,gosec // Best-effort cleanup
		t.traceFile.Close()
		//nolint:errcheck // Best-effort debug output
		_, _ = fmt.Fprintf(t.output, "\nTrace written to: %s\n", t.traceFile.Name())
		//nolint:errcheck // Best-effort debug output
		_, _ = fmt.Fprintf(t.output, "View with: go tool trace %s\n\n", t.traceFile.Name())
	}

	// Print summary
	t.printSummary()
}

//nolint:errcheck // Best-effort debug output - errors writing to stderr are not actionable
func (t *Tracer) printSummary() {
	totalDuration := time.Since(t.startTime)

	_, _ = fmt.Fprintf(t.output, "\n")
	_, _ = fmt.Fprintf(t.output, "╔══════════════════════════════════════════════════════════════════╗\n")
	_, _ = fmt.Fprintf(t.output, "║                    PERFORMANCE SUMMARY                           ║\n")
	_, _ = fmt.Fprintf(t.output, "╠══════════════════════════════════════════════════════════════════╣\n")
	_, _ = fmt.Fprintf(t.output, "║ Total startup time: %-44s ║\n", totalDuration.Round(time.Microsecond))
	_, _ = fmt.Fprintf(t.output, "╚══════════════════════════════════════════════════════════════════╝\n\n")

	// Collect all spans (flatten for sorting)
	var allSpans []*Span
	var collectSpans func(spans []*Span, depth int)
	collectSpans = func(spans []*Span, depth int) {
		for _, s := range spans {
			allSpans = append(allSpans, s)
			collectSpans(s.Children, depth+1)
		}
	}
	collectSpans(t.rootSpans, 0)

	// Sort by duration (longest first)
	sort.Slice(allSpans, func(i, j int) bool {
		return allSpans[i].Duration() > allSpans[j].Duration()
	})

	// Print timing breakdown
	_, _ = fmt.Fprintf(t.output, "Timing Breakdown (sorted by duration):\n")
	_, _ = fmt.Fprintf(t.output, "────────────────────────────────────────────────────────────────────\n")
	_, _ = fmt.Fprintf(t.output, "%-45s %12s %8s\n", "OPERATION", "DURATION", "% TOTAL")
	_, _ = fmt.Fprintf(t.output, "────────────────────────────────────────────────────────────────────\n")

	for _, span := range allSpans {
		duration := span.Duration()
		percent := float64(duration) / float64(totalDuration) * 100

		// Indent based on parent hierarchy
		depth := 0

		for p := span.Parent; p != nil; p = p.Parent {
			depth++
		}

		indent := strings.Repeat("  ", depth)
		name := indent + span.Name

		// Truncate name if too long
		if len(name) > 43 {
			name = name[:40] + "..."
		}

		bar := t.progressBar(percent, 8)
		_, _ = fmt.Fprintf(t.output, "%-45s %12s %s %.1f%%\n",
			name,
			duration.Round(time.Microsecond),
			bar,
			percent)
	}

	_, _ = fmt.Fprintf(t.output, "────────────────────────────────────────────────────────────────────\n\n")

	// Print hierarchical view
	_, _ = fmt.Fprintf(t.output, "Hierarchical View:\n")
	_, _ = fmt.Fprintf(t.output, "────────────────────────────────────────────────────────────────────\n")
	t.printHierarchy(t.rootSpans, 0, totalDuration)
	_, _ = fmt.Fprintf(t.output, "\n")
}

//nolint:errcheck // Best-effort debug output
func (t *Tracer) printHierarchy(spans []*Span, depth int, totalDuration time.Duration) {
	// Sort spans by start time
	sorted := make([]*Span, len(spans))
	copy(sorted, spans)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].StartTime.Before(sorted[j].StartTime)
	})

	for _, span := range sorted {
		indent := strings.Repeat("│ ", depth)
		prefix := "├─"

		if depth == 0 {
			prefix = "▸ "
		}

		duration := span.Duration()
		percent := float64(duration) / float64(totalDuration) * 100

		_, _ = fmt.Fprintf(t.output, "%s%s %s (%s, %.1f%%)\n",
			indent,
			prefix,
			span.Name,
			duration.Round(time.Microsecond),
			percent)

		if len(span.Children) > 0 {
			t.printHierarchy(span.Children, depth+1, totalDuration)
		}
	}
}

func (t *Tracer) progressBar(percent float64, width int) string {
	filled := int(percent / 100.0 * float64(width))

	if filled > width {
		filled = width
	}

	if filled < 0 {
		filled = 0
	}

	return "[" + strings.Repeat("█", filled) + strings.Repeat("░", width-filled) + "]"
}

// Mark records a named milestone timestamp for later analysis.
func Mark(name string) {
	if globalTracer == nil || !globalTracer.enabled {
		return
	}

	globalTracer.mark(name)
}

//nolint:errcheck // Best-effort debug output
func (t *Tracer) mark(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	elapsed := time.Since(t.startTime)
	_, _ = fmt.Fprintf(t.output, "[PERF] %s: %s (elapsed: %s)\n", time.Now().Format("15:04:05.000"), name, elapsed.Round(time.Microsecond))
}
