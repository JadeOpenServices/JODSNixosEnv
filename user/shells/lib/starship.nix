{ ... }:
{
  programs.starship = {
    enable = true;

    settings = {
      format = ''
        [┌─>](bold bright-red) $all[└─>](bold bright-magenta) $character
      '';

      add_newline = true;

      character = {
        error_symbol = "[ ](bold bright-red)";
        success_symbol = "[ ](bold bright-green)";
        vicmd_symbol = "[ ](bold bright-yellow)";
        format = "[󰮯 ](bold bright-green)[|](bold bright-black) ";
      };

      username = {
        show_always = true;
        style_user = "bold bright-cyan";
      };

      hostname = {
        ssh_only = true;
        format = "[$hostname](bold bright-blue) ";
        disabled = false;
      };

      directory = {
        read_only = " ";
        style = "bold bright-green";
      };

      git_branch = {
        symbol = " ";
        style = "bold bright-yellow";
      };

      git_status.style = "bold bright-red";

      nix_shell = {
        symbol = " ";
        style = "bold bright-magenta";
      };

      golang = {
        symbol = " ";
        style = "bold bright-cyan";
      };

      python = {
        symbol = " ";
        style = "bold bright-yellow";
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

      cmd_duration.style = "bold bright-green";

      line_break.disable = true;
    };
  };
}
