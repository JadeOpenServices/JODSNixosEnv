{ pkgs }:
let
  threshold = "4.5";

  guard = pkgs.writeShellApplication {
    name = "gjallar-theme-contrast";

    runtimeInputs = [
      pkgs.gawk
    ];

    text = ''
      threshold=4.5

      if [ "''${1:-}" = "--threshold" ]; then
        threshold="''${2:-}"
        shift 2
      fi

      if [ "$#" -eq 0 ] || [ $(( $# % 3 )) -ne 0 ]; then
        echo "usage: gjallar-theme-contrast [--threshold RATIO] NAME BACKGROUND FOREGROUND [...]" >&2
        exit 2
      fi

      failed=0

      while [ "$#" -gt 0 ]; do
        name="$1"
        background="$2"
        foreground="$3"
        shift 3

        if [[ ! "$background" =~ ^#[0-9A-Fa-f]{6}$ ]] \
          || [[ ! "$foreground" =~ ^#[0-9A-Fa-f]{6}$ ]]
        then
          printf 'FAIL: theme contrast %-18s invalid colors: %s / %s\n' \
            "$name" "$background" "$foreground" >&2
          failed=1
          continue
        fi

        ratio="$(
          gawk \
            -v background="$background" \
            -v foreground="$foreground" '
              function channel(hex, offset, value) {
                value = strtonum("0x" substr(hex, offset, 2)) / 255

                if (value <= 0.04045)
                  return value / 12.92

                return ((value + 0.055) / 1.055) ^ 2.4
              }

              function luminance(hex) {
                return \
                  0.2126 * channel(hex, 2) + \
                  0.7152 * channel(hex, 4) + \
                  0.0722 * channel(hex, 6)
              }

              BEGIN {
                a = luminance(background)
                b = luminance(foreground)

                high = a > b ? a : b
                low = a > b ? b : a

                printf "%.4f", (high + 0.05) / (low + 0.05)
              }
            '
        )"

        if gawk \
          -v ratio="$ratio" \
          -v threshold="$threshold" \
          'BEGIN { exit !(ratio >= threshold) }'
        then
          printf 'PASS: theme contrast %-18s %s:1\n' "$name" "$ratio"
        else
          printf 'FAIL: theme contrast %-18s %s:1 below %s:1 (%s / %s)\n' \
            "$name" "$ratio" "$threshold" "$background" "$foreground" >&2
          failed=1
        fi
      done

      exit "$failed"
    '';
  };

  pair =
    name: background: foreground:
    ''${name} "{{ colors.${background}.default.hex }}" "{{ colors.${foreground}.default.hex }}"'';
in
{
  inherit guard threshold;

  # One contrast contract for every Noctalia-driven application adapter.
  #
  # 4.5:1 is the WCAG normal-text threshold. A failed pre-hook prevents the
  # application theme output from being replaced by an unsafe palette.
  preHook = builtins.concatStringsSep " " (
    [
      "${guard}/bin/gjallar-theme-contrast"
      "--threshold"
      threshold
    ]
    ++ [
      (pair "surface" "surface" "on_surface")
      (pair "surface-variant" "surface_variant" "on_surface_variant")
      (pair "primary" "primary" "on_primary")
      (pair "secondary" "secondary" "on_secondary")
      (pair "tertiary" "tertiary" "on_tertiary")
      (pair "error" "error" "on_error")
      (pair "hover" "surface_container_high" "on_surface")
      (pair "selection" "primary" "on_primary")
    ]
  );
}
