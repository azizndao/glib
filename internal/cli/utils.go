package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/azizndao/glib/internal/cli/ui"
	"github.com/azizndao/glib/internal/scanner"
	"github.com/azizndao/glib/internal/validator"
)

// ensureCacheDir creates cache directory with secure permissions
func ensureCacheDir(cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0o700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	return nil
}

// clearCache removes cache directory contents
func clearCache(cacheDir string, verbose bool) error {
	if err := os.RemoveAll(cacheDir); err != nil && !os.IsNotExist(err) {
		if verbose {
			fmt.Println(ui.Warningf("Failed to clear cache: %v", err))
		}
		return err
	}
	if verbose {
		fmt.Println(ui.Infof("Cache cleared"))
	}
	return nil
}

// OutputMode defines the level of output detail
type OutputMode string

const (
	OutputModeDetailed OutputMode = "detailed" // Full verbose output
	OutputModeCompact  OutputMode = "compact"  // Compact dev mode output
	OutputModeSilent   OutputMode = "silent"   // Minimal output
)

// printScanStats prints scan statistics based on output mode
func printScanStats(stats scanner.ScanStats, duration time.Duration, mode OutputMode, isIncremental bool) {
	switch mode {
	case OutputModeDetailed:
		// Detailed stats for generate command - show full breakdown
		fmt.Printf("  %s Files scanned: %d\n", ui.IconBullet, stats.FilesScanned)
		fmt.Printf("  %s Providers: %d\n", ui.IconBullet, stats.Providers)
		fmt.Printf("  %s Controllers: %d\n", ui.IconBullet, stats.Controllers)
		fmt.Printf("  %s Middleware: %d\n", ui.IconBullet, stats.Middleware)
		fmt.Printf("  %s Duration: %dms\n", ui.IconBullet, duration.Milliseconds())

	case OutputModeCompact:
		// Compact stats for dev mode
		fmt.Printf("  %s Scanned: %d providers, %d controllers, %d middleware (%dms)\n",
			ui.IconCheck,
			stats.Providers,
			stats.Controllers,
			stats.Middleware,
			duration.Milliseconds())

	case OutputModeSilent:
		// No output
	}
}

// printValidationStats prints validation statistics based on output mode
func printValidationStats(stats *validator.ValidationStats, duration time.Duration, mode OutputMode) {
	switch mode {
	case OutputModeDetailed:
		if stats != nil {
			fmt.Printf("  %s Validation: %d components (%dms)\n",
				ui.IconCheck,
				stats.ComponentsValidated,
				duration.Milliseconds())
		} else {
			fmt.Printf("  %s Validation passed (%dms)\n", ui.IconCheck, duration.Milliseconds())
		}

	case OutputModeCompact:
		fmt.Printf("  %s Validation passed (%dms)\n", ui.IconCheck, duration.Milliseconds())

	case OutputModeSilent:
		// No output
	}
}

// printScanSummaryTable prints a formatted summary table of scan statistics (for verbose mode)
func printScanSummaryTable(scanStats scanner.ScanStats, valStats *validator.ValidationStats, cacheEnabled bool) {
	fmt.Println(ui.BoldTextf("  Scan Summary:"))
	fmt.Println(ui.Mutedf("  ┌────────────────────────┬──────────────┐"))

	// Components found
	fmt.Printf(ui.Mutedf("  │")+" %-22s "+ui.Mutedf("│")+" %12s "+ui.Mutedf("│")+"\n",
		"Providers", ui.Cyan+fmt.Sprintf("%d", scanStats.Providers)+ui.Reset)
	fmt.Printf(ui.Mutedf("  │")+" %-22s "+ui.Mutedf("│")+" %12s "+ui.Mutedf("│")+"\n",
		"Controllers", ui.Blue+fmt.Sprintf("%d", scanStats.Controllers)+ui.Reset)
	fmt.Printf(ui.Mutedf("  │")+" %-22s "+ui.Mutedf("│")+" %12s "+ui.Mutedf("│")+"\n",
		"Middleware", ui.Yellow+fmt.Sprintf("%d", scanStats.Middleware)+ui.Reset)
	fmt.Printf(ui.Mutedf("  │")+" %-22s "+ui.Mutedf("│")+" %12s "+ui.Mutedf("│")+"\n",
		"Handlers", ui.Green+fmt.Sprintf("%d", scanStats.Handlers)+ui.Reset)

	if cacheEnabled && scanStats.FilesScanned > 0 {
		fmt.Println(ui.Mutedf("  ├────────────────────────┼──────────────┤"))

		hitRate := float64(scanStats.CacheHits) * 100 / float64(scanStats.FilesScanned)
		fmt.Printf(ui.Mutedf("  │")+" %-22s "+ui.Mutedf("│")+" %12d "+ui.Mutedf("│")+"\n",
			"Files Scanned", scanStats.FilesScanned)

		cacheHitStr := fmt.Sprintf("%d", scanStats.CacheHits)
		fmt.Printf(ui.Mutedf("  │")+" %-22s "+ui.Mutedf("│")+" "+ui.Green+"%-12s"+ui.Reset+" "+ui.Mutedf("│")+"\n",
			"Cache Hits", fmt.Sprintf("%s (%.1f%%)", cacheHitStr, hitRate))

		fmt.Printf(ui.Mutedf("  │")+" %-22s "+ui.Mutedf("│")+" "+ui.Yellow+"%-12d"+ui.Reset+" "+ui.Mutedf("│")+"\n",
			"Cache Misses", scanStats.CacheMisses)
	}

	if valStats != nil && cacheEnabled {
		fmt.Println(ui.Mutedf("  ├────────────────────────┼──────────────┤"))

		valHitRate := float64(valStats.CacheHits) * 100 / float64(valStats.ComponentsValidated)
		fmt.Printf(ui.Mutedf("  │")+" %-22s "+ui.Mutedf("│")+" %12d "+ui.Mutedf("│")+"\n",
			"Components Validated", valStats.ComponentsValidated)

		valCacheStr := fmt.Sprintf("%d", valStats.CacheHits)
		fmt.Printf(ui.Mutedf("  │")+" %-22s "+ui.Mutedf("│")+" "+ui.Green+"%-12s"+ui.Reset+" "+ui.Mutedf("│")+"\n",
			"Validation Cached", fmt.Sprintf("%s (%.1f%%)", valCacheStr, valHitRate))
	}

	fmt.Println(ui.Mutedf("  └────────────────────────┴──────────────┘"))
}

// printSeparator prints a visual separator
func printSeparator() {
	fmt.Println(ui.Gray + "─────────────────────────────────────────" + ui.Reset)
}
