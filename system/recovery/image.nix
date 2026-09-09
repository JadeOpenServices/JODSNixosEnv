{ config, lib, pkgs, modulesPath, releaseVersion, settings, repoSource, ... }:
let
  tools = import ./tools.nix { inherit config pkgs; };

  getSetting = name: fallback:
    if builtins.hasAttr name settings
    then builtins.getAttr name settings
    else fallback;

  sourceUserConfig = repoSource + "/user.config.json";

  generatedInstallerPreset = pkgs.writeText "gjallar-installer-fallback.json" (
    builtins.toJSON {
      # IMPORTANT:
      # This object is config.User input, NOT rendered settings.nix output.
      # Do not add derived fields such as aiModel, graphicsVendor,
      # aiContextTokens, wifiDriver, etc. Those are rediscovered/rendered
      # later by the installer.
      system = getSetting "system" pkgs.stdenv.hostPlatform.system;
      profile = getSetting "profile" "laptop";
      hostname = getSetting "hostname" "gjallarOS";
      username = getSetting "username" "user";

      timezone = getSetting "timezone" "UTC";
      locale = getSetting "locale" "en_US.UTF-8";
      keyboardLayout = getSetting "keyboardLayout" "us";
      keyboardVariant = getSetting "keyboardVariant" "";

      touchpadWorkspaceSwipe = getSetting "touchpadWorkspaceSwipe" true;
      clamshellEnable = getSetting "clamshellEnable" true;
      usbguardEnable = getSetting "usbguardEnable" false;

      name = getSetting "name" "";
      email = getSetting "email" "";
      githubUsername = getSetting "githubUsername" "";

      dotfilesDir = "";

      workUserEnable = getSetting "workUserEnable" false;
      workUsername = getSetting "workUsername" "";

      dockerEnable = getSetting "dockerEnable" false;
      debugFunctions = getSetting "debugFunctions" false;

      shell = getSetting "shell" "zsh";
      editors = getSetting "editors" [ "vscodium" ];
      browsers = getSetting "browsers" [ "librewolf" ];
      preferredEditor = getSetting "preferredEditor" "vscodium";
      preferredBrowser = getSetting "preferredBrowser" "librewolf";

      planeEnable = getSetting "planeEnable" false;
      planeHost = getSetting "planeHost" "";
      drawioEnable = getSetting "drawioEnable" false;
      drawioSelfHosted = getSetting "drawioSelfHosted" false;
      drawioHost = getSetting "drawioHost" "";

      theme = getSetting "theme" "noctalia";
      backgroundNormal = getSetting "backgroundNormal" "";
      backgroundWork = getSetting "backgroundWork" "";
      backgroundGaming = getSetting "backgroundGaming" "";

      enableScrobbling = getSetting "enableScrobbling" false;
      enableLastfm = getSetting "enableLastfm" false;
      enableListenbrainz = getSetting "enableListenbrainz" false;

      frameworkEnable = getSetting "frameworkEnable" false;
      frameworkModel = getSetting "frameworkModel" "";

      aiEnable = getSetting "aiEnable" false;
      aiAgentMode = getSetting "aiAgentMode" "workspace";
      overrideAiSelection = false;
      overrideModelWith = "";

      nemuEnable = getSetting "nemuEnable" false;
      nemuGpuPassthrough = getSetting "nemuGpuPassthrough" false;

      luksTpm2Enable = getSetting "luksTpm2Enable" false;

      recoveryEnable = getSetting "recoveryEnable" true;
      recoveryPartitionEnable = getSetting "recoveryPartitionEnable" true;

      secureBootEnable = getSetting "secureBootEnable" false;
      secureBootPrompt = false;

      endpointManagedDevice = false;
      jodsPrebootLockEnable = false;

      autoReboot = false;
      runUpdateChecks = false;
      writeConfig = true;
      runRebuild = true;
      unattendedInstall = false;
    }
  );

  # Preserve the device's actual user configuration when one exists in the
  # source used to build the recovery image. This is the normal local fallback.
  installerFallbackPreset =
    if builtins.pathExists sourceUserConfig
    then sourceUserConfig
    else generatedInstallerPreset;
