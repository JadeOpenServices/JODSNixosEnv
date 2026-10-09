{ lib, ... }:
{
  # systemd-cryptsetup counts every unlock method against tries= (default
  # 3), not only typed passphrases: the TPM2 token takes the first try, and
  # after a wrong passphrase it retries the same one once without asking. One
  # typo then ended in emergency mode with root locked (e2e-fw13, 2026-10-09,
  # systemd 260). The counter restarts on every boot, so it never limited
  # guessing; keep asking instead.
  options.boot.initrd.luks.devices = lib.mkOption {
    type = lib.types.attrsOf (
      lib.types.submodule {
        config.crypttabExtraOpts = [ "tries=0" ];
      }
    );
  };
}
