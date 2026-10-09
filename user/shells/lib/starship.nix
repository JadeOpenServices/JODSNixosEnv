{ ... }:
{
  # Two-line prompt framed on the left: user, path and repo state on top,
  # input below. The icons need a Nerd Font, which every theme ships.
  programs.starship = {
    enable = true;
    settings = {
      add_newline = true;
      format = "[╭─](bold bright-black)$username$hostname$directory$git_branch$git_state$git_status$nix_shell$python$golang$rust$lua$c$cmd_duration$line_break[╰─](bold bright-black)$character";

      username = {
        show_always = true;
        format = "[ $user]($style) ";
        style_user = "bold bright-cyan";
        style_root = "bold bright-red";
      };
      hostname = {
        ssh_only = true;
        format = "[@$hostname]($style) ";
        style = "bold bright-blue";
      };
      directory = {
        format = "[ $path]($style)[$read_only]($read_only_style) ";
        style = "bold bright-green";
        truncation_length = 4;
        truncate_to_repo = true;
        read_only = " ";
      };
      git_branch = {
        format = "on [$symbol$branch]($style) ";
        symbol = " ";
        style = "bold bright-yellow";
      };
      git_status.style = "bold bright-red";
      nix_shell = {
        format = "via [$symbol$name]($style) ";
        symbol = " ";
        style = "bold bright-magenta";
      };
      python = {
        symbol = " ";
        style = "bold bright-yellow";
      };
      golang = {
        symbol = " ";
        style = "bold bright-cyan";
      };
      rust = {
        symbol = " ";
        style = "bold bright-red";
      };
      lua = {
        symbol = " ";
        style = "bold bright-blue";
      };
      c = {
        symbol = " ";
        style = "bold bright-magenta";
      };
      cmd_duration = {
        min_time = 2000;
        format = "took [ $duration]($style) ";
        style = "bold bright-green";
      };
      character = {
        success_symbol = "[❯](bold bright-green)";
        error_symbol = "[❯](bold bright-red)";
        vimcmd_symbol = "[❮](bold bright-yellow)";
      };
    };
  };
}
