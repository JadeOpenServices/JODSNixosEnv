{ inputs, config, pkgs, lib, settings, ... }: let
    shell = settings.themeDetails.shell or "noctalia";
in {
    imports = [
        ../common/wayland.nix
    ] ++ lib.optional (shell == "noctalia") ../shells/noctalia.nix;

    programs = {
        hyprland = {
            enable = true;
            xwayland.enable = true;
            # package = inputs.hyprland.packages.${pkgs.system}.default;
            # portalPackage = inputs.hyprland.packages.${pkgs.system}.xdg-desktop-portal-hyprland;
            package = pkgs.hyprland;
            portalPackage = pkgs.xdg-desktop-portal-hyprland;
        };
    };

    nix.settings = {
        substituters = ["https://hyprland.cachix.org"];
        trusted-public-keys = ["hyprland.cachix.org-1:a7pgxzMz7+chwVL3/pzj6jIBMioiJM7ypFP8PwtkuGc="];
    };

    xdg.portal = {
        enable = true;
        xdgOpenUsePortal = true;
        config = {
            hyprland.default = ["hyprland"];
        };
    };
}
