{
  lib,
  pkgs,
  settings,
  ...
}:
let
  diagnosticsEnabled = settings.debugFunctions or false;
  systemctl = "${pkgs.systemd}/bin/systemctl";
  systemdEscape = "${pkgs.systemd}/bin/systemd-escape";
  coreutils = "${pkgs.coreutils}/bin";
  bootDiagnostic = pkgs.writeShellScript "gjallar-boot-diagnostics" ''
    set -u
    log_file="/var/lib/gjallar-diagnostics/boot-$(${coreutils}/cat /proc/sys/kernel/random/boot_id).log"
    exec > >(${coreutils}/tee "$log_file") 2>&1

    log() { printf 'gjallar-boot-diagnostics: %s\n' "$*"; }
    report_unit() {
      local unit="$1"
      log "unit=$unit"
      ${systemctl} show "$unit" --property=LoadState --property=ActiveState --property=SubState --property=Result --no-pager || true
      ${systemctl} status "$unit" --no-pager --full || true
    }
    report_home() {
      local home="$1"
      if ${coreutils}/test -L "$home/.config/hypr/hyprland.conf"; then
        log "hyprland-config=$(${coreutils}/readlink -f "$home/.config/hypr/hyprland.conf")"
      else
        log "hyprland-config missing: $home/.config/hypr/hyprland.conf"
      fi
    }

    log 'begin'
    ${coreutils}/date --iso-8601=seconds
    ${coreutils}/uname -a
    ${coreutils}/df -h / /boot || true
    ${systemctl} --failed --no-pager || true
    report_unit NetworkManager.service
    report_unit display-manager.service
    report_unit "home-manager-$(${systemdEscape} -- "${settings.username}").service"
    report_home "/home/${settings.username}"
    ${lib.optionalString settings.workUserEnable ''
      report_unit "home-manager-$(${systemdEscape} -- "${settings.workUsername}").service"
      report_home "/home/${settings.workUsername}"
    ''}
    ${pkgs.networkmanager}/bin/nmcli general status || true
    ${pkgs.networkmanager}/bin/nmcli device status || true
    ${pkgs.systemd}/bin/journalctl -b -p warning..alert --no-pager || true
    log 'end'
  '';
  usbguardReviewModule =
    { pkgs, ... }:
    let
      reviewScript = pkgs.writeShellScript "gjallar-usbguard-review" ''
        set -u
        declare -A prompted=()
        while :; do
          active_ids=' '
          while IFS= read -r line; do
            id="$(${pkgs.gnused}/bin/sed -n 's/^\([0-9][0-9]*\):.*/\1/p' <<< "$line")"
            [ -n "$id" ] || continue
            active_ids+="$id "
            [ -z "''${prompted[$id]+x}" ] || continue
            prompted[$id]=1
            ${pkgs.yad}/bin/yad --question --title='USBGuard approval' \
              --text="A new USB device is blocked:\n\n$line\n\nAllow it?" \
              --width=620 --center \
              --button='Allow once:0' --button='Always allow:2' --button='Keep blocked:1'
            choice="$?"
            case "$choice" in
              0) ${pkgs.usbguard}/bin/usbguard allow-device "$id" || true ;;
              2) ${pkgs.usbguard}/bin/usbguard allow-device --permanent "$id" || true ;;
            esac
          done < <(${pkgs.usbguard}/bin/usbguard list-devices --blocked 2>/dev/null || true)
          for id in "''${!prompted[@]}"; do
            [[ "$active_ids" == *" $id "* ]] || unset 'prompted[$id]'
          done
          ${pkgs.coreutils}/bin/sleep 2
        done
      '';
    in
    {
      systemd.user.services.gjallar-usbguard-review = {
        Unit = {
          Description = "GjallarOS USBGuard approval prompts";
          After = [ "hyprland-session.target" ];
          PartOf = [ "hyprland-session.target" ];
        };
        Service = {
          ExecStart = reviewScript;
          Restart = "on-failure";
          RestartSec = 3;
        };
        Install.WantedBy = [ "hyprland-session.target" ];
      };
    };
  homeDiagnosticsModule =
    { config, ... }:
    let
      sessionDiagnostic = pkgs.writeShellScript "gjallar-hyprland-session-diagnostics" ''
        set -u
        state_dir="$HOME/.local/state/gjallar-diagnostics"
        ${coreutils}/mkdir -p "$state_dir"
        boot_id="$(${coreutils}/cat /proc/sys/kernel/random/boot_id)"
        exec > >(${coreutils}/tee -a "$state_dir/hyprland-$boot_id.log") 2>&1
        log() { printf 'gjallar-hyprland-session-diagnostics: %s\n' "$*"; }
        log 'begin'
        ${coreutils}/date --iso-8601=seconds
        ${coreutils}/id
        ${pkgs.systemd}/bin/systemctl --user show hyprland-session.target --property=ActiveState --property=SubState --property=Result --no-pager || true
        ${coreutils}/printenv | ${pkgs.gnugrep}/bin/grep -E '^(PATH|XDG_|WAYLAND_DISPLAY|HYPRLAND_INSTANCE_SIGNATURE|DBUS_SESSION_BUS_ADDRESS)=' || true
        hyprland_ready=false
        for attempt in $(${coreutils}/seq 1 300); do
          if ${pkgs.hyprland}/bin/hyprctl monitors >/dev/null 2>&1; then
            hyprland_ready=true
            break
          fi
          ${coreutils}/bin/sleep 0.1
        done
        log "hyprland-ipc-ready=$hyprland_ready"
        for command in fuzzel kitty asztal; do
          path="$(${pkgs.findutils}/bin/find "$HOME/.nix-profile/bin" "/etc/profiles/per-user/$USER/bin" -maxdepth 1 -name "$command" \( -type l -o -type f \) -print -quit 2>/dev/null || true)"
          if [ -n "$path" ]; then log "command=$command path=$path"; else log "command=$command missing from user profiles"; fi
        done
        if ${coreutils}/test -L "$HOME/.config/hypr/hyprland.conf"; then
          log "hyprland-config=$(${coreutils}/readlink -f "$HOME/.config/hypr/hyprland.conf")"
        else
          log 'hyprland-config missing'
        fi
        ${pkgs.hyprland}/bin/hyprctl version || true
        ${pkgs.hyprland}/bin/hyprctl configerrors || true
        ${pkgs.hyprland}/bin/hyprctl monitors || true
        ${pkgs.hyprland}/bin/hyprctl clients || true
        ${lib.getExe config.programs.noctalia.package} config validate || true
        ${lib.getExe config.programs.noctalia.package} msg status || true
        ${pkgs.systemd}/bin/journalctl --user -b --no-pager -u gjallar-hyprland-session-diagnostics.service || true
        ${pkgs.procps}/bin/pgrep -f -a -u "$USER" 'Hyprland|noctalia|fuzzel|kitty|swaybg|waybar|ags' || true
        log 'end'
      '';
    in
    {
      systemd.user.services.gjallar-hyprland-session-diagnostics = {
        Unit = {
          Description = "GjallarOS Hyprland session diagnostics";
          After = [ "hyprland-session.target" ];
          PartOf = [ "hyprland-session.target" ];
        };
        Service = {
          Type = "oneshot";
          RemainAfterExit = true;
          ExecStart = sessionDiagnostic;
        };
        Install.WantedBy = [ "hyprland-session.target" ];
      };
    };
