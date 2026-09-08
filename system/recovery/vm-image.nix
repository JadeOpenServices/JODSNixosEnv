{ lib, pkgs, ... }:
{
  boot.kernelParams = [
    "console=tty0"
    "console=ttyS0,115200n8"
  ];

  services.getty.autologinUser = lib.mkForce "nixos";

  # Disposable VM base-install proof only.
  #
  # Production recovery retains the real Secure Boot/TPM/JODS checks.
  # This VM profile tests only:
  #
  #   GPT -> LUKS2 -> Btrfs -> nixos-install -> boot
  #
  # Use the ordinary runtime device-config provider contract so this does
  # not add a special bypass inside the installer itself.
  systemd.services.gjallar-vm-device-config = {
    description = "Prepare VM-only GjallarOS installer device configuration";
    wantedBy = [ "multi-user.target" ];
    after = [ "gjallar-installer-repository.service" ];
    before = [ "getty.target" ];

    serviceConfig = {
      Type = "oneshot";
      RemainAfterExit = true;
    };

    script = ''
      set -eu

      src=/etc/gjallar/installer-fallback-user.config.json
      dst=/run/gjallarOS/device-config/user.config.json

      test -s "$src"
      mkdir -p "$(dirname "$dst")"

      ${pkgs.python3}/bin/python3 - "$src" "$dst" <<'PYJSON'
import json
import os
import sys
import tempfile

src, dst = sys.argv[1], sys.argv[2]

with open(src, "r", encoding="utf-8") as handle:
    config = json.load(handle)

# VM-only scope. Production recovery does not receive these overrides.
config["secureBootEnable"] = False
config["secureBootPrompt"] = False
config["luksTpm2Enable"] = False
config["jodsPrebootLockEnable"] = False
config["endpointManagedDevice"] = False

# This disposable VM exists to prove the complete canonical fresh-install
# storage layout, including the dedicated JODS recovery partition. These
# settings do not alter production recovery-media policy.
config["recoveryEnable"] = True
config["recoveryPartitionEnable"] = True

directory = os.path.dirname(dst)
fd, temporary = tempfile.mkstemp(
    prefix=".user.config.",
    dir=directory,
    text=True,
)

try:
    with os.fdopen(fd, "w", encoding="utf-8") as handle:
        json.dump(config, handle, indent=2, sort_keys=True)
        handle.write("\n")
        handle.flush()
        os.fsync(handle.fileno())

    os.chmod(temporary, 0o600)
    os.replace(temporary, dst)
finally:
    if os.path.exists(temporary):
        os.unlink(temporary)
PYJSON

      test -s "$dst"
    '';
  };

}
