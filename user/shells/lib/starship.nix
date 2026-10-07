{ ... }:
{
  # Two-line prompt: context on the first line, input on the second.
  programs.starship = {
    enable = true;
    settings = {
      add_newline = true;
      format = "$username$hostname$directory$git_branch$git_state$git_status$nix_shell$python$golang$rust$cmd_duration$line_break$character";

      username = {
        show_always = true;
        format = "[$user]($style) ";
        style_user = "bold cyan";
      };
      hostname = {
        ssh_only = true;
        format = "@[$hostname]($style) ";
        style = "bold blue";
      };
      directory = {
        style = "bold green";
        truncation_length = 4;
        read_only = " ro";
      };
      git_branch = {
        format = "[$symbol$branch]($style) ";
        symbol = "git:";
        style = "bold yellow";
      };
      git_status.style = "bold red";
      nix_shell = {
        format = "[nix:$name]($style) ";
        style = "bold purple";
      };
      python.format = "[py:$version]($style) ";
      golang.format = "[go:$version]($style) ";
      rust.format = "[rs:$version]($style) ";
      cmd_duration = {
        min_time = 2000;
        format = "took [$duration]($style) ";
        style = "bold green";
      };
      character = {
        success_symbol = "[>](bold green)";
        error_symbol = "[>](bold red)";
        vimcmd_symbol = "[<](bold yellow)";
      };
    };
  };
}
