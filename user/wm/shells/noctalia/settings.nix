{
  # ============================================================
  # Noctalia bar
  # ============================================================
  #
  # The bar is intentionally a status/navigation surface.
  # Detailed system controls live in the Control Center.
  #
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

  # ============================================================
  # Control Center
  # ============================================================

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
        type = "weather";
      }
      {
        type = "session";
      }
    ];
  };

  # ============================================================
  # Weather
  # ============================================================

  weather = {
    enabled = true;
  };

  # ============================================================
  # System monitor
  # ============================================================

  system.monitor = {
    enabled = true;
  };

  # ============================================================
  # Widget defaults
  # ============================================================

  widget = {
    launcher = {
      scale = 1.15;
    };

    workspaces = {
      style = "minimal";
      occupied_color = "on_surface";
      scale = 1.1;
    };

    active_window = {
      max_length = 50;
    };

    clock = {
      format = "{:%a %d %b  %H:%M}";
    };

    media = {
      hide_album_art = true;
    };

    volume = {
      show_label = false;
      scroll_step = 2;
    };

    brightness = {
      scroll_step = 5;
    };

    power_profile = {
      show_text = false;
    };

    privacy = {
      show_text = false;
    };

    notifications = {
      show_text = false;
    };

    clipboard = {
      show_text = false;
    };

    battery = {
      show_text = false;
    };

    network = {
      show_text = false;
    };

    bluetooth = {
      show_text = false;
    };

    tray = {
      drawer = true;
    };
  };
}
