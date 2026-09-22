{
  settings,
  swayShellDetails,
  swayWorkspaces,
  ...
}:
{
  wayland.windowManager.sway.config = {
    bars = [ ];
    output = { };
    workspaceOutputAssign = swayWorkspaces.assignments;
    startup = map (command: {
      inherit command;
      always = true;
    }) swayShellDetails.launchCommands;
    window.titlebar = false;
    floating.titlebar = true;
    input = {
      "type:keyboard" = {
        xkb_layout = "us,ru";
        xkb_options = "grp:win_space_toggle";
      };
      "type:pointer" = {
        pointer_accel = "-1.0";
      };
    };
  };
}
