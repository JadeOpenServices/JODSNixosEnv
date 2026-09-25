{ pkgs, config, ... }:
let
  semanticTheme = import ../../../../themes/lib/semantic.nix { inherit config; };
in
{
  home.packages = with pkgs; [
    television
    nix-search-tv
  ];

  home.file.".config/television/themes/stylix.toml".text = ''
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

  home.file.".config/television/config.toml".source = ./config.toml;
  home.file.".config/television/cable/nix.toml".source = ./nix.toml;
}
