package metricsdisplay

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/AntonChubarov/surrealdb-exporter-utils/database-load-test/internal/metrics"

	// Pretty tables & text styling
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// DisplayConfig provides metrics collection configuration
// (do not change per user request)
type DisplayConfig interface {
	MetricsDisplayInterval() time.Duration
}

// SnapshotSource is the provider of metrics snapshots
// (do not change per user request)
type SnapshotSource interface {
	// GetSnapshot returns a snapshot of current metrics
	GetSnapshot() metrics.Snapshot
}

// Display handles displaying metrics to the terminal in a compact, pretty layout.
// It periodically refreshes the screen with a dashboard-like view using ANSI control
// sequences and "go-pretty" tables.
type Display struct {
	collector SnapshotSource
	cfg       DisplayConfig

	// internal state for nicer UX
	start time.Time
}

// New creates a new metrics display
func New(collector SnapshotSource, cfg DisplayConfig) *Display {
	return &Display{
		collector: collector,
		cfg:       cfg,
		start:     time.Now(),
	}
}

// Start begins displaying metrics at configured intervals. It clears and redraws the
// terminal on each tick, creating a live dashboard effect. When the context is cancelled,
// a final (non-clearing) report is printed.
func (d *Display) Start(ctx context.Context) {
	ticker := time.NewTicker(d.cfg.MetricsDisplayInterval())
	defer ticker.Stop()

	// Initial banner
	fmt.Println("=== Load Test Started ===")
	fmt.Println()

	for {
		select {
		case <-ctx.Done():
			// Print a final static report without clearing the screen
			d.displayFinalReport()
			return
		case <-ticker.C:
			// Live dashboard refresh
			clearScreen()
			d.displaySnapshot()
		}
	}
}

// displaySnapshot renders a compact dashboard using the current snapshot only (and a short header).
func (d *Display) displaySnapshot() {
	snapshot := d.collector.GetSnapshot()

	// Calculate totals from current step
	totals := calculateTotals(snapshot.CurrentStep)
	uptime := time.Since(d.start).Truncate(time.Second)

	// Header (single line, compact)
	header := table.NewWriter()
	header.SetOutputMirror(os.Stdout)
	header.AppendHeader(table.Row{"TIME", "UPTIME", "OPS", "SUCCESS", "FAIL", "SUCCESS%", "AVG"})
	header.AppendRow(table.Row{
		snapshot.Timestamp.Format("15:04:05"),
		uptime,
		formatInt64(totals.count),
		formatInt64(totals.success),
		formatInt64(totals.failure),
		formatPct(rate(totals.success, totals.count)),
		formatDur(totals.avgDuration),
	})
	header.SetStyle(compactStyle())
	header.Render()
	fmt.Println()

	// Current step by operation (sorted by Count desc)
	if len(snapshot.CurrentStep) > 0 {
		fmt.Println(text.FgHiCyan.Sprint("Operations (Current Step)"))
		d.renderStatsTable(snapshot.CurrentStep, false /*detailed*/)
		fmt.Println()
	}

	// Overall (cumulative) for context
	if len(snapshot.Overall) > 0 {
		fmt.Println(text.FgHiMagenta.Sprint("Operations (Overall)"))
		d.renderStatsTable(snapshot.Overall, true /*detailed*/)
	}
}

