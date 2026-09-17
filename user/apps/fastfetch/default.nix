{ config, pkgs, ... }:

{
  home.packages = with pkgs; [
    fastfetch
    pokemon-colorscripts
  ];

  home.file.".config/fastfetch/config.jsonc".text = ''
    {
      "$schema": "https://github.com/fastfetch-cli/fastfetch/wiki/Configuration",
      "logo": {
        "type": "command-raw",
        "source": "${pkgs.pokemon-colorscripts}/bin/pokemon-colorscripts -r --no-title"
      },
      "modules": [
        "title",
        "separator",
        "os",
        "host",
        "kernel",
        "uptime",
        "packages",
        "shell",
        "cpu",
        "gpu",
        "memory",
        "disk",
        "separator",
        "colors"
      ]
    }
  '';
}
