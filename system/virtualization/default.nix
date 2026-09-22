{
  lib,
  settings,
  ...
}:
{
  imports =
    lib.optionals settings.nemuEnable [
      ./nemu
      ./spice.nix
    ]
    ++ lib.optional settings.containersEnable ./containers.nix;
}
