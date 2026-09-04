// Package graphics discovers PCI graphics controllers without a shell.
package graphics

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

type Controller struct {
	BDF      string
	VendorID string
	DeviceID string
	Text     string
}

type Result struct {
	Vendor          string
	Type            string
	Compute         bool
	BusID           string
	IntegratedBusID string
	PassthroughIDs  []string
}

var graphicsLine = regexp.MustCompile(`^([[:xdigit:]]{4}:[[:xdigit:]]{2}:[[:xdigit:]]{2}\.[[:digit:]]) .*?(?:VGA compatible controller|3D controller|Display controller).*?\[([[:xdigit:]]{4}):([[:xdigit:]]{4})\]`)

// Detect executes only lspci with fixed arguments. It never invokes a shell.
func Detect(ctx context.Context) (Result, error) {
	lspci, err := exec.LookPath("lspci")
	if err != nil {
		return Result{}, errors.New("lspci is unavailable")
	}
	output, err := exec.CommandContext(ctx, lspci, "-Dnn").CombinedOutput()
	if err != nil {
		return Result{}, fmt.Errorf("run lspci -Dnn: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return Parse(string(output)), nil
}

func Parse(output string) Result {
	controllers := parseControllers(output)
	result := Result{Vendor: "unknown", Type: "integrated"}
	if len(controllers) == 0 {
		return result
	}

	for _, controller := range controllers {
		if strings.Contains(strings.ToLower(controller.Text), "nvidia") {
			result.Vendor, result.Compute = "nvidia", true
			break
		}
	}
	if result.Vendor == "unknown" {
		for _, controller := range controllers {
			if isAMD(controller.Text) {
				result.Vendor, result.Compute = "amd", true
				break
			}
		}
	}
	if result.Vendor == "unknown" {
		for _, controller := range controllers {
			if strings.Contains(strings.ToLower(controller.Text), "intel") {
				result.Vendor = "intel"
				break
			}
		}
	}

	result.BusID = xorgBusID(controllers[0].BDF)
	integrated := firstIntegrated(controllers)
	if len(controllers) > 1 {
		result.Type = "hybrid"
	} else if integrated == nil {
		result.Type = "dedicated"
	}
	if integrated != nil {
		result.IntegratedBusID = xorgBusID(integrated.BDF)
	}

	ids := make(map[string]struct{})
	for _, controller := range controllers {
		if strings.Contains(strings.ToLower(controller.Text), "intel") {
			continue
		}
		ids[strings.ToLower(controller.VendorID+":"+controller.DeviceID)] = struct{}{}
	}
	for id := range ids {
		result.PassthroughIDs = append(result.PassthroughIDs, id)
	}
	sort.Strings(result.PassthroughIDs)
	return result
}

func parseControllers(output string) []Controller {
	var controllers []Controller
	for _, line := range strings.Split(output, "\n") {
		match := graphicsLine.FindStringSubmatch(line)
		if match != nil {
			controllers = append(controllers, Controller{BDF: match[1], VendorID: match[2], DeviceID: match[3], Text: line})
		}
	}
	return controllers
}

func firstIntegrated(controllers []Controller) *Controller {
	for index := range controllers {
		if strings.Contains(strings.ToLower(controllers[index].Text), "intel") || isAMDIntegrated(controllers[index].Text) {
			return &controllers[index]
		}
	}
	return nil
}

func isAMD(text string) bool {
	lower := strings.ToLower(text)
	return strings.Contains(lower, "amd/ati") || strings.Contains(lower, "advanced micro devices")
}

func isAMDIntegrated(text string) bool {
	lower := strings.ToLower(text)
	if !isAMD(lower) {
		return false
	}
	for _, token := range []string{
		"radeon graphics", "phoenix", "rembrandt", "barcelo", "cezanne", "lucienne", "renoir",
		"mendocino", "van gogh", "raphael", "strix", "hawk point", "hera", "780m", "760m", "740m",
	} {
		if strings.Contains(lower, token) {
			return true
		}
	}
	return false
}

func xorgBusID(bdf string) string {
	parts := strings.FieldsFunc(bdf, func(r rune) bool { return r == ':' || r == '.' })
	if len(parts) != 4 {
		return ""
	}
	bus, busErr := strconv.ParseInt(parts[1], 16, 32)
	device, deviceErr := strconv.ParseInt(parts[2], 16, 32)
	function, functionErr := strconv.ParseInt(parts[3], 10, 32)
	if busErr != nil || deviceErr != nil || functionErr != nil {
		return ""
	}
	return fmt.Sprintf("PCI:%d:%d:%d", bus, device, function)
}
