// gjallarctl is the safe, non-interactive control tool for GjallarOS.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/bakanura/gjallarOS/internal/ai/profile"
	"github.com/bakanura/gjallarOS/internal/hardware/graphics"
	"github.com/bakanura/gjallarOS/internal/input/xkb"
	"github.com/bakanura/gjallarOS/internal/installercheck"
	"github.com/bakanura/gjallarOS/internal/preset"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" {
		printUsage(stdout)
		return 0
	}

	switch args[0] {
	case "check":
		return runCheck(args[1:], stdout, stderr)
	case "detect":
		return runDetect(args[1:], stdout, stderr)
	case "ai":
		return runAI(args[1:], stdout, stderr)
	case "normalize":
		return runNormalize(args[1:], stdout, stderr)
	case "preset":
		return runPreset(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "ERROR: unknown command %q\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runPreset(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || (args[0] != "validate" && args[0] != "get" && args[0] != "list" && args[0] != "bool") {
		fmt.Fprintln(stderr, "Usage: gjallarctl preset {validate|get|list|bool} --config PATH [--key NAME]")
		return 2
	}
	flags := flag.NewFlagSet("gjallarctl preset "+args[0], flag.ContinueOnError)
	flags.SetOutput(stderr)
	config := flags.String("config", "user.config.json", "preset JSON path")
	key := flags.String("key", "", "preset field name")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 || (args[0] != "validate" && *key == "") {
		return 2
	}
	document, err := preset.Load(*config)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	switch args[0] {
	case "validate":
		return 0
	case "get":
		value, err := document.String(*key)
		if err == nil {
			fmt.Fprintln(stdout, value)
		}
		return presetResult(err, stderr)
	case "list":
		values, err := document.Strings(*key)
		if err == nil {
			for _, value := range values {
				fmt.Fprintln(stdout, value)
			}
		}
		return presetResult(err, stderr)
	case "bool":
		value, err := document.Bool(*key)
		if err == nil {
			fmt.Fprintln(stdout, value)
		}
		return presetResult(err, stderr)
	default:
		return 2
	}
}

func presetResult(err error, stderr io.Writer) int {
	if err == nil {
		return 0
	}
	fmt.Fprintf(stderr, "ERROR: %v\n", err)
	return 1
}

func runNormalize(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "keyboard" {
		fmt.Fprintln(stderr, "Usage: gjallarctl normalize keyboard --layout VALUE")
		return 2
	}
	flags := flag.NewFlagSet("gjallarctl normalize keyboard", flag.ContinueOnError)
	flags.SetOutput(stderr)
	layout := flags.String("layout", "", "XKB keyboard layout")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 {
		return 2
	}
	result, err := xkb.Normalize(*layout)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "layout=%s\nvariant=%s\n", result.Name, result.Variant)
	return 0
}

func runAI(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 || args[0] != "profile" {
		fmt.Fprintln(stderr, "Usage: gjallarctl ai profile [--config PATH]")
		return 2
	}
	flags := flag.NewFlagSet("gjallarctl ai profile", flag.ContinueOnError)
	flags.SetOutput(stderr)
	config := flags.String("config", "user.config.json", "user configuration path")
	if err := flags.Parse(args[1:]); err != nil || flags.NArg() != 0 {
		return 2
	}
	result, err := profile.Detect(context.Background(), *config)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: AI profile detection failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "profile=%s\nmodel=%s\ncontext_tokens=%d\nvram_mb=%d\nram_gb=%d\ngpu_vendor=%s\ngpu_type=%s\n",
		result.Profile, result.Model, result.ContextTokens, result.VRAMMB, result.RAMGB, result.GPUVendor, result.GPUType)
	return 0
}

func runDetect(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 || args[0] != "graphics" {
		fmt.Fprintln(stderr, "Usage: gjallarctl detect graphics")
		return 2
	}
	result, err := graphics.Detect(context.Background())
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: graphics detection failed: %v\n", err)
		return 1
	}
	fmt.Fprintf(stdout, "vendor=%s\ntype=%s\ncompute=%t\nbus=%s\nintegrated_bus=%s\npassthrough_ids=%s\n",
		result.Vendor, result.Type, result.Compute, result.BusID, result.IntegratedBusID,
		strings.Join(result.PassthroughIDs, " "))
	return 0
}

func runCheck(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("gjallarctl check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repo := flags.String("repo", ".", "GjallarOS repository root")
	timeout := flags.Duration("timeout", 5*time.Second, "maximum time for each safe external check")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "ERROR: check accepts no positional arguments")
		return 2
	}
	if *timeout <= 0 || *timeout > 30*time.Second {
		fmt.Fprintln(stderr, "ERROR: --timeout must be greater than zero and no more than 30s")
		return 2
	}

	root, err := installercheck.ResolveRepository(*repo)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	report := installercheck.Check(ctx, root)
	for _, finding := range report.Findings {
		fmt.Fprintf(stdout, "%s: %s\n", finding.Level, finding.Message)
	}
	if report.Failed() {
		fmt.Fprintln(stderr, "ERROR: GjallarOS validation failed")
		return 1
	}
	return 0
}

func printUsage(out io.Writer) {
	fmt.Fprintln(out, "Usage: gjallarctl check [--repo PATH] [--timeout DURATION]\n       gjallarctl detect graphics\n       gjallarctl ai profile [--config PATH]\n       gjallarctl normalize keyboard --layout VALUE\n       gjallarctl preset {validate|get|list|bool} --config PATH [--key NAME]")
	fmt.Fprintln(out, "\nSafe GjallarOS maintenance commands. No command uses sudo.")
}
