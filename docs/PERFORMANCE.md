# Performance Analysis Guide

This document describes how to measure and analyze the startup performance of auto-worktree.

## Quick Start

### Human-Readable Performance Summary

Run any auto-worktree command with `AUTO_WORKTREE_PERF=1`:

```bash
AUTO_WORKTREE_PERF=1 auto-worktree version
AUTO_WORKTREE_PERF=1 auto-worktree list
AUTO_WORKTREE_PERF=1 auto-worktree  # interactive mode
```

This outputs:
- Milestone markers with timestamps
- Performance summary table sorted by duration
- Hierarchical view of timing spans

### Go Trace File (for detailed analysis)

Generate a trace file compatible with `go tool trace`:

```bash
AUTO_WORKTREE_TRACE=startup.trace auto-worktree version
go tool trace startup.trace
```

This opens an interactive browser UI with:
- Goroutine analysis
- Network/sync blocking analysis
- Syscall analysis
- Scheduler latency analysis
- Flame graphs

## Understanding the Output

### Timing Breakdown

```
OPERATION                                         DURATION  % TOTAL
────────────────────────────────────────────────────────────────────
main                                              74.814ms [███████░] 100.0%
  startup-cleanup                                 74.726ms [███████░] 99.9%
cleanup-get-candidates                            63.748ms [██████░░] 85.2%
```

- **OPERATION**: Name of the instrumented span
- **DURATION**: Wall-clock time spent in the operation
- **% TOTAL**: Percentage of total startup time
- **[████░░░░]**: Visual progress bar

### Hierarchical View

```
▸  main (74.814ms, 100.0%)
│ ├─ startup-cleanup (74.726ms, 99.9%)
│ ├─ run-command (3µs, 0.0%)
```

Shows parent-child relationships between operations.

## Instrumented Spans

The following operations are instrumented:

### Main Entry Points
| Span Name | Description |
|-----------|-------------|
| `main` | Total main() execution |
| `startup-cleanup` | Orphaned/merged worktree cleanup |
| `interactive-menu` | TUI menu display and interaction |
| `run-command` | CLI command execution |

### Startup Cleanup
| Span Name | Description |
|-----------|-------------|
| `cleanup-repo-init` | Repository initialization for cleanup |
| `cleanup-get-candidates` | Finding cleanup candidates |

### Git Repository Operations
| Span Name | Description |
|-----------|-------------|
| `git-repo-init-total` | Total repository initialization |
| `git-is-repository` | Check if in git repo (`git rev-parse --git-dir`) |
| `git-get-root` | Get repo root (`git rev-parse --show-toplevel`) |
| `git-get-homedir` | Get user home directory |
| `git-new-config` | Create config object |

### Worktree Operations
| Span Name | Description |
|-----------|-------------|
| `git-list-worktrees-with-merge-status` | Full worktree listing with merge status |
| `git-basic-worktree-list` | Basic worktree list |
| `git-worktree-list` | Raw `git worktree list --porcelain` |
| `git-worktree-parse-enrich` | Parse and enrich worktree data |
| `git-enrich-merge-status-all` | Check merge status for all worktrees |

### TUI Operations
| Span Name | Description |
|-----------|-------------|
| `menu-items-create` | Create menu item list |
| `menu-model-create` | Create Bubbletea menu model |
| `tea-program-create` | Create Bubbletea program |
| `tea-program-run` | Run TUI (includes render + user interaction) |

## Baseline Measurements

### Measuring Startup Time

For consistent measurements, run multiple times and average:

```bash
# Run 10 iterations, capture timing
for i in {1..10}; do
  AUTO_WORKTREE_PERF=1 auto-worktree version 2>&1 | grep "Total startup time"
done
```

### Measuring Time to First Render

The `menu-ready-to-render` milestone marks when the TUI is ready:

```bash
AUTO_WORKTREE_PERF=1 auto-worktree 2>&1 | grep "menu-ready-to-render"
```

## Common Performance Issues

### 1. Slow Startup Cleanup (>50ms)

**Symptom**: `startup-cleanup` or `cleanup-get-candidates` takes >50ms

**Cause**: Many worktrees requiring merge status checks

**Solution**:
- Reduce number of worktrees
- Consider lazy loading merge status
- Parallelize merge status checks

### 2. Slow Repository Init (>20ms)

**Symptom**: `git-repo-init-total` takes >20ms

**Cause**: Git subprocess overhead

**Solution**:
- Cache repository metadata
- Use libgit2 bindings instead of subprocess

### 3. Slow Worktree Enrichment (>30ms)

**Symptom**: `git-worktree-parse-enrich` takes >30ms

**Cause**: Multiple git commands per worktree

**Solution**:
- Batch git operations
- Parallelize worktree enrichment
- Cache worktree metadata

## Adding New Instrumentation

To add new timing spans:

