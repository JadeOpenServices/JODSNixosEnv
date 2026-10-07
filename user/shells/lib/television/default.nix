# television fuzzy finder: ctrl-t completes the command line, ctrl-r
# searches history (both from the packaged zsh integration), and a `nix`
# channel searches packages and options through nix-search-tv.
{ pkgs, config, ... }:
let
  semanticTheme = import ../../../../themes/lib/semantic.nix { inherit config; };
in
{
  programs.television = {
    enable = true;
    enableZshIntegration = true;
    settings.ui = {
      theme = "stylix";
      input_bar_position = "top";
      show_help_bar = false;
    };
    channels.nix = {
      metadata = {
        name = "nix";
        description = "NixOS packages and options";
        requirements = [ "nix-search-tv" ];
      };
      source.command = "nix-search-tv print";
      preview.command = "nix-search-tv preview {}";
    };
  };
  home.packages = [ pkgs.nix-search-tv ];

  xdg.configFile."television/themes/stylix.toml".text = ''
    remote_control_mode_bg = '#00000000'
    border_fg = '#${semanticTheme.fallback.outline}'
    text_fg = '#${semanticTheme.fallback.onSurface}'
    dimmed_text_fg = '#${semanticTheme.fallback.primary}'
    input_text_fg = '#${semanticTheme.fallback.error}'
    result_count_fg = '#${semanticTheme.fallback.error}'
    result_name_fg = '#${semanticTheme.fallback.primary}'
    result_line_number_fg = '#${semanticTheme.fallback.secondary}'
    result_value_fg = '#${semanticTheme.fallback.terminal.bright.white}'
    selection_fg = '#${semanticTheme.fallback.terminal.normal.green}'
    selection_bg = '#${semanticTheme.fallback.surfaceContainerHigh}'
    match_fg = '#${semanticTheme.fallback.error}'
    preview_title_fg = '#${semanticTheme.fallback.terminal.extended.orange}'
    channel_mode_fg = '#${semanticTheme.fallback.terminal.extended.light}'
    remote_control_mode_fg = '#${semanticTheme.fallback.terminal.normal.green}'
    send_to_channel_mode_fg = '#${semanticTheme.fallback.primary}'
  '';
}
