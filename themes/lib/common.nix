{
  pkgs,
  lib,
  settings,
  config,
  ...
}:
let
  details = settings.themeDetails;
  noctaliaPalettePath = builtins.getEnv "GJALLAR_NOCTALIA_PALETTE";
  noctaliaPalette =
    if noctaliaPalettePath != "" && builtins.pathExists noctaliaPalettePath then
      builtins.fromJSON (builtins.readFile noctaliaPalettePath)
    else
      { };
in
{
  stylix = {
    enable = true;
    polarity = "dark";
    base16Scheme = lib.mkIf (
      details.themeName != null
    ) "${pkgs.base16-schemes}/share/themes/${details.themeName}.yaml";
    override =
      let
        configured = if details.override == null then { } else details.override;
      in
      lib.mkIf (configured != { } || noctaliaPalette != { }) (configured // noctaliaPalette);
    opacity = {
      terminal = details.opacity;
      applications = details.opacity;
      desktop = details.opacity;
      popups = details.opacity;
    };

    cursor = {
      size = 32;
      name = "phinger-cursors-light";
      package = pkgs.phinger-cursors;
    };

    fonts = {
      sansSerif = {
        package = details.fontPkg;
        name = details.font;
      };
      serif = config.stylix.fonts.sansSerif;
      monospace = config.stylix.fonts.sansSerif;
      emoji = config.stylix.fonts.sansSerif;
    };
  };
}
