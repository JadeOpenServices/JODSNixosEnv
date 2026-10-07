# w3m terminal browser; inline images go through the kitty graphics protocol.
{ pkgs, ... }:
{
  home.packages = [ pkgs.w3m ];
  home.file.".w3m/config".text = ''
    inline_img_protocol 4
  '';
}
