{ pkgs, lib, settings, config, ... }: let
    details = settings.themeDetails;
in {
    # Comfortable inner spacing and rounded controls for GTK dialogs such as
    # the installer's Zenity windows.
    environment.etc."gtk-3.0/gtk.css".text = ''
        dialog {
            padding: 18px;
            background-color: #1e1e2e;
            color: #cdd6f4;
        }
        dialog box,
        dialog grid {
            padding: 8px;
            border-spacing: 10px;
        }
        dialog entry,
        dialog button,
        dialog treeview,
        dialog combobox {
            padding: 8px 12px;
            border-radius: 12px;
        }
        dialog button {
            margin: 4px;
            min-height: 34px;
            padding: 8px 18px;
            color: #cdd6f4;
            background-color: #313244;
            border: 1px solid #585b70;
            border-radius: 10px;
            box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.10),
                inset 0 -2px 0 rgba(17, 17, 27, 0.35),
                0 2px 4px rgba(17, 17, 27, 0.45);
        }
        dialog button:hover {
            background-color: #45475a;
            border-color: #89b4fa;
        }
        dialog button:active,
        dialog button:checked {
            background-color: #585b70;
            box-shadow: inset 0 2px 3px rgba(17, 17, 27, 0.50);
        }
        /* Explicit generic selectors keep Zenity's inner widgets consistent
           even when a theme engine changes its widget hierarchy. */
        window entry,
        window textview,
        window treeview,
        window combobox {
            background-color: #181825;
            color: #cdd6f4;
            border: 2px solid #45475a;
            border-radius: 12px;
            padding: 10px 14px;
        }
        window button {
            color: #11111b;
            background-color: #89b4fa;
            border: 2px solid #cdd6f4;
            border-radius: 12px;
            padding: 10px 22px;
            margin: 6px;
            box-shadow: inset 0 2px 0 rgba(255, 255, 255, 0.55),
                inset 0 -3px 0 rgba(17, 17, 27, 0.45),
                0 3px 6px rgba(17, 17, 27, 0.60);
        }
        button {
            background-image: none;
        }
        entry, textview, treeview, combobox {
            background-image: none;
        }
        /* Zenity creates some controls outside the dialog selector; keep all
           inner surfaces on the same Catppuccin palette. */
        window.dialog,
        window.dialog box,
        window.dialog grid,
        window.dialog scrolledwindow,
        window.dialog viewport {
            background-color: #1e1e2e;
            color: #cdd6f4;
        }
        window.dialog entry,
        window.dialog treeview,
        window.dialog combobox,
        window.dialog spinbutton {
            color: #cdd6f4;
            background-color: #181825;
            border: 1px solid #45475a;
            border-radius: 10px;
            padding: 9px 12px;
        }
        window.dialog button {
            color: #11111b;
            background-image: linear-gradient(to bottom, #b4befe, #89b4fa);
            border: 1px solid #cdd6f4;
            border-radius: 10px;
            box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.55),
                inset 0 -2px 0 rgba(30, 30, 46, 0.35),
                0 2px 5px rgba(17, 17, 27, 0.55);
        }
        window.dialog button:hover {
            background-image: linear-gradient(to bottom, #cba6f7, #b4befe);
        }
        window.dialog button:active,
        window.dialog button:checked {
            background-image: linear-gradient(to bottom, #89b4fa, #74c7ec);
            box-shadow: inset 0 2px 4px rgba(17, 17, 27, 0.55);
        }
    '';
    stylix = {
        enable = true;
        polarity = "dark";
        image = details.wallpaper.center;
        base16Scheme = lib.mkIf (details.themeName != null)
            "${pkgs.base16-schemes}/share/themes/${details.themeName}.yaml";
        override = lib.mkIf (details.override != null) details.override;
        opacity = {
            terminal = details.opacity;
            applications = details.opacity;
            desktop = details.opacity;
            popups = details.opacity;
        };

        cursor = {
            size = 32;
            name = "phinger-cursors-light";
            package = pkgs.phinger-cursors;
        };

        fonts = {
            sansSerif = {
                package = details.fontPkg;
                name = details.font;
            };
            serif = config.stylix.fonts.sansSerif;
            monospace = config.stylix.fonts.sansSerif;
            emoji = config.stylix.fonts.sansSerif;
        };
    };
}
