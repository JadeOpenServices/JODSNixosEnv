
{ config, pkgs, ...}:

{
  home.packages = with pkgs; [
    fastfetch
  ];

  home.file.".config/fastfetch/config.jsonc".text = ''
    {
      "$schema": "https://github.com/fastfetch-cli/fastfetch/wiki/Configuration",
      "modules": ["title", "separator", "os", "kernel", "uptime", "disk"]
    }
  '';
}
