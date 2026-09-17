{
  pkgs,
  inputs,
  ...
}:
{
  environment.systemPackages = with pkgs; [
    sops
    age
  ];

  imports = [ inputs.sops-nix.nixosModules.sops ];

  sops.defaultSopsFormat = "yaml";
  sops.age.generateKey = false;
}
