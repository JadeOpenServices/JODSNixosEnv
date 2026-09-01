{
  bar.default = {
    position = "top";

    start = [
      "workspaces"
    ];

    center = [
      "clock"
    ];

    end = [
      "network"
      "bluetooth"
      "volume"
      "battery"
      "weather"
      "control-center"
      "session"
    ];
  };

  control_center = {
    sidebar = "full";

    shortcuts = [
      {
        type = "wifi";
      }
      {
        type = "bluetooth";
      }
      {
        type = "audio";
      }
      {
        type = "weather";
      }
      {
        type = "system";
      }
      {
        type = "session";
      }
    ];
  };

  weather = {
    enabled = true;
  };

  system.monitor = {
    enabled = true;
  };
}