// displayFinalReport prints a static final report with both overall stats and a comparison
// between the last step and overall percentiles.
func (d *Display) displayFinalReport() {
	snapshot := d.collector.GetSnapshot()
	totals := calculateTotals(snapshot.Overall)

	fmt.Println()
	fmt.Println("╔═══════════════════════════════════════════════════════════╗")
	fmt.Println("║                 FINAL LOAD TEST REPORT                    ║")
	fmt.Println("╚═══════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Summary line
	summary := table.NewWriter()
	summary.SetOutputMirror(os.Stdout)
	summary.AppendHeader(table.Row{"TOTAL OPS", "SUCCESS", "FAIL", "SUCCESS%", "AVG"})
	summary.AppendRow(table.Row{
		formatInt64(totals.count),
		formatInt64(totals.success),
		formatInt64(totals.failure),
		formatPct(rate(totals.success, totals.count)),
		formatDur(totals.avgDuration),
	})
	summary.SetStyle(compactStyle())
	summary.Render()
	fmt.Println()

	if len(snapshot.Overall) > 0 {
		fmt.Println(text.FgHiMagenta.Sprint("Detailed Statistics by Operation (Overall)"))
		d.renderStatsTable(snapshot.Overall, true)
		fmt.Println()

		if len(snapshot.CurrentStep) > 0 {
			fmt.Println(text.FgHiCyan.Sprint("Current Step vs Overall (P95 focus)"))
			d.renderComparison(snapshot.CurrentStep, snapshot.Overall)
			fmt.Println()
		}
	}

	fmt.Println("=== Load Test Completed ===")
	fmt.Println()
}

// renderStatsTable shows statistics in a formatted table.
// If detailed is true: Count, Success%, Avg, Median, P90, P95, P99, Max
// Else (compact): Count, Success%, Avg, P95
func (d *Display) renderStatsTable(statsMap map[string]metrics.Stats, detailed bool) {
	if len(statsMap) == 0 {
		fmt.Println("  (no data)")
		return
	}

	// Sorted by Count desc, then name asc
	opTypes := make([]string, 0, len(statsMap))
	for k := range statsMap {
		opTypes = append(opTypes, k)
	}
	sort.Slice(opTypes, func(i, j int) bool {
		si, sj := statsMap[opTypes[i]], statsMap[opTypes[j]]
		if si.Count == sj.Count {
			return opTypes[i] < opTypes[j]
		}
		return si.Count > sj.Count
	})

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	if detailed {
		t.AppendHeader(table.Row{"OP", "COUNT", "SUCCESS%", "AVG", "MED", "P90", "P95", "P99", "MAX"})
	} else {
		t.AppendHeader(table.Row{"OP", "COUNT", "SUCCESS%", "AVG", "P95"})
	}

	for _, op := range opTypes {
		s := statsMap[op]
		if s.Count == 0 {
			continue
		}
		if detailed {
			t.AppendRow(table.Row{
				ellipsis(op, 24),
				formatInt64(s.Count),
				formatPct(rate(s.SuccessCount, s.Count)),
				formatDur(s.Average),
				formatDur(s.Median),
				formatDur(s.P90),
				formatDur(s.P95),
				formatDur(s.P99),
				formatDur(s.MaxDuration),
			})
		} else {
			t.AppendRow(table.Row{
				ellipsis(op, 24),
				formatInt64(s.Count),
				formatPct(rate(s.SuccessCount, s.Count)),
				formatDur(s.Average),
				formatDur(s.P95),
			})
		}
	}

	// Compact, borderless-lean style
	t.SetStyle(compactStyle())
	t.SortBy([]table.SortBy{{Name: "COUNT", Mode: table.DscNumeric}})
	t.Render()
}

// renderComparison shows current step vs overall metrics side by side (compact),
// emphasizing P95 comparison and counts.
func (d *Display) renderComparison(current, overall map[string]metrics.Stats) {
	// Collect unique operations
	ops := make(map[string]struct{})
	for k := range current {
		ops[k] = struct{}{}
	}
	for k := range overall {
		ops[k] = struct{}{}
	}
	keys := make([]string, 0, len(ops))
	for k := range ops {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"OP", "CURR: COUNT", "CURR: P95", "OVERALL: COUNT", "OVERALL: P95"})
	for _, k := range keys {
		cs := current[k]
		os := overall[k]
		var cCount, oCount string
		var cP95, oP95 string
		if cs.Count > 0 {
			cCount = formatInt64(cs.Count)
			cP95 = formatDur(cs.P95)
		} else {
			cCount, cP95 = "-", "-"
		}
		if os.Count > 0 {
			oCount = formatInt64(os.Count)
			oP95 = formatDur(os.P95)
		} else {
			oCount, oP95 = "-", "-"
		}
		t.AppendRow(table.Row{ellipsis(k, 24), cCount, cP95, oCount, oP95})
	}
	t.SetStyle(compactStyle())
	t.Render()
}

// ===== Helpers =====

// calculateTotals aggregates metrics from all operations in the map
// (kept identical logically to original behavior)
type aggregateTotals struct {
	count       int64
	success     int64
	failure     int64
	avgDuration time.Duration
}

func calculateTotals(statsMap map[string]metrics.Stats) aggregateTotals {
	totals := aggregateTotals{}
	var totalDuration time.Duration
	for _, s := range statsMap {
		totals.count += s.Count
		totals.success += s.SuccessCount
		totals.failure += s.FailureCount
		totalDuration += s.Average * time.Duration(s.Count) // weighted
	}
	if totals.count > 0 {
		totals.avgDuration = totalDuration / time.Duration(totals.count)
	}
	return totals
}

func rate(num, den int64) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den) * 100
}

// compactStyle returns a lean go-pretty style suited for fast-refresh dashboards.
func compactStyle() table.Style {
	st := table.StyleLight
	st.Options = table.Options{
		DrawBorder:      true,
		SeparateColumns: true,
		SeparateHeader:  true,
		SeparateRows:    false,
	}
	st.Format.Header = text.FormatTitle
	st.Color = table.ColorOptions{ // neutral colors for wide compatibility
		Header: text.Colors{text.FgHiWhite, text.Bold},
	}
	return st
}

// clearScreen performs an ANSI clear + cursor-home. This is widely supported
// on Unix terminals and on Windows 10+ terminals. If unsupported, it simply
// results in raw characters (harmless).
func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

// format helpers keep numbers/durations compact and readable in a terminal
func formatInt64(n int64) string { return fmt.Sprintf("%d", n) }

func formatPct(p float64) string { return fmt.Sprintf("%.1f%%", p) }

// formatDur renders time.Duration in a compact unit (ns/µs/ms/s) with up to 3 sig figs.
func formatDur(d time.Duration) string {
	if d <= 0 {
		return "-"
	}
	ns := float64(d.Nanoseconds())
	switch {
	case ns < 1_000:
		return fmt.Sprintf("%dns", int64(ns))
	case ns < 1_000_000:
		return fmt.Sprintf("%.2fµs", ns/1_000)
	case ns < 1_000_000_000:
		return fmt.Sprintf("%.2fms", ns/1_000_000)
	default:
		return fmt.Sprintf("%.2fs", ns/1_000_000_000)
	}
}

// ellipsis truncates a string to max characters (>= 3) adding "..." when needed.
func ellipsis(s string, max int) string {
	if max <= 3 || len(s) <= max {
		if max <= 0 {
			return s
		}
		return s
	}
	if len(s) > max {
		return s[:max-3] + "..."
	}
	return s
}
