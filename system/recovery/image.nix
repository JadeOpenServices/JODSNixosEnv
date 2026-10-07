{
  config,
  lib,
  pkgs,
  modulesPath,
  releaseVersion,
  settings,
  repoSource,
  sourceRevision,
  ...
}:
let
  tools = import ./tools.nix { inherit config pkgs sourceRevision; };

  getSetting =
    name: fallback: if builtins.hasAttr name settings then builtins.getAttr name settings else fallback;

  sourceUserConfig = repoSource + "/user.config.json";

  webApplicationPresetFields =
    if settings ? webApplications then
      {
        webApplications = getSetting "webApplications" [ ];
      }
    else
      {
        # Preserve pre-GJAL-95 generated-state semantics. Absence of the
        # canonical field tells config normalization to migrate legacy state.
        planeEnable = getSetting "planeEnable" false;
        planeHost = getSetting "planeHost" "";
        drawioEnable = getSetting "drawioEnable" false;
        drawioSelfHosted = getSetting "drawioSelfHosted" false;
        drawioHost = getSetting "drawioHost" "";
      };

  generatedInstallerPresetData = {
      system = getSetting "system" pkgs.stdenv.hostPlatform.system;
      hostname = getSetting "hostname" "gjallarOS";
      username = getSetting "username" "user";

      timezone = getSetting "timezone" "UTC";
      locale = getSetting "locale" "en_US.UTF-8";
      keyboardLayout = getSetting "keyboardLayout" "us";
      keyboardVariant = getSetting "keyboardVariant" "";

      touchpadWorkspaceSwipe = getSetting "touchpadWorkspaceSwipe" true;
      clamshellEnable = getSetting "clamshellEnable" true;
      usbguardEnable = getSetting "usbguardEnable" false;
      usbTrustEnforce = getSetting "usbTrustEnforce" false;
      usbTrustTpmHandle = getSetting "usbTrustTpmHandle" "";

      name = getSetting "name" "";
      email = getSetting "email" "";
      githubUsername = getSetting "githubUsername" "";

      dotfilesDir = "";

      apps = settings.apps;
      debugFunctions = getSetting "debugFunctions" false;

      shell = getSetting "shell" "zsh";
      editors = getSetting "editors" [ "vscodium" ];
      browsers = getSetting "browsers" [ "librewolf" ];
      preferredEditor = getSetting "preferredEditor" "vscodium";
      preferredBrowser = getSetting "preferredBrowser" "librewolf";

      theme = getSetting "theme" "noctalia";
      backgroundNormal = getSetting "backgroundNormal" "";


      aiAgentMode = getSetting "aiAgentMode" "workspace";
      overrideAiSelection = false;
      overrideModelWith = "";

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
    // webApplicationPresetFields;

  generatedInstallerPreset = pkgs.writeText "gjallar-installer-fallback.json" (
    builtins.toJSON generatedInstallerPresetData
  );

  installerFallbackPreset =
    if builtins.pathExists sourceUserConfig then sourceUserConfig else generatedInstallerPreset;

  # The recovery partition starts this UKI, not the ISO's GRUB: GRUB needs
  # shim under Secure Boot (e2e-fw13, 2026-10-07: shim_lock protocol not
  # found), and the ISO's /iso mount wants an iso9660 device. findiso= makes
  # stage 1 mount the ISO file install-partition.sh copies next to it.
  recoveryPartitionUki = pkgs.runCommand "gjallar-recovery-partition.efi" { } ''
    ${pkgs.buildPackages.systemdUkify}/lib/systemd/ukify build \
      --linux=${config.boot.kernelPackages.kernel}/${config.system.boot.loader.kernelFile} \
      --initrd=${config.system.build.initialRamdisk}/${config.system.boot.loader.initrdFile} \
      --cmdline="init=${config.system.build.toplevel}/init ${toString config.boot.kernelParams} findiso=/gjallar-recovery.iso" \
      --stub=${pkgs.systemd}/lib/systemd/boot/efi/linux${pkgs.stdenv.hostPlatform.efiArch}.efi.stub \
      --uname=${config.boot.kernelPackages.kernel.modDirVersion} \
      --os-release=@${config.system.build.etc}/etc/os-release \
      --output=$out
  '';
in
{
  imports = [ "${modulesPath}/installer/cd-dvd/installation-cd-minimal.nix" ];

  services.usbguard.enable = lib.mkForce false;

  image.fileName = lib.mkForce "gjallar-recovery-${config.system.nixos.label}-${pkgs.stdenv.hostPlatform.system}.iso";
  isoImage.squashfsCompression = "zstd -Xcompression-level 15";
  isoImage.contents = [
    {
      source = recoveryPartitionUki;
      target = "/EFI/gjallar/recovery-partition.efi";
    }
  ];
  # findiso= runs only in the scripted stage 1.
  boot.initrd.systemd.enable = lib.mkForce false;
  boot.zfs.forceImportRoot = false;

  services.xserver.xkb = {
    layout = settings.keyboardLayout;
    variant = settings.keyboardVariant;
  };

  console =
    if settings.keyboardVariant == "" then
      {
        keyMap = settings.keyboardLayout;
      }
    else
      {
        useXkbConfig = true;
      };

  networking = {
    hostName = "gjallar-recovery";
    networkmanager.enable = true;
    firewall.enable = true;
  };
  services.openssh.enable = lib.mkForce false;
  programs.ssh.startAgent = false;

  services.fprintd.enable = true;

  systemd.services.installer-repository = {
    description = "Prepare writable installer repository";
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

      ${pkgs.coreutils}/bin/rm -f "$target/user.config.json"
      ${pkgs.coreutils}/bin/chown -R nixos:users "$target"
      ${pkgs.coreutils}/bin/chmod -R u+rwX "$target"

      test -f "$target/flake.nix"
      test -f "$target/scripts/installation/install.sh"
      test -f "$target/cmd/gjallar-installer/main.go"
    '';
  };

  systemd.services.installer-device-probe = {
    description = "Collect installer hardware state";
    wantedBy = [ "multi-user.target" ];

    after = [ "installer-repository.service" ];
    requires = [ "installer-repository.service" ];

    path = with pkgs; [
      pciutils
      usbutils
      util-linux
      libinput
      iio-sensor-proxy
      fprintd
      bolt
      alsa-utils
      pipewire
      fwupd
      ethtool
      iw
      systemd
    ];

    serviceConfig = {
      Type = "oneshot";
      NoNewPrivileges = true;
      PrivateTmp = true;
      ProtectHome = true;
      ProtectSystem = "strict";
      ReadWritePaths = [ "/run/gjallarOS" ];
      UMask = "0022";
    };

    script = ''
      set -eu

      ${tools.gjallarctl}/bin/gjallarctl         device-probe refresh         --output /run/gjallarOS/device-probe.json
    '';
  };

  services.udev.extraRules = ''
    ACTION=="add|remove|change", SUBSYSTEM=="pci", RUN+="${pkgs.systemd}/bin/systemctl --no-block start installer-device-probe.service"
    ACTION=="add|remove|change", SUBSYSTEM=="usb", RUN+="${pkgs.systemd}/bin/systemctl --no-block start installer-device-probe.service"
    ACTION=="add|remove|change", SUBSYSTEM=="input", RUN+="${pkgs.systemd}/bin/systemctl --no-block start installer-device-probe.service"
    ACTION=="add|remove|change", SUBSYSTEM=="iio", RUN+="${pkgs.systemd}/bin/systemctl --no-block start installer-device-probe.service"
    ACTION=="add|remove|change", SUBSYSTEM=="thunderbolt", RUN+="${pkgs.systemd}/bin/systemctl --no-block start installer-device-probe.service"
  '';

  environment.etc."systemd/system-sleep/installer-device-probe-refresh" = {
    mode = "0755";
    text = ''
      #!/bin/sh
      if [ "$1" = post ]; then
        ${pkgs.systemd}/bin/systemctl --no-block start installer-device-probe.service
      fi
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
    tools.installer
    tools.recovery
    tools.recoveryExecutor
  ];

  nix.settings = {
    experimental-features = [
      "nix-command"
      "flakes"
    ];
    require-sigs = true;
  };

  environment.etc."gjallar/recovery-image".text = ''
    GjallarOS minimal recovery and installer environment.
    Verify the signed release manifest before copying this image to recovery media.
    Network access is opt-in; SSH is disabled.
  '';

  systemd.tmpfiles.rules = [
    "d /run/gjallarOS/device-config 0700 root root - -"
  ];

  environment.etc."gjallar/installer-fallback-user.config.json".source = installerFallbackPreset;

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
