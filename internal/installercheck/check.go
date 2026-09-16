// Package installercheck validates repository inputs without changing the host.
package installercheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/bakanura/gjallarOS/internal/hardware/graphics"
)

type Level string

const (
	Error Level = "ERROR"
	Warn  Level = "WARN"
	OK    Level = "OK"
)

type Finding struct {
	Level   Level
	Message string
}

type Report struct{ Findings []Finding }

func (r Report) Failed() bool {
	for _, finding := range r.Findings {
		if finding.Level == Error {
			return true
		}
	}
	return false
}

// ResolveRepository resolves a supplied directory once and requires the two
// repository markers used by the installer. This prevents checks from reading
// arbitrary parent directories after a typo or symlink traversal.
func ResolveRepository(path string) (string, error) {
	if path == "" {
		return "", errors.New("repository path is empty")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve repository path: %w", err)
	}
	root, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return "", fmt.Errorf("resolve repository symlinks: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("repository is not a directory: %s", root)
	}
	for _, marker := range []string{"flake.nix", "scripts/installation/install.sh"} {
		info, err := os.Stat(filepath.Join(root, marker))
		if err != nil || info.IsDir() {
			return "", fmt.Errorf("not a GjallarOS repository (missing %s)", marker)
		}
	}
	return root, nil
}

func Check(ctx context.Context, root string) Report {
	r := Report{}
	preset, presetOK := readJSONObject(&r, filepath.Join(root, "scripts/installation/user_PresetJSON/default.user.config.json"), "default installer preset")
	user, userOK := readJSONObject(&r, filepath.Join(root, "user.config.json"), "user.config.json")
	if presetOK && userOK {
		checkPresetCompatibility(&r, preset, user)
		validatePresetSchema(&r, user)
	}

	checkRequiredFiles(&r, root)
	checkNativeGraphics(ctx, &r, root)
	checkRegistration(&r, root)
	return r
}

type presetValueType uint8

const (
	presetString presetValueType = iota
	presetBool
	presetStringList
)

var presetSchema = map[string]presetValueType{
	"system": presetString, "profile": presetString, "hostname": presetString, "username": presetString,
	"timezone": presetString, "locale": presetString, "keyboardLayout": presetString, "keyboardVariant": presetString,
	"touchpadWorkspaceSwipe": presetBool, "clamshellEnable": presetBool, "usbguardEnable": presetBool,
	"allowUnvalidatedODDCModel": presetBool,
	"unattendedInstall":         presetBool,
	"name":                      presetString, "email": presetString, "githubUsername": presetString, "dotfilesDir": presetString,
	"shell": presetString, "editors": presetStringList, "browsers": presetStringList,
	"preferredEditor": presetString, "preferredBrowser": presetString, "theme": presetString,
	"weatherCity": presetString, "weatherCountry": presetString,
	"planeEnable": presetBool, "planeHost": presetString, "drawioEnable": presetBool, "drawioSelfHosted": presetBool, "drawioHost": presetString,
	"backgroundNormal": presetString, "backgroundWork": presetString, "backgroundGaming": presetString,
	"workUserEnable": presetBool, "workUsername": presetString, "workUserPasswordFile": presetString,
	"dockerEnable": presetBool, "debugFunctions": presetBool, "aiEnable": presetBool,
	"overrideAiSelection": presetBool, "overrideModelWith": presetString, "aiAgentMode": presetString,
	"jodsFingerprintEnrollmentAllowed": presetBool,
	"enableScrobbling":                 presetBool, "enableLastfm": presetBool, "enableListenbrainz": presetBool,
	"lastfmUsername": presetString, "listenbrainzUsername": presetString,
	"oddcModel":       presetString,
	"deviceSysVendor": presetString, "deviceProductName": presetString, "deviceProductVersion": presetString,
	"deviceBoardVendor": presetString, "deviceBoardName": presetString, "deviceBoardVersion": presetString,
	"nemuEnable": presetBool, "nemuGpuPassthrough": presetBool, "luksTpm2Enable": presetBool,
	"recoveryEnable":          presetBool,
	"recoveryPartitionEnable": presetBool, "jodsPrebootLockEnable": presetBool,
	"secureBootEnable": presetBool, "secureBootPrompt": presetBool, "endpointManagedDevice": presetBool,
	"jodsEndpoint": presetString, "jodsPolicySigningPublicKey": presetString,
	"jodsRecoveryCommandSigningPublicKey": presetString, "jodsEnrollmentMode": presetString,
	"jodsAllowInsecureTls": presetBool, "jodsDeviceClass": presetString, "jodsDesktopProfile": presetString,
	"autoReboot": presetBool, "runUpdateChecks": presetBool, "writeConfig": presetBool, "runRebuild": presetBool, "forceRedeploy": presetBool,
}

func validatePresetSchema(r *Report, user map[string]json.RawMessage) {
	for key, raw := range user {
		if installerManagedFields[key] {
			r.Findings = append(r.Findings, Finding{Warn, fmt.Sprintf("%s is installer-managed and should be removed from user.config.json", key)})
			continue
		}
		expected, known := presetSchema[key]
		if !known {
			r.Findings = append(r.Findings, Finding{Warn, fmt.Sprintf("unknown user.config.json field: %s", key)})
			continue
		}
		if !matchesPresetType(raw, expected) {
			r.Findings = append(r.Findings, Finding{Error, fmt.Sprintf("user.config.json field %s has the wrong type", key)})
		}
	}
}

func matchesPresetType(raw json.RawMessage, expected presetValueType) bool {
	switch expected {
	case presetString:
		var value string
		return json.Unmarshal(raw, &value) == nil
	case presetBool:
		var value bool
		return json.Unmarshal(raw, &value) == nil
	case presetStringList:
		var value []string
		return json.Unmarshal(raw, &value) == nil
	default:
		return false
	}
}