```go
import "github.com/kaeawc/auto-worktree/internal/perf"

func myFunction() {
    // Simple span
    end := perf.StartSpan("my-operation")
    defer end()

    // ... operation code ...
}

func myNestedFunction() {
    endOuter := perf.StartSpan("outer-operation")
    defer endOuter()

    // Child span
    endInner := perf.StartSpanWithParent("inner-operation", "outer-operation")
    // ... inner code ...
    endInner()
}

func myMilestone() {
    // Record a milestone timestamp
    perf.Mark("milestone-name")
}
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `AUTO_WORKTREE_PERF=1` | Enable human-readable performance output |
| `AUTO_WORKTREE_TRACE=<file>` | Generate Go trace file |

Both can be used simultaneously:

```bash
AUTO_WORKTREE_PERF=1 AUTO_WORKTREE_TRACE=trace.out auto-worktree version
```

## Performance Targets

| Metric | Target | Before | After |
|--------|--------|--------|-------|
| Time to version output | <50ms | ~86ms | **~60µs** ✓ |
| Time to menu render | <100ms | TBD | ~53ms ✓ |
| Repository init | <10ms | ~11ms | **~6ms** ✓ |

## Optimization Results (January 2026)

### Summary

| Command | Before | After | Improvement |
|---------|--------|-------|-------------|
| `version` | 86ms | **60µs** | **1433x faster** |
| `help` | 86ms | **60µs** | **1433x faster** |
| `list` (with cleanup) | 86ms | **53ms** | **38% faster** |

### Optimizations Implemented

1. **P0: Skip cleanup for simple commands** - `version`, `help` skip cleanup entirely
2. **P1: Parallel worktree enrichment** - Goroutines for concurrent git calls
3. **P2: Parallel merge status checking** - Concurrent merge status checks (14ms vs 28ms)
4. **P3: Combined git subprocess calls** - Single `rev-parse` call (6ms vs 11ms)

### Bubbletea Performance Tips (from community research)

- Keep the event loop fast - offload expensive operations to `tea.Cmd`
- Use hierarchical models for better message routing
- Avoid blocking operations in `Update()`
- Use `tea.Sequence()` for order-dependent operations

## Baseline Analysis (January 2026)

### Current State

**Average startup time**: ~86ms (range: 81-92ms)

### Timing Breakdown

| Operation | Duration | % Total | Notes |
|-----------|----------|---------|-------|
| `startup-cleanup` | 81.8ms | 99.9% | **Dominant cost** |
| `cleanup-get-candidates` | 70.7ms | 86.4% | Worktree scanning |
| `git-basic-worktree-list` | 42.0ms | 51.4% | git worktree list + parse |
| `git-worktree-parse-enrich` | 36.5ms | 44.6% | Per-worktree enrichment |
| `git-enrich-merge-status-all` | 28.7ms | 35.0% | Merge status checks |
| `cleanup-repo-init` | 11.1ms | 13.5% | Repository initialization |
| `git-is-repository` | 5.5ms | 6.7% | `git rev-parse --git-dir` |
| `git-get-root` | 5.6ms | 6.8% | `git rev-parse --show-toplevel` |
| `run-command` | 0.002ms | 0.0% | Command execution overhead |

### Key Findings

1. **Startup cleanup is the bottleneck** - 99.9% of startup time
2. **Worktree operations dominate** - Full worktree list with enrichment takes 70ms
3. **Git subprocess overhead** - Each git command costs ~5ms
4. **Sequential processing** - Per-worktree operations run sequentially

## Identified Optimization Opportunities

### High Impact (>30ms savings potential)

#### 1. Lazy/Async Startup Cleanup
- **Current**: Runs synchronously before showing UI
- **Proposed**: Run in background after menu is displayed
- **Estimated savings**: 70-80ms
- **Complexity**: Medium
- **Risk**: User might see cleanup prompts after menu loads

#### 2. Cache Worktree Metadata
- **Current**: Full `git worktree list` on every startup
- **Proposed**: Cache worktree list, invalidate on file changes
- **Estimated savings**: 30-40ms (on subsequent runs)
- **Complexity**: Medium-High
- **Risk**: Stale cache data

#### 3. Parallelize Worktree Enrichment
- **Current**: Sequential per-worktree git commands
- **Proposed**: Use goroutines for parallel enrichment
- **Estimated savings**: 20-30ms
- **Complexity**: Low-Medium
- **Risk**: Git lock contention

### Medium Impact (5-20ms savings potential)

#### 4. Lazy Load Merge Status
- **Current**: Check merge status for all worktrees upfront
- **Proposed**: Only check when cleanup is triggered
- **Estimated savings**: 25-30ms
- **Complexity**: Low
- **Risk**: Delayed cleanup prompts

#### 5. Combine Git Subprocess Calls
- **Current**: Separate calls for `--git-dir` and `--show-toplevel`
- **Proposed**: Single combined call
- **Estimated savings**: 5ms
- **Complexity**: Low
- **Risk**: Low

### Low Impact (optimization deferred)

#### 6. Use libgit2 Bindings
- **Current**: Shell out to git CLI
- **Proposed**: Use go-git or libgit2 bindings
- **Estimated savings**: 10-20ms
- **Complexity**: High
- **Risk**: Compatibility, maintenance burden

## Optimization Priority

| Priority | Optimization | Expected | Actual | Status |
|----------|--------------|----------|--------|--------|
| P0 | Skip cleanup for simple commands | 70-80ms | **86ms** | ✅ Implemented |
| P1 | Parallelize worktree enrichment | 20-30ms | **~15ms** | ✅ Implemented |
| P2 | Parallel merge status checking | 25-30ms | **~14ms** | ✅ Implemented |
| P3 | Combine git subprocess calls | 5ms | **~5ms** | ✅ Implemented |
| P4 | Cache worktree list | 30-40ms | - | Consider for v2 |

## Related Files

- `internal/perf/tracer.go` - Performance measurement framework
- `cmd/auto-worktree/main.go` - Main entry point instrumentation
- `internal/cmd/commands.go` - Command instrumentation
- `internal/git/repository.go` - Git operation instrumentation
- `internal/git/worktree.go` - Worktree operation instrumentation
