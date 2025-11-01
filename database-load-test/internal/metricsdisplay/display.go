package metricsdisplay

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/AntonChubarov/surrealdb-exporter-utils/database-load-test/internal/metrics"
)

// DisplayConfig provides metrics collection configuration
type DisplayConfig interface {
	MetricsDisplayInterval() time.Duration
}

type SnapshotSource interface {
	// GetSnapshot returns a snapshot of current metrics
	GetSnapshot() metrics.Snapshot
}

// Display handles displaying metrics to the console
type Display struct {
	collector SnapshotSource
	cfg       DisplayConfig
}

// New creates a new metrics display
func New(collector SnapshotSource, cfg DisplayConfig) *Display {
	return &Display{
		collector: collector,
		cfg:       cfg,
	}
}

// Start begins displaying metrics at configured intervals
func (d *Display) Start(ctx context.Context) {
	ticker := time.NewTicker(d.cfg.MetricsDisplayInterval())
	defer ticker.Stop()

	fmt.Println("=== Load Test Started ===")
	fmt.Println()

	for {
		select {
		case <-ctx.Done():
			d.displayFinalReport()
			return
		case <-ticker.C:
			d.displaySnapshot()
		}
	}
}

func (d *Display) displaySnapshot() {
	snapshot := d.collector.GetSnapshot()

	// Calculate totals from current step
	totals := calculateTotals(snapshot.CurrentStep)

	fmt.Println("--- Metrics Snapshot ---")
	fmt.Printf("Time: %s\n", snapshot.Timestamp.Format("15:04:05"))
	fmt.Printf("Current Step Operations: %d (Success: %d, Failed: %d)\n",
		totals.count, totals.success, totals.failure)
	if totals.count > 0 {
		successRate := float64(totals.success) / float64(totals.count) * 100
		fmt.Printf("Success Rate: %.2f%%\n", successRate)
		fmt.Printf("Average Duration: %v\n", totals.avgDuration)
	}
	fmt.Println()

	if len(snapshot.CurrentStep) > 0 {
		fmt.Println("Operations by Type (Current Step):")
		d.displayStatsTable(snapshot.CurrentStep, false)
	}
	fmt.Println()
}

func (d *Display) displayFinalReport() {
	snapshot := d.collector.GetSnapshot()

	// Calculate totals from overall stats
	totals := calculateTotals(snapshot.Overall)

	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║           FINAL LOAD TEST REPORT                         ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Summary Section
	fmt.Println("SUMMARY:")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("  Total Operations:  %d\n", totals.count)
	if totals.count > 0 {
		fmt.Printf("  Successful:        %d (%.2f%%)\n",
			totals.success,
			float64(totals.success)/float64(totals.count)*100)
		fmt.Printf("  Failed:            %d (%.2f%%)\n",
			totals.failure,
			float64(totals.failure)/float64(totals.count)*100)
		fmt.Printf("  Average Duration:  %v\n", totals.avgDuration)
	}
	fmt.Println()

	if len(snapshot.Overall) > 0 {
		fmt.Println("DETAILED STATISTICS BY OPERATION:")
		fmt.Println(strings.Repeat("─", 60))
		d.displayStatsTable(snapshot.Overall, true)
		fmt.Println()

		// Show comparison if current step has data
		if len(snapshot.CurrentStep) > 0 {
			fmt.Println("CURRENT STEP vs OVERALL COMPARISON:")
			fmt.Println(strings.Repeat("─", 60))
			d.displayComparison(snapshot.CurrentStep, snapshot.Overall)
			fmt.Println()
		}
	}

	fmt.Println("=== Load Test Completed ===")
	fmt.Println()
}

// displayStatsTable shows statistics in a formatted table
func (d *Display) displayStatsTable(statsMap map[string]metrics.Stats, detailed bool) {
	if len(statsMap) == 0 {
		fmt.Println("  (no data)")
		return
	}

	// Get sorted operation types for consistent output
	opTypes := make([]string, 0, len(statsMap))
	for opType := range statsMap {
		opTypes = append(opTypes, opType)
	}
	sort.Strings(opTypes)

	if detailed {
		// Detailed view with percentiles
		fmt.Printf("  %-20s %8s %9s %10s %10s %10s %10s\n",
			"Operation", "Count", "Success%", "Avg", "Median", "P95", "Max")
		fmt.Println("  " + strings.Repeat("─", 80))

		for _, opType := range opTypes {
			stats := statsMap[opType]
			if stats.Count == 0 {
				continue
			}
			successRate := float64(stats.SuccessCount) / float64(stats.Count) * 100
			fmt.Printf("  %-20s %8d %8.1f%% %10v %10v %10v %10v\n",
				truncate(opType, 20),
				stats.Count,
				successRate,
				stats.Average,
				stats.Median,
				stats.P95,
				stats.MaxDuration)
		}
	} else {
		// Compact view for snapshots
		fmt.Printf("  %-20s %8s %9s %10s %10s\n",
			"Operation", "Count", "Success%", "Avg", "P95")
		fmt.Println("  " + strings.Repeat("─", 60))

		for _, opType := range opTypes {
			stats := statsMap[opType]
			if stats.Count == 0 {
				continue
			}
			successRate := float64(stats.SuccessCount) / float64(stats.Count) * 100
			fmt.Printf("  %-20s %8d %8.1f%% %10v %10v\n",
				truncate(opType, 20),
				stats.Count,
				successRate,
				stats.Average,
				stats.P95)
		}
	}
}

// displayComparison shows current step vs overall metrics side by side
func (d *Display) displayComparison(current, overall map[string]metrics.Stats) {
	// Get all unique operation types
	opTypeSet := make(map[string]bool)
	for op := range current {
		opTypeSet[op] = true
	}
	for op := range overall {
		opTypeSet[op] = true
	}

	opTypes := make([]string, 0, len(opTypeSet))
	for op := range opTypeSet {
		opTypes = append(opTypes, op)
	}
	sort.Strings(opTypes)

	fmt.Printf("  %-20s | %-25s | %-25s\n", "Operation", "Current Step", "Overall")
	fmt.Println("  " + strings.Repeat("─", 75))

	for _, opType := range opTypes {
		currentStats := current[opType]
		overallStats := overall[opType]

		currentStr := "-"
		if currentStats.Count > 0 {
			currentStr = fmt.Sprintf("%d ops, P95: %v",
				currentStats.Count, currentStats.P95)
		}

		overallStr := "-"
		if overallStats.Count > 0 {
			overallStr = fmt.Sprintf("%d ops, P95: %v",
				overallStats.Count, overallStats.P95)
		}

		fmt.Printf("  %-20s | %-25s | %-25s\n",
			truncate(opType, 20), currentStr, overallStr)
	}
}

// aggregateTotals holds aggregated metrics across all operation types
type aggregateTotals struct {
	count       int64
	success     int64
	failure     int64
	avgDuration time.Duration
}

// calculateTotals aggregates metrics from all operations in the map
func calculateTotals(statsMap map[string]metrics.Stats) aggregateTotals {
	totals := aggregateTotals{}

	var totalDuration time.Duration
	for _, stats := range statsMap {
		totals.count += stats.Count
		totals.success += stats.SuccessCount
		totals.failure += stats.FailureCount
		// Weight each operation's average by its count
		totalDuration += stats.Average * time.Duration(stats.Count)
	}

	if totals.count > 0 {
		totals.avgDuration = totalDuration / time.Duration(totals.count)
	}

	return totals
}

// truncate truncates a string to the specified length
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
