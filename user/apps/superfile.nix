{
  inputs,
  config,
  pkgs,
  lib,
  settings,
  ...
}:

let
  preferredEditor = lib.getExe pkgs.${settings.preferredEditor};
in
{
  home.packages = with pkgs; [
    superfile
    glib
    ffmpeg
    poppler-utils
    exiftool
    zoxide
  ];

  home.file.".config/superfile/config.toml".text = ''
    # change your theme
    # Rendered by Noctalia into ~/.config/superfile/theme/noctalia.toml.
    theme = 'noctalia'
    #
    # File editor
    # Uses the configured preferred editor executable from settings.nix.
    editor = "${preferredEditor}"
    #
    # Directory editor
    # Uses the configured preferred editor executable from settings.nix.
    dir_editor = "${preferredEditor}"
    #
    # Auto check for update
    auto_check_update = false
    #
    # Cd on quit
    cd_on_quit = true
    #
    # Whether to open file preview automatically every time superfile is opened.
    default_open_file_preview = true
    show_image_preview = true
    show_panel_footer_info = true
    #
    # The path of the first file panel when superfile is opened. (DON'T USE '~')
    default_directory = "."
    #
    # Display file sizes using powers of 1000 (kB, MB, GB) instead of powers of 1024 (KiB, GiB, etc.).
    file_size_use_si = false
    # Natural sorting keeps numbered names in their expected order.
    default_sort_type = 4
    sort_order_reversed = false
    case_sensitive_sort = false
    #
    # ================   Style =================
    #
    nerdfont = true
    #
    transparent_background = false
    code_previewer = "bat"
    #
    file_preview_width = 0
    enable_file_preview_border = true
    #
    sidebar_width = 20
    #
    border_top = '─'
    border_bottom = '─'
    border_left = '│'
    border_right = '│'
    border_top_left = '╭'
    border_top_right = '╮'
    border_bottom_left = '╰'
    border_bottom_right = '╯'
    border_middle_left = '├'
    border_middle_right = '┤'
    #
    # ==========PLUGINS========== #
    #
    metadata = true
    #
    enable_md5_checksum = false
    zoxide_support = true

    # Superfile's normal open action uses the system default application.
    # Keep common text and source files deterministic: open them in the
    # configured editor instead of handing them to a browser or MIME handler.
    [open_with]
    txt = "${preferredEditor}"
    md = "${preferredEditor}"
    nix = "${preferredEditor}"
    json = "${preferredEditor}"
    toml = "${preferredEditor}"
    yaml = "${preferredEditor}"
    yml = "${preferredEditor}"
    sh = "${preferredEditor}"
    py = "${preferredEditor}"
    js = "${preferredEditor}"
    ts = "${preferredEditor}"
    html = "${preferredEditor}"
    css = "${preferredEditor}"
  '';
}
