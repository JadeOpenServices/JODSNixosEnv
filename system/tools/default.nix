{ lib, pkgs, settings, ... }:
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
    usbguardReviewModule = { pkgs, ... }: let
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
    in {
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
    homeDiagnosticsModule = { ... }: let
        sessionDiagnostic = pkgs.writeShellScript "gjallar-hyprland-session-diagnostics" ''
            set -u
            state_dir="$HOME/.local/state/gjallar-diagnostics"
            ${coreutils}/mkdir -p "$state_dir"
            exec > >(${coreutils}/tee "$state_dir/hyprland-session.log") 2>&1
            log() { printf 'gjallar-hyprland-session-diagnostics: %s\n' "$*"; }
            log 'begin'
            ${coreutils}/date --iso-8601=seconds
            ${coreutils}/id
            ${pkgs.systemd}/bin/systemctl --user show hyprland-session.target --property=ActiveState --property=SubState --property=Result --no-pager || true
            ${coreutils}/printenv | ${pkgs.gnugrep}/bin/grep -E '^(PATH|XDG_|WAYLAND_DISPLAY|HYPRLAND_INSTANCE_SIGNATURE|DBUS_SESSION_BUS_ADDRESS)=' || true
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
            ${pkgs.procps}/bin/pgrep -a -u "$USER" 'Hyprland|fuzzel|ags' || true
            log 'end'
        '';
    in {
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
in {
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

    systemd.services.plymouth-quit-wait.enable = false;


    systemd.services.noctalia-greeter-plymouth = {
      description = "Wait for Noctalia Greeter before quitting Plymouth";
      wantedBy = [ "graphical.target" ];
      after = [ "greetd.service" ];
      wants = [ "greetd.service" ];
      serviceConfig = {
        Type = "oneshot";
        ExecStart = pkgs.writeShellScript "wait-for-noctalia-greeter" ''
          for i in $(seq 1 600); do
            if ${pkgs.procps}/bin/pgrep -f noctalia-greeter-compositor >/dev/null; then
              ${pkgs.plymouth}/bin/plymouth quit --wait || true
              exit 0
            fi
            ${pkgs.coreutils}/bin/sleep 0.1
          done
          ${pkgs.plymouth}/bin/plymouth quit --wait || true
          exit 0
        '';
      };
    };

    programs.nh = {
        enable = true;
        clean.enable = false;
        flake = settings.dotfilesDir;
    };
    environment.systemPackages = with pkgs; [
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
      after = [ "NetworkManager.service" "display-manager.service" "home-manager-${settings.username}.service" ];
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
