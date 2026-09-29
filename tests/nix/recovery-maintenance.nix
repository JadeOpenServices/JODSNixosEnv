{
  nixpkgs,
  system,
  recoveryModule,
  # Trees scanned for systemd-ask-password calls feeding cryptsetup.
  sources,
}:
let
  pkgs = nixpkgs.legacyPackages.${system};
  evaluated = nixpkgs.lib.nixosSystem {
    inherit system;
    modules = [
      recoveryModule
      {
        system.stateVersion = "26.05";
        fileSystems."/" = {
          device = "/dev/mapper/cryptroot";
          fsType = "btrfs";
        };
        boot.initrd.luks.devices.cryptroot.device = "/dev/disk/by-partlabel/root";
      }
    ];
    specialArgs = {
      sourceRevision = "test";
      settings = {
        recoveryEnable = true;
        jodsPrebootLockEnable = false;
        endpointManagedDevice = false;
        secureBootEnable = false;
        luksTpm2Enable = false;
      };
    };
  };
  # specialisation.<name>.configuration is the evaluated child config.
  maintenance = evaluated.config.specialisation.gjallar-recovery-maintenance.configuration;
  script = maintenance.boot.initrd.systemd.services.recovery-storage-setup.script;
in
# The maintenance initrd found no root when the credential kept the newline
# systemd-ask-password prints by default: cryptsetup --key-file uses every
# byte of the file (e2e-full, 2026-09-29).
assert pkgs.lib.hasInfix "systemd-ask-password --timeout=0 --newline=no" script;
assert pkgs.lib.hasInfix "try_candidate cryptroot /dev/disk/by-partlabel/root" script;
pkgs.runCommand "gjallar-recovery-maintenance-check"
  {
    nativeBuildInputs = [ pkgs.cryptsetup ];
  }
  ''
    # The contract the fix relies on, checked against real cryptsetup.
    truncate -s 32M disk.img
    printf 'secret' > key
    printf 'secret\n' > key-newline
    cryptsetup luksFormat --batch-mode --disable-locks --type luks2 \
      --pbkdf pbkdf2 --pbkdf-force-iterations 1000 --key-file key disk.img
    cryptsetup open --disable-locks --test-passphrase --key-file key disk.img
    if cryptsetup open --disable-locks --test-passphrase --key-file key-newline disk.img; then
      echo "cryptsetup accepted a key file with a trailing newline" >&2
      exit 1
    fi

    # Same bug in every prompt that feeds cryptsetup or systemd-cryptenroll:
    # each systemd-ask-password call must pass --newline=no.
    sources=(${pkgs.lib.escapeShellArgs (map (source: "${source}") sources)})
    for source in "''${sources[@]}"; do
      test -d "$source"
    done
    if grep -rn 'systemd-ask-password' "''${sources[@]}" |
      grep -vE '^[^:]+:[0-9]+:[[:space:]]*#' |
      grep -v -- '--newline=no'
    then
      echo "systemd-ask-password without --newline=no (see above)" >&2
      exit 1
    fi
    touch "$out"
  ''
