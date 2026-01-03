#!/bin/bash
# Measure startup performance of auto-worktree
# Usage: ./scripts/measure-startup.sh [iterations]

set -e

ITERATIONS=${1:-5}
BINARY="./aw-perf-test"

echo "Building auto-worktree..."
go build -o "$BINARY" ./cmd/auto-worktree

echo ""
echo "Running $ITERATIONS iterations of 'version' command..."
echo "=============================================="

declare -a TIMES

for i in $(seq 1 $ITERATIONS); do
    # Extract total startup time from output
    OUTPUT=$(AUTO_WORKTREE_PERF=1 "$BINARY" version 2>&1)
    TIME=$(echo "$OUTPUT" | grep "Total startup time" | grep -oE '[0-9]+\.[0-9]+ms' | head -1)
    echo "Run $i: $TIME"

    # Store numeric value for averaging
    NUMERIC=$(echo "$TIME" | tr -d 'ms')
    TIMES+=("$NUMERIC")
done

# Calculate average
SUM=0
for t in "${TIMES[@]}"; do
    SUM=$(echo "$SUM + $t" | bc)
done
AVG=$(echo "scale=3; $SUM / $ITERATIONS" | bc)

echo ""
echo "Average startup time: ${AVG}ms over $ITERATIONS runs"
echo ""

echo "Detailed breakdown from last run:"
echo "=============================================="
AUTO_WORKTREE_PERF=1 "$BINARY" version 2>&1 | tail -40

# Cleanup
rm -f "$BINARY"