in
{
  imports = [ "${modulesPath}/installer/cd-dvd/installation-cd-minimal.nix" ];

  image.fileName = lib.mkForce "gjallar-recovery-${config.system.nixos.label}-${pkgs.stdenv.hostPlatform.system}.iso";
  isoImage.squashfsCompression = "zstd -Xcompression-level 15";
  boot.zfs.forceImportRoot = false;

  # Installer/recovery media follows the keyboard selected for GjallarOS.
  services.xserver.xkb = {
    layout = settings.keyboardLayout;
    variant = settings.keyboardVariant;
  };

  # A minimal recovery ISO primarily runs on the Linux virtual console.
  # For ordinary layouts without an XKB variant, set the VC keymap directly
  # so the selected layout is active from the first login prompt onward.
  console =
    if settings.keyboardVariant == ""
    then {
      keyMap = settings.keyboardLayout;
    }
    else {
      useXkbConfig = true;
    };

  networking = {
    hostName = "gjallar-recovery";
    networkmanager.enable = true;
    firewall.enable = true;
  };
  services.openssh.enable = lib.mkForce false;
  programs.ssh.startAgent = false;

  # The installer needs a writable GjallarOS repository because fresh-target
  # hardware configuration and generated installer settings are produced
  # during installation. The flake source itself is immutable in the Nix
  # store, so copy it into ephemeral /run storage at boot.
  systemd.services.gjallar-installer-repository = {
    description = "Prepare writable GjallarOS installer repository";
    wantedBy = [ "multi-user.target" ];
    before = [
      "getty@tty1.service"
      "serial-getty@ttyS0.service"
    ];

    serviceConfig = {
      Type = "oneshot";
      RemainAfterExit = true;
    };

    script = ''
      set -eu

      target=/run/gjallarOS/repo

      ${pkgs.coreutils}/bin/mkdir -p "$target"
      ${pkgs.coreutils}/bin/cp -a ${repoSource}/. "$target"/

      # Configuration authority is resolved immediately before the
      # installer runs. Never inherit user.config.json merely because
      # it happened to be present in the embedded source checkout.
      ${pkgs.coreutils}/bin/rm -f "$target/user.config.json"
      ${pkgs.coreutils}/bin/chown -R nixos:users "$target"
      ${pkgs.coreutils}/bin/chmod -R u+rwX "$target"

      test -f "$target/flake.nix"
      test -f "$target/scripts/installation/install.sh"
      test -f "$target/cmd/gjallar-installer/main.go"
    '';
  };

  environment.systemPackages = with pkgs; [
    btrfs-progs
    dosfstools
    gptfdisk
    parted
    xorriso
    age
    cryptsetup
    curl
    efibootmgr
    gitMinimal
    jq
    nixos-install-tools
    openssl
    sbctl
    systemd
    tpm2-tools
    util-linux
    tools.gjallarctl
    tools.recovery
    tools.recoveryExecutor
  ];

  nix.settings = {
    experimental-features = [ "nix-command" "flakes" ];
    require-sigs = true;
  };

  environment.etc."gjallar/recovery-image".text = ''
    GjallarOS minimal recovery and installer environment.
    Verify the signed release manifest before copying this image to recovery media.
    Network access is opt-in; SSH is disabled.
  '';

  # Sourced from deployment/release-policy.json by the root flake. The
  # recovery image can therefore never silently drift from the main release.
  systemd.tmpfiles.rules = [
    "d /run/gjallarOS/device-config 0700 root root - -"
  ];

  environment.etc."gjallar/installer-fallback-user.config.json".source =
    installerFallbackPreset;

  environment.etc."gjallar/device-config-contract".text = ''
    GjallarOS fresh-install device configuration handoff

    Runtime provider path:
      /run/gjallarOS/device-config/user.config.json

    Embedded fallback:
      /etc/gjallar/installer-fallback-user.config.json

    Resolution order:
      1. Runtime device configuration
      2. Embedded device configuration
      3. Interactive installer fallback

    The runtime provider may later be JODS, local recovery storage, or another
    trusted provisioning component.

    Supplying configuration does not grant storage authority and does not
    bypass target-disk validation, destructive confirmation, LUKS credential
    handling, Secure Boot checks, or recovery lifecycle policy.

    Plaintext passwords, LUKS keys, TPM secrets, private signing keys, and
    enrollment secrets must not be stored in this configuration handoff.
  '';

  system.stateVersion = releaseVersion;
}
