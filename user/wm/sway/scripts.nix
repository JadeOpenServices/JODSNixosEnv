{
  pkgs,
  ...
}:

let
  swayWorkspace = pkgs.writeShellApplication {
    name = "sway-workspace";

    runtimeInputs = [
      pkgs.jq
      pkgs.sway
    ];

    text = ''
      usage() {
        cat >&2 <<'EOF'
      Usage:
        sway-workspace focus <1-10>
        sway-workspace move <1-10>
        sway-workspace prev
        sway-workspace next
        sway-workspace move-prev
        sway-workspace move-next
      EOF
        exit 2
      }

      action="$1"

      case "$action" in
        focus|move|prev|next|move-prev|move-next)
          ;;
        *)
          usage
          ;;
      esac

      focused_output="$(
        swaymsg -r -t get_outputs |
          jq -er '.[] | select(.active == true and .focused == true) | .name'
      )"

      offset=-1
      index=0

      while IFS= read -r output; do
        if [ "$output" = "$focused_output" ]; then
          offset=$((index * 10))
          break
        fi
        index=$((index + 1))
      done < <(
        swaymsg -r -t get_outputs |
          jq -r '[.[] | select(.active == true) | .name] | sort[]'
      )

      if [ "$offset" -lt 0 ]; then
        printf 'sway-workspace: focused output "%s" is not active\n' \
          "$focused_output" >&2
        exit 1
      fi

      case "$action" in
        focus|move)
          local_number="''${2:-}"

          case "$local_number" in
            1|2|3|4|5|6|7|8|9|10)
              ;;
            *)
              usage
              ;;
          esac
          ;;

        prev|next|move-prev|move-next)
          current_global="$(
            swaymsg -r -t get_workspaces |
              jq -er '.[] | select(.focused == true) | .num'
          )"

          current_local=$((current_global - offset))

          if [ "$current_local" -lt 1 ] ||
             [ "$current_local" -gt 10 ]; then
            printf >&2 \
              'sway-workspace: workspace number %s does not belong to output "%s"\n' \
              "$current_global" \
              "$focused_output"
            exit 1
          fi

          case "$action" in
            prev|move-prev)
              if [ "$current_local" -eq 1 ]; then
                local_number=10
              else
                local_number=$((current_local - 1))
              fi
              ;;

            next|move-next)
              if [ "$current_local" -eq 10 ]; then
                local_number=1
              else
                local_number=$((current_local + 1))
              fi
              ;;
          esac
          ;;
      esac

      global_number=$((offset + local_number))
      workspace="$global_number:$local_number"

      case "$action" in
        focus|prev|next)
          swaymsg -q -- \
            "workspace --no-auto-back-and-forth $workspace"
          ;;

        move|move-prev|move-next)
          swaymsg -q -- \
            "move container to workspace $workspace"

          swaymsg -q -- \
            "workspace --no-auto-back-and-forth $workspace"
          ;;
      esac
    '';
  };
in
{
  _module.args.swayWorkspaces = {
    command = "${swayWorkspace}/bin/sway-workspace";
    assignments = [ ];
  };

  home.packages = [
    swayWorkspace
  ];
}
