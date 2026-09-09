#!/usr/bin/env python3

from pathlib import Path
import re

repo = Path(__file__).resolve().parents[2]
config_go = repo / "internal/installer/config/config.go"
image_nix = repo / "system/recovery/image.nix"

config_text = config_go.read_text()
image_text = image_nix.read_text()

accepted = set(
    re.findall(r'`json:"([^"]+)"`', config_text)
)

start = image_text.find(
    'generatedInstallerPreset = pkgs.writeText "gjallar-installer-fallback.json"'
)
if start < 0:
    raise RuntimeError("generated installer fallback missing")

end = image_text.find(
    "# Preserve the device's actual user configuration",
    start,
)
if end < 0:
    raise RuntimeError("generated installer fallback end missing")

block = image_text[start:end]

generated = set(
    re.findall(r'^\s{6}([A-Za-z][A-Za-z0-9]*)\s*=', block, re.MULTILINE)
)

unknown = sorted(generated - accepted)

if unknown:
    raise RuntimeError(
        "generated recovery fallback contains fields not accepted by "
        f"config.User: {', '.join(unknown)}"
    )

for forbidden in [
    "aiModel",
    "aiAccelerationProfile",
    "aiContextTokens",
    "aiVramMb",
    "graphicsVendor",
    "graphicsType",
    "graphicsCompute",
    "graphicsBusID",
    "graphicsIntegratedBusID",
    "wifiDriver",
]:
    if re.search(rf'^\s{{6}}{re.escape(forbidden)}\s*=', block, re.MULTILINE):
        raise RuntimeError(
            f"derived rendered field leaked into installer preset: {forbidden}"
        )

print("PASS: every generated fallback field is accepted by config.User")
print("PASS: rendered/hardware-derived fields are absent")
