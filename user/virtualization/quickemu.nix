{ config, pkgs, settings, ... }:

let
    vmDirectory = "${config.home.homeDirectory}/VMs/fedora";
    vmConfig = "${vmDirectory}/fedora-44-Sway.conf";

    fedoraWaypipe = pkgs.writeShellApplication {
        name = "fedora-waypipe";
        runtimeInputs = with pkgs; [
            coreutils
            openssh
            systemd
            waypipe
        ];
        text = ''
            if ! ssh -o BatchMode=yes -o ConnectTimeout=2 -p 22220 \
                ${settings.username}@localhost true; then
                systemctl --user start quickemu-fedora.service
            fi

            for _ in $(seq 1 120); do
                if ssh -o BatchMode=yes -o ConnectTimeout=2 -p 22220 \
                    ${settings.username}@localhost true; then
                    exec waypipe ssh -p 22220 ${settings.username}@localhost "$@"
                fi
                sleep 1
            done

            printf '%s\n' 'Timed out waiting for the Fedora VM SSH service.' >&2
            exit 1
        '';
    };
in {
    home.packages = with pkgs; [
        quickemu
        waypipe
        swtpm
        fedoraWaypipe
    ];

    systemd.user.services.quickemu-fedora = {
        Unit = {
            Description = "Fedora Sway Quickemu VM";
        };
        Service = {
            Type = "forking";
            WorkingDirectory = vmDirectory;
            ExecStart = "${pkgs.quickemu}/bin/quickemu --vm ${vmConfig} --display none --viewer none";
            PIDFile = "${vmDirectory}/fedora-44-Sway/fedora-44-Sway.pid";
        };
    };

    xdg.desktopEntries.fedora-waypipe-firefox = {
        name = "Firefox (Fedora VM)";
        comment = "Open Firefox from the Fedora VM over Waypipe";
        exec = "${fedoraWaypipe}/bin/fedora-waypipe firefox";
        icon = "firefox";
        categories = [ "Network" "WebBrowser" ];
        terminal = false;
    };

    xdg.desktopEntries.fedora-waypipe-foot = {
        name = "Foot (Fedora VM)";
        comment = "Open Foot from the Fedora VM over Waypipe";
        exec = "${fedoraWaypipe}/bin/fedora-waypipe foot";
        icon = "utilities-terminal";
        categories = [ "System" "TerminalEmulator" ];
        terminal = false;
    };
}
