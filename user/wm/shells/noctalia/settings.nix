{
  bar.default = {
    position = "top";

    start = [
      "launcher"
      "workspaces"
      "active_window"
    ];

    center = [
      "clock"
    ];

    end = [
      "network"
      "bluetooth"
      "privacy"
      "brightness"
      "volume"
      "power_profile"
      "battery"
      "notifications"
      "clipboard"
      "screenshot"
      "control-center"
      "session"
    ];

    dead_zone = {
      actions = {
        left = "panel-toggle launcher";
        right = "panel-toggle control-center";
        scroll_up = "volume-up 2";
        scroll_down = "volume-down 2";
      };
    };
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
        type = "system";
      }
      {
        type = "session";
      }
    ];
  };


  plugins = {
    enabled = [
    ];
  };

  weather = {
    enabled = true;
  };


  system.monitor = {
    enabled = true;
  };


  widget = {
    launcher = {
      scale = 1.15;
    };

    workspaces = {
      occupied_color = "on_surface";
      scale = 1.1;
    };

    clock = {
      format = "{:%a %d %b  %H:%M}";
    };

    media = {
      hide_album_art = true;
    };

    volume = {
      show_label = false;
    };

    brightness = {
    };

    power_profile = {
    };

    privacy = {
    };

    notifications = {
    };

    clipboard = {
    };

    battery = {
    };

    network = {
    };

    bluetooth = {
    };

    tray = {
      drawer = true;
    };
  };
}