in
{
  imports = [ ./scripts/default.nix ];
  config = lib.mkMerge [
    {
      services.usbguard = {
        enable = settings.usbguardEnable or false;
        dbus.enable = settings.usbguardEnable or false;
        IPCAllowedGroups = [ "wheel" ];
      };

      # Replace the noisy kernel/systemd console with Plymouth's graphical
      # spinner. Kernel errors remain visible when a boot actually fails.
      boot = {
        plymouth.enable = true;

        consoleLogLevel = 0;
        initrd.verbose = false;

        kernelParams = [
          "quiet"
          "splash"
          "rd.systemd.show_status=false"
          "systemd.show_status=false"
          "udev.log_level=0"
          "rd.udev.log_level=0"
        ];
      };

      # Noctalia's greetd compositor takes over the DRM device. Keep Plymouth
      # visible until that compositor and its greeter client are both running.
      systemd.services.plymouth-quit.enable = false;
      systemd.services.plymouth-quit-wait.enable = false;

      # Stop Plymouth immediately before greetd launches, retaining its last
      # frame. This releases DRM without the multi-second blocking delay of
      # `plymouth deactivate`; the greeter then overwrites the retained frame.
      systemd.services.greetd.serviceConfig.ExecStartPre = pkgs.writeShellScript "handoff-plymouth-to-greetd" ''
        ${pkgs.plymouth}/bin/plymouth quit --retain-splash || true
      '';

      # Start the shutdown splash before systemd tears down normal services.
      # Stopping the display manager first releases DRM for Plymouth; from
      # that point its shutdown renderer stays up through final poweroff.
      systemd.services.plymouth-poweroff = {
        before = [ "shutdown.target" ];
        conflicts = [ "display-manager.service" ];
      };
      systemd.services.plymouth-reboot = {
        before = [ "shutdown.target" ];
        conflicts = [ "display-manager.service" ];
      };
      systemd.services.plymouth-halt = {
        before = [ "shutdown.target" ];
        conflicts = [ "display-manager.service" ];
      };
      systemd.services.plymouth-kexec = {
        before = [ "shutdown.target" ];
        conflicts = [ "display-manager.service" ];
      };

      programs.nh = {
        enable = true;
        clean.enable = false;
        flake = settings.dotfilesDir;
      };
      environment.systemPackages = with pkgs; [
        gh
        go_1_26
        nixfmt
        python3
        nix-output-monitor
        nvd
        lm_sensors
        procps
        gawk
        findutils
        openssl
        power-profiles-daemon
      ];
    }
    (lib.mkIf (settings.usbguardEnable or false) {
      # Only the primary interactive user receives hardware approval prompts.
      home-manager.users.${settings.username} = usbguardReviewModule;
    })
    (lib.mkIf diagnosticsEnabled {
      home-manager.sharedModules = [ homeDiagnosticsModule ];
      systemd.services.gjallar-boot-diagnostics = {
        description = "GjallarOS boot diagnostics";
        wantedBy = [ "multi-user.target" ];
        after = [
          "NetworkManager.service"
          "display-manager.service"
          "home-manager-${settings.username}.service"
        ];
        serviceConfig = {
          Type = "oneshot";
          ExecStart = bootDiagnostic;
          StandardOutput = "journal+console";
          StandardError = "journal+console";
          StateDirectory = "gjallar-diagnostics";
        };
      };
    })
  ];
}
