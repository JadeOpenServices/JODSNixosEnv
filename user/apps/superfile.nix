{ inputs, config, pkgs, lib, settings, ... }:

let
    preferredEditor = lib.getExe pkgs.${settings.preferredEditor};
in
{
    home.packages = with pkgs; [
        superfile
        glib
    ];

    home.file.".config/superfile/config.toml".text = ''
# change your theme
theme = 'catppuccin'
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
cd_on_quit = false
#
# Whether to open file preview automatically every time superfile is opened.
default_open_file_preview = true
#
# The path of the first file panel when superfile is opened. (DON'T USE '~')
default_directory = "."
#
# Display file sizes using powers of 1000 (kB, MB, GB) instead of powers of 1024 (KiB, GiB, etc.).
file_size_use_si = false
#
# ================   Style =================
#
nerdfont = true
#
transparent_background = true
#
file_preview_width = 0
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
metadata = false
#
enable_md5_checksum = false
    '';
}
