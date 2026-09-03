{
  config,
  pkgs,
  lib,
  ...
}:

let
  pluginFiles = builtins.filter (file: lib.hasSuffix ".nix" (toString file)) (
    lib.filesystem.listFilesRecursive ./plugins
  );

  pluginPackages = lib.concatMap (
    file:
    let
      plugin = import file { inherit config pkgs lib; };
    in
    plugin.packages or [ ]
  ) pluginFiles;

  python = pkgs.python3.withPackages (_: pluginPackages);
in
{
  home.packages = [
    python
  ];
}