func checkNativeGraphics(ctx context.Context, r *Report, root string) {
	live, err := graphics.Detect(ctx)
	if err != nil {
		r.Findings = append(r.Findings, Finding{Warn, fmt.Sprintf("native graphics detection unavailable: %v", err)})
		return
	}
	r.Findings = append(r.Findings, Finding{OK, fmt.Sprintf("native graphics detection: %s (%s), %s", live.Vendor, live.Type, live.BusID)})

	generated, err := readGraphicsSettings(filepath.Join(root, "settings.nix"))
	if err != nil {
		r.Findings = append(r.Findings, Finding{Warn, fmt.Sprintf("generated graphics settings unavailable: %v", err)})
		return
	}
	for key, actual := range map[string]string{
		"graphicsVendor":          live.Vendor,
		"graphicsDeviceId":        live.DeviceID,
		"graphicsType":            live.Type,
		"graphicsCompute":         fmt.Sprintf("%t", live.Compute),
		"graphicsBusId":           live.BusID,
		"graphicsIntegratedBusId": live.IntegratedBusID,
	} {
		if generated[key] != actual {
			r.Findings = append(r.Findings, Finding{Warn, fmt.Sprintf("generated %s is %q; live detection is %q; rerun the installer to regenerate settings.nix", key, generated[key], actual)})
			return
		}
	}
	r.Findings = append(r.Findings, Finding{OK, "generated graphics settings match native detection"})
}

var graphicsSetting = regexp.MustCompile(`(?m)^\s*(graphicsVendor|graphicsDeviceId|graphicsType|graphicsCompute|graphicsBusId|graphicsIntegratedBusId)\s*=\s*(?:"([^"]*)"|(true|false));`)

func readGraphicsSettings(path string) (map[string]string, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	settings := make(map[string]string)
	for _, match := range graphicsSetting.FindAllStringSubmatch(string(contents), -1) {
		settings[match[1]] = match[2] + match[3]
	}
	if len(settings) != 6 {
		return nil, errors.New("required graphics fields are missing")
	}
	return settings, nil
}

func readJSONObject(r *Report, path, label string) (map[string]json.RawMessage, bool) {
	contents, err := os.ReadFile(path)
	if err != nil {
		r.Findings = append(r.Findings, Finding{Error, fmt.Sprintf("%s unreadable: %v", label, err)})
		return nil, false
	}
	var object map[string]json.RawMessage
	decoder := json.NewDecoder(strings.NewReader(string(contents)))
	if err := decoder.Decode(&object); err != nil || object == nil {
		if err == nil {
			err = errors.New("top-level value must be an object")
		}
		r.Findings = append(r.Findings, Finding{Error, fmt.Sprintf("%s is invalid JSON: %v", label, err)})
		return nil, false
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			err = errors.New("additional JSON value")
		}
		r.Findings = append(r.Findings, Finding{Error, fmt.Sprintf("%s contains trailing JSON values", label)})
		return nil, false
	}
	r.Findings = append(r.Findings, Finding{OK, fmt.Sprintf("%s is valid JSON", label)})
	return object, true
}

func checkPresetCompatibility(r *Report, preset, user map[string]json.RawMessage) {
	missing := make([]string, 0)
	managed := make([]string, 0)
	for key := range preset {
		if _, exists := user[key]; !exists {
			if installerManagedFields[key] {
				managed = append(managed, key)
			} else {
				missing = append(missing, key)
			}
		}
	}
	sort.Strings(missing)
	sort.Strings(managed)
	if len(missing) == 0 {
		r.Findings = append(r.Findings, Finding{OK, "user.config.json explicitly sets every preset field"})
	} else {
		r.Findings = append(r.Findings, Finding{Warn, fmt.Sprintf("installer defaults apply to missing preset fields: %s", strings.Join(missing, ", "))})
	}
	if len(managed) != 0 {
		r.Findings = append(r.Findings, Finding{OK, fmt.Sprintf("installer-managed fields are intentionally absent: %s", strings.Join(managed, ", "))})
	}
}

// These values are selected from detected hardware by configure_ai.sh. They
// must not be requested from, or persisted by, the user preset.
var installerManagedFields = map[string]bool{
	"aiModel":               true,
	"aiAccelerationProfile": true,
	"aiContextTokens":       true,
	"aiVramMB":              true,
}

func checkRequiredFiles(r *Report, root string) {
	for _, relative := range []string{
		"scripts/installation/install.sh",
		"cmd/gjallar-installer/main.go",
		"internal/installer/app/app.go",
		"system/apps/ollama.nix",
		"system/tools/commands/default.nix",
	} {
		if info, err := os.Stat(filepath.Join(root, relative)); err != nil || info.IsDir() {
			r.Findings = append(r.Findings, Finding{Error, fmt.Sprintf("required file missing: %s", relative)})
		} else {
			r.Findings = append(r.Findings, Finding{OK, fmt.Sprintf("required file present: %s", relative)})
		}
	}
}

func checkRegistration(r *Report, root string) {
	for _, relative := range []string{"system/tools/commands/default.nix"} {
		contents, err := os.ReadFile(filepath.Join(root, relative))
		if err != nil {
			continue
		}
		if strings.Contains(string(contents), "check-installer") {
			r.Findings = append(r.Findings, Finding{OK, fmt.Sprintf("installer checker registered in %s", relative)})
		} else {
			r.Findings = append(r.Findings, Finding{Warn, fmt.Sprintf("installer checker not registered in %s", relative)})
		}
	}
}
