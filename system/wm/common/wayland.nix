{
  inputs,
  pkgs,
  settings,
  ...
}:

{
  environment.systemPackages = with pkgs; [
    wayland
    wl-clipboard
    bibata-cursors
  ];

  services.xserver = {
    enable = true;

    xkb = {
      variant = settings.keyboardVariant;
      layout = settings.keyboardLayout;
      options = "grp:win_space_toggle";
    };
  };

  programs.noctalia-greeter = {
    enable = true;
    package = pkgs.callPackage ../../../pkgs/noctalia-greeter {
      noctalia-greeter = inputs.noctalia-greeter.packages.${pkgs.stdenv.hostPlatform.system}.default;
    };

    settings = {
      appearance = {
        scheme = "Synced";
        theme_mode = "dark";
        corner_radius_scale = 2.0;
        font_family = settings.themeDetails.font;
        password_style = "random";
        hide_logo = false;
      };

      session.default = "GjallarOS Hyprland";

      cursor = {
        theme = "Bibata-Modern-Classic";
        size = 24;
        path = "${pkgs.bibata-cursors}/share/icons";
      };

      keyboard = {
        layout = settings.keyboardLayout;
        variant = settings.keyboardVariant;
        options = "grp:win_space_toggle";
      };
    };
  };

  services.greetd.greeterManagesPlymouth = true;

  security.polkit.enable = true;
}
