{
  config,
  lib,
  pkgs,
  settings,
  ...
}:
let
  dolphinNextcloudIcons = import ../../themes/apps/dolphin/nextcloud-icons.nix;

  enable = settings.nextcloudEnable or false;
  host = settings.nextcloudHost or "";
  preferredBrowser = settings.preferredBrowser or "firefox";
  browserExe = lib.getExe config.programs.${preferredBrowser}.finalPackage;
  browserArgs = [ "--new-window" ];
  browserNeedles = [ preferredBrowser ];

  openvfs = pkgs.stdenv.mkDerivation {
    pname = "openvfs";
    version = "0.1.0";

    # Qt6 is present only so ECM can query qtpaths during configuration.
    # OpenVFS itself does not require Qt application wrapping.
    dontWrapQtApps = true;

    src = pkgs.fetchFromGitHub {
      owner = "opencloud-eu";
      repo = "openvfs";
      rev = "2fdf61c8a7c1bb57d78b618ac17eaed12208db7d";
      sha256 = "1nh1pbdb96zyyjsbjf2hfg5whxjdzf5w67yygb2z8rdzzj0chvq4";
    };

    # Bound hydration waits, deliver cancellation, and recognize Nix-wrapped KIO.
    patches = [ ./patches/openvfs-hydration-liveness.patch ];

    nativeBuildInputs = [
      pkgs.cmake
      pkgs.kdePackages.extra-cmake-modules
      pkgs.kdePackages.qtbase
      pkgs.pkg-config
    ];

    buildInputs = [
      pkgs.fuse3
      pkgs.nlohmann_json
    ];
  };

  clientSource = import ./client-source.nix { inherit (pkgs) fetchFromGitHub; };

  client = pkgs.nextcloud-client.overrideAttrs (old: {
    version = "34.0.50-openvfs";

    inherit (clientSource) src patches;

    buildInputs = (old.buildInputs or [ ]) ++ [
      openvfs
      pkgs.kdsingleapplication
    ];

    cmakeFlags = (old.cmakeFlags or [ ]) ++ [
      "-DENFORCE_SINGLE_ACCOUNT=ON"
      # No classic-sync choice, and a classic folder becomes virtual files
      # on the next client start.
      "-DENFORCE_VIRTUAL_FILES_SYNC_FOLDER=ON"
    ];

    postPatch = (old.postPatch or "") + ''
      if ! grep -Fq '"--overrideserverurl"' src/gui/application.cpp \
        || ! grep -Fq '"--overridelocaldir"' src/gui/application.cpp
      then
        echo "Nextcloud interactive wizard override CLI changed" >&2
        exit 1
      fi

      if ! grep -Eq 'option\([[:space:]]*ENFORCE_SINGLE_ACCOUNT' NEXTCLOUD.cmake; then
        echo "Nextcloud no longer exposes ENFORCE_SINGLE_ACCOUNT" >&2
        exit 1
      fi

      account_settings='src/gui/accountsettings.cpp'
      vfs_mode_old='if (folderWizard->property("useVirtualFiles").toBool()) {
        definition.virtualFilesMode = bestAvailableVfsMode();
    }'
      vfs_mode_new='if (ConfigFile().isVfsEnabled() || folderWizard->property("useVirtualFiles").toBool()) {
        definition.virtualFilesMode = bestAvailableVfsMode();
    }'

      vfs_wizard_old='if (definition.virtualFilesMode != Vfs::Off && folderWizard->property("useVirtualFiles").toBool()) {'
      vfs_wizard_new='if (definition.virtualFilesMode != Vfs::Off && (ConfigFile().isVfsEnabled() || folderWizard->property("useVirtualFiles").toBool())) {'

      if [ ! -f "$account_settings" ]; then
        echo "Nextcloud account settings source moved" >&2
        exit 1
      fi

      if grep -Fq "$vfs_mode_old" "$account_settings"; then
        substituteInPlace "$account_settings" \
          --replace-fail \
            "$vfs_mode_old" \
            "$vfs_mode_new"
      elif ! grep -Fq "$vfs_mode_new" "$account_settings"; then
        echo "Nextcloud VFS mode-selection policy changed" >&2
        exit 1
      fi

      if grep -Fq "$vfs_wizard_old" "$account_settings"; then
        substituteInPlace "$account_settings" \
          --replace-fail \
            "$vfs_wizard_old" \
            "$vfs_wizard_new"
      elif ! grep -Fq "$vfs_wizard_new" "$account_settings"; then
        echo "Nextcloud VFS folder-wizard policy changed" >&2
        exit 1
      fi

      echo "GjallarOS OpenVFS folder-wizard policy enforced"

      application='src/gui/application.cpp'
      autostart_enable='Utility::setLaunchOnStartup(_theme->appName(), _theme->appNameGUI(), true);'
      autostart_disable='Utility::setLaunchOnStartup(_theme->appName(), _theme->appNameGUI(), false);'

      if [ ! -f "$application" ]; then
        echo "Nextcloud application startup policy source moved" >&2
        exit 1
      fi

      if grep -Fq "$autostart_enable" "$application"; then
        substituteInPlace "$application" \
          --replace-fail \
            "$autostart_enable" \
            "$autostart_disable"
      elif ! grep -Fq "$autostart_disable" "$application"; then
        echo "Nextcloud wizard autostart policy changed" >&2
        exit 1
      fi

      echo "upstream Nextcloud autostart disabled"

      browser_utility='src/gui/guiutility.cpp'
      if [ ! -f "$browser_utility" ] \
        || ! grep -Fq 'bool Utility::openBrowser(const QUrl &url' "$browser_utility"
      then
        echo "Nextcloud shared browser-opening API changed" >&2
        exit 1
      fi

      if ! grep -Fq '#include <QProcess>' "$browser_utility"; then
        substituteInPlace "$browser_utility" \
          --replace-fail \
            '#include <QDesktopServices>' \
            '#include <QDesktopServices>
#include <QProcess>'
      fi

      if grep -Fq 'if (!QDesktopServices::openUrl(url)) {' "$browser_utility"; then
        substituteInPlace "$browser_utility" \
          --replace-fail \
            'if (!QDesktopServices::openUrl(url)) {' \
            'const auto nextcloudBrowserOpener = qEnvironmentVariable("NEXTCLOUD_BROWSER_OPENER");
    if (!nextcloudBrowserOpener.isEmpty()) {
        const QStringList nextcloudBrowserArguments{url.toString(QUrl::FullyEncoded)};
        if (QProcess::startDetached(nextcloudBrowserOpener, nextcloudBrowserArguments)) {
            return true;
        }
        qCWarning(lcUtility) << "Nextcloud browser opener failed for" << url;
        return false;
    }

    if (!QDesktopServices::openUrl(url)) {'
      elif ! grep -Fq 'NEXTCLOUD_BROWSER_OPENER' "$browser_utility"; then
        echo "Nextcloud browser-opening implementation changed" >&2
        exit 1
      fi

      flow2=""
      for candidate in src/gui/creds/flow2auth.cpp src/gui/flow2auth.cpp; do
        if [ -f "$candidate" ]; then
          flow2="$candidate"
          break
        fi
      done

      if [ -z "$flow2" ]; then
        echo "Nextcloud Flow2 authentication source moved" >&2
        exit 1
      fi

      if grep -Fq 'QDesktopServices::openUrl(' "$flow2"; then
        if ! grep -Fq '#include "guiutility.h"' "$flow2"; then
          sed -i '0,/^#include /s//#include "guiutility.h"\n&/' "$flow2"
        fi
        substituteInPlace "$flow2" \
          --replace-fail \
            'QDesktopServices::openUrl(' \
            'Utility::openBrowser('
      elif ! grep -Fq 'Utility::openBrowser(' "$flow2"; then
        echo "unknown Nextcloud Flow2 browser-opening path" >&2
        exit 1
      fi

      if ! grep -Fq '#include <QProcess>' "$flow2"; then
        sed -i '0,/^#include /s//#include <QProcess>\n&/' "$flow2"
      fi

      if grep -Fq 'emit result(LoggedIn, QString(), loginName, appPassword);' "$flow2"; then
        substituteInPlace "$flow2" \
          --replace-fail \
            'emit result(LoggedIn, QString(), loginName, appPassword);' \
            'emit result(LoggedIn, QString(), loginName, appPassword);
    const auto nextcloudBrowserCloser = qEnvironmentVariable("NEXTCLOUD_BROWSER_CLOSER");
    if (!nextcloudBrowserCloser.isEmpty()) {
        QProcess::startDetached(nextcloudBrowserCloser, {});
    }'
      elif grep -Fq 'Q_EMIT result(LoggedIn, QString(), loginName, appPassword);' "$flow2"; then
        substituteInPlace "$flow2" \
          --replace-fail \
            'Q_EMIT result(LoggedIn, QString(), loginName, appPassword);' \
            'Q_EMIT result(LoggedIn, QString(), loginName, appPassword);
    const auto nextcloudBrowserCloser = qEnvironmentVariable("NEXTCLOUD_BROWSER_CLOSER");
    if (!nextcloudBrowserCloser.isEmpty()) {
        QProcess::startDetached(nextcloudBrowserCloser, {});
    }'
      elif ! grep -Fq 'NEXTCLOUD_BROWSER_CLOSER' "$flow2"; then
        echo "unknown Nextcloud Flow2 success path" >&2
        exit 1
      fi

      socket_api='src/gui/socketapi/socketapi.cpp'
      dolphin_overlay='shell_integration/dolphin/ownclouddolphinoverlayplugin.cpp'

      if [ ! -f "$socket_api" ] || [ ! -f "$dolphin_overlay" ]; then
        echo "Nextcloud Dolphin/OpenVFS shell integration source moved" >&2
        exit 1
      fi

      availability_helper='void setClipboardText(const QString &text)'
      availability_helper_new='QString openVfsAvailabilitySocketSuffix(OCC::Folder *folder, const QString &relativePath)
{
    if (!folder || folder->vfs().mode() != OCC::Vfs::OpenVFS) {
        return {};
    }

    const auto availability = folder->vfs().availability(
        relativePath,
        OCC::Vfs::AvailabilityRecursivity::NotRecursiveAvailability);

    if (!availability) {
        return {};
    }

    switch (*availability) {
    case OCC::VfsItemAvailability::AlwaysLocal:
    case OCC::VfsItemAvailability::AllHydrated:
        return QStringLiteral("+VFS_LOCAL");
    case OCC::VfsItemAvailability::AllDehydrated:
    case OCC::VfsItemAvailability::OnlineOnly:
        return QStringLiteral("+VFS_ONLINE_ONLY");
    case OCC::VfsItemAvailability::Mixed:
        return QStringLiteral("+VFS_HYDRATING");
    }

    return {};
}

void setClipboardText(const QString &text)'

      if grep -Fq "$availability_helper" "$socket_api" \
        && ! grep -Fq 'openVfsAvailabilitySocketSuffix' "$socket_api"
      then
        substituteInPlace "$socket_api" \
          --replace-fail \
            "$availability_helper" \
            "$availability_helper_new"
      elif ! grep -Fq 'openVfsAvailabilitySocketSuffix' "$socket_api"; then
        echo "Nextcloud socket availability helper anchor changed" >&2
        exit 1
      fi

      broadcast_old='    QString msg = buildMessage(QLatin1String("STATUS"), systemPath, fileStatus.toSocketAPIString());'
      broadcast_new='    const auto fileData = FileData::get(systemPath);
    const QString statusString = fileStatus.toSocketAPIString()
        + openVfsAvailabilitySocketSuffix(fileData.folder, fileData.folderRelativePath);
    QString msg = buildMessage(QLatin1String("STATUS"), systemPath, statusString);'

      if grep -Fq "$broadcast_old" "$socket_api"; then
        substituteInPlace "$socket_api" \
          --replace-fail \
            "$broadcast_old" \
            "$broadcast_new"
      elif ! grep -Fq 'openVfsAvailabilitySocketSuffix(fileData.folder, fileData.folderRelativePath)' "$socket_api"; then
        echo "Nextcloud socket status-broadcast implementation changed" >&2
        exit 1
      fi

      retrieve_old='        SyncFileStatus fileStatus = fileData.syncFileStatus();
        statusString = fileStatus.toSocketAPIString();'
      retrieve_new='        SyncFileStatus fileStatus = fileData.syncFileStatus();
        statusString = fileStatus.toSocketAPIString()
            + openVfsAvailabilitySocketSuffix(fileData.folder, fileData.folderRelativePath);'

      if grep -Fq "$retrieve_old" "$socket_api"; then
        substituteInPlace "$socket_api" \
          --replace-fail \
            "$retrieve_old" \
            "$retrieve_new"
      elif ! grep -Fq 'statusString = fileStatus.toSocketAPIString()' "$socket_api" \
        || ! grep -Fq 'openVfsAvailabilitySocketSuffix(fileData.folder, fileData.folderRelativePath)' "$socket_api"
      then
        echo "Nextcloud socket status-retrieval implementation changed" >&2
        exit 1
      fi

      overlay_old='        if (status.startsWith("OK")) {
            r << QStringLiteral("vcs-normal");
        }'
      overlay_new='        if (status.startsWith("OK")) {
            if (status.contains("+VFS_ONLINE_ONLY")) {
                r << QStringLiteral("${dolphinNextcloudIcons.onlineOnly}");
            } else if (status.contains("+VFS_HYDRATING")) {
                r << QStringLiteral("vcs-update-required");
            } else {
                // Any other OK state is fully synced and locally available.
                r << QStringLiteral("${dolphinNextcloudIcons.local}");
            }
        }'

      if grep -Fq "$overlay_old" "$dolphin_overlay"; then
        substituteInPlace "$dolphin_overlay" \
          --replace-fail \
            "$overlay_old" \
            "$overlay_new"
      elif ! grep -Fq '+VFS_ONLINE_ONLY' "$dolphin_overlay"; then
        echo "Nextcloud Dolphin overlay mapping changed" >&2
        exit 1
      fi

      echo "Nextcloud Dolphin OpenVFS availability overlays secured"

      echo "Nextcloud Flow2 auth-window close hook secured"
      echo "Nextcloud enrollment browser hook secured"
    '';

    postInstall = (old.postInstall or "") + ''
      vfs_plugin="$(${pkgs.findutils}/bin/find "$out" \
        \( -type f -o -type l \) \
        -name 'nextcloudsync_vfs_*.so' \
        -print -quit)"

      if [ -z "$vfs_plugin" ]; then
        echo "Nextcloud build has no Linux VFS plugin; refusing classic-sync fallback" >&2
        exit 1
      fi

      echo "Nextcloud Linux VFS plugin: $vfs_plugin"
    '';
  });

  nextcloudBrowserCloser = pkgs.writeShellScript "nextcloud-browser-close" ''
    session="''${NEXTCLOUD_AUTH_SESSION:-}"

    if [ -z "$session" ]; then
      exit 0
    fi

    if ! printf '%s\n' "$session" \
      | ${pkgs.gnugrep}/bin/grep -Eq '^[A-Za-z0-9._-]+$'
    then
      echo "invalid Nextcloud auth session identifier." >&2
      exit 64
    fi

    runtime="''${XDG_RUNTIME_DIR:-/run/user/$(${pkgs.coreutils}/bin/id -u)}"
    state_dir="$runtime/nextcloud-auth"
    state="$state_dir/$session.window"

    if [ ! -f "$state" ]; then
      exit 0
    fi

    IFS= read -r address < "$state" || address=""
    ${pkgs.coreutils}/bin/rm -f "$state"

    if ! printf '%s\n' "$address" \
      | ${pkgs.gnugrep}/bin/grep -Eq '^0x[[:xdigit:]]+$'
    then
      exit 0
    fi

    if [ -z "''${HYPRLAND_INSTANCE_SIGNATURE:-}" ]; then
      exit 0
    fi

    hyprctl=${lib.getExe' pkgs.hyprland "hyprctl"}
    python=${lib.getExe pkgs.python3}

    valid="$($hyprctl clients -j 2>/dev/null \
      | $python -c '
import json, sys

target = sys.argv[1]
needles = [value.lower() for value in sys.argv[2:] if value]

try:
    clients = json.load(sys.stdin)
except Exception:
    raise SystemExit(0)

for client in clients:
    if client.get("address") != target:
        continue

    identity = " ".join(str(client.get(key, "")) for key in (
        "class", "initialClass", "title", "initialTitle"
    )).lower()

    if needles and not any(needle in identity for needle in needles):
        raise SystemExit(0)

    print(target)
    break
' "$address" ${lib.escapeShellArgs browserNeedles} 2>/dev/null || true)"

    if [ "$valid" = "$address" ]; then
      $hyprctl dispatch closewindow "address:$address" \
        >/dev/null 2>&1 || true
    fi

    ${pkgs.coreutils}/bin/rmdir "$state_dir" 2>/dev/null || true
  '';

  nextcloudBrowserOpener = pkgs.writeShellScript "nextcloud-browser-open" ''
    if [ "$#" -ne 1 ]; then
      echo "Nextcloud browser opener expects exactly one URL." >&2
      exit 64
    fi

    url="$1"
    case "$url" in
      https://*) ;;
      *)
        echo "Nextcloud browser opener rejected non-HTTPS URL." >&2
        exit 64
        ;;
    esac

    hyprctl=${lib.getExe' pkgs.hyprland "hyprctl"}
    python=${lib.getExe pkgs.python3}
    workspace="''${NEXTCLOUD_WORKSPACE:-}"

    if [ -n "''${HYPRLAND_INSTANCE_SIGNATURE:-}" ]; then
      nextcloud_workspace="$($hyprctl clients -j 2>/dev/null \
        | $python -c '
import json, sys
try:
    clients = json.load(sys.stdin)
except Exception:
    clients = []

candidates = []
for client in clients:
    identity = " ".join(str(client.get(key, "")) for key in (
        "class", "initialClass", "title", "initialTitle"
    )).lower()
    if "nextcloud" not in identity:
        continue
    workspace = client.get("workspace") or {}
    wid = workspace.get("id")
    if not isinstance(wid, int) or wid <= 0:
        continue
    focus = client.get("focusHistoryID")
    candidates.append((focus if isinstance(focus, int) else 999999, wid))

if candidates:
    candidates.sort()
    print(candidates[0][1])
' 2>/dev/null || true)"

      if [ -n "$nextcloud_workspace" ]; then
        workspace="$nextcloud_workspace"
      fi
    fi

    before="$(${pkgs.coreutils}/bin/mktemp)"
    trap '${pkgs.coreutils}/bin/rm -f "$before"' EXIT

    if [ -n "''${HYPRLAND_INSTANCE_SIGNATURE:-}" ]; then
      $hyprctl clients -j 2>/dev/null \
        | $python -c '
import json, sys
try:
    clients = json.load(sys.stdin)
except Exception:
    clients = []
for client in clients:
    address = client.get("address")
    if address:
        print(address)
' >"$before" 2>/dev/null || true
    fi

    ${lib.escapeShellArgs ([ browserExe ] ++ browserArgs)} "$url" \
      >/dev/null 2>&1 &

    if [ -n "''${HYPRLAND_INSTANCE_SIGNATURE:-}" ] && [ -n "$workspace" ]; then
      address=""
      attempts=0
      while [ "$attempts" -lt 50 ] && [ -z "$address" ]; do
        address="$($hyprctl clients -j 2>/dev/null \
          | $python -c '
import json, sys
before_path = sys.argv[1]
needles = [value.lower() for value in sys.argv[2:] if value]
try:
    with open(before_path, encoding="utf-8") as handle:
        before = {line.strip() for line in handle if line.strip()}
    clients = json.load(sys.stdin)
except Exception:
    raise SystemExit(0)

candidates = []
for client in clients:
    address = client.get("address")
    if not address or address in before:
        continue
    identity = " ".join(str(client.get(key, "")) for key in (
        "class", "initialClass", "title", "initialTitle"
    )).lower()
    if needles and not any(needle in identity for needle in needles):
        continue
    focus = client.get("focusHistoryID")
    candidates.append((focus if isinstance(focus, int) else 999999, address))

if candidates:
    candidates.sort()
    print(candidates[0][1])
' "$before" ${lib.escapeShellArgs browserNeedles} 2>/dev/null || true)"

        if [ -z "$address" ]; then
          ${pkgs.coreutils}/bin/sleep 0.1
        fi
        attempts=$((attempts + 1))
      done

      if [ -n "$address" ]; then
        $hyprctl dispatch movetoworkspacesilent "$workspace,address:$address" >/dev/null 2>&1 || true
        $hyprctl dispatch focuswindow "address:$address" >/dev/null 2>&1 || true

        session="''${NEXTCLOUD_AUTH_SESSION:-}"
        if [ -n "$session" ] \
          && printf '%s\n' "$session" \
            | ${pkgs.gnugrep}/bin/grep -Eq '^[A-Za-z0-9._-]+$'
        then
          runtime="''${XDG_RUNTIME_DIR:-/run/user/$(${pkgs.coreutils}/bin/id -u)}"
          state_dir="$runtime/nextcloud-auth"

          ${pkgs.coreutils}/bin/install -d -m 0700 "$state_dir"

          printf '%s\n' "$address" >"$state_dir/$session.window"
          ${pkgs.coreutils}/bin/chmod 0600 "$state_dir/$session.window"
        fi
      else
        echo "browser opened, but its new Hyprland window could not be identified." >&2
      fi
    fi
  '';

  nextcloudConfigPolicy = pkgs.writeShellScript "nextcloud-config-policy" ''
    exec ${lib.getExe pkgs.python3} - "$@" <<'PY'
import os
import re
import sys
from pathlib import Path

if len(sys.argv) != 2:
    print("usage: nextcloud-config-policy <config>", file=sys.stderr)
    raise SystemExit(64)

path = Path(os.path.expanduser(sys.argv[1]))

path.parent.mkdir(parents=True, exist_ok=True)
try:
    text = path.read_text(encoding="utf-8", errors="replace")
except FileNotFoundError:
    text = ""

lines = text.splitlines()
required = {
    "showMainDialogAsNormalWindow": "true",
    "isVfsEnabled": "true",
}

section_start = None
section_end = len(lines)
for index, line in enumerate(lines):
    if line.strip() == "[General]":
        section_start = index
        for end in range(index + 1, len(lines)):
            stripped = lines[end].strip()
            if stripped.startswith("[") and stripped.endswith("]"):
                section_end = end
                break
        break

if section_start is None:
    prefix = ["[General]"] + [f"{key}={value}" for key, value in required.items()]
    lines = prefix + ([""] if lines else []) + lines
else:
    seen = set()
    for index in range(section_start + 1, section_end):
        if "=" not in lines[index]:
            continue
        key = lines[index].split("=", 1)[0].strip()
        if key in required:
            lines[index] = f"{key}={required[key]}"
            seen.add(key)
    for key in reversed([key for key in required if key not in seen]):
        lines.insert(section_start + 1, f"{key}={required[key]}")
        section_end += 1

output = "\n".join(lines).rstrip("\n") + "\n"
tmp = path.with_name(f".{path.name}.policy-{os.getpid()}")
tmp.write_text(output, encoding="utf-8")
os.chmod(tmp, 0o600)
os.replace(tmp, path)
PY
  '';

  nextcloudFolderPolicy = pkgs.writeShellScript "nextcloud-folder-policy" ''
    exec ${lib.getExe pkgs.python3} - "$@" <<'PY'
import os
import re
import sys

if len(sys.argv) != 3:
    print("usage: nextcloud-folder-policy <config> <canonical-root>", file=sys.stderr)
    raise SystemExit(64)

config, canonical_root = sys.argv[1:]
canonical_root = os.path.realpath(os.path.expanduser(canonical_root))

home = os.path.dirname(canonical_root)
try:
    entries = os.scandir(home)
except FileNotFoundError:
    entries = ()

for entry in entries:
    if not entry.is_dir(follow_symlinks=False):
        continue
    if not re.fullmatch(r"Nextcloud(?:[ _-]?\d+| \(\d+\))", entry.name):
        continue
    duplicate = os.path.realpath(entry.path)
    if duplicate == canonical_root:
        continue
    print(
        "Nextcloud policy refused: duplicate-looking local root exists:\n"
        f"  duplicate: {duplicate}\n"
        f"  canonical: {canonical_root}",
        file=sys.stderr,
    )
    raise SystemExit(78)

try:
    with open(config, "r", encoding="utf-8", errors="replace") as handle:
        lines = handle.readlines()
except FileNotFoundError:
    lines = []

accounts = set()
folders = {}

for raw in lines:
    line = raw.rstrip("\r\n")
    if "=" not in line:
        continue

    key, value = line.split("=", 1)

    match = re.fullmatch(r"(\d+)\\url", key)
    if match and value.strip():
        accounts.add(match.group(1))
        continue

    match = re.fullmatch(
        r"(\d+)\\(Folders|FoldersWithPlaceholders)\\(\d+)\\"
        r"(localPath|targetPath|virtualFilesMode)",
        key,
    )
    if not match:
        continue

    account, folder_kind, folder, field = match.groups()
    fields = folders.setdefault((account, folder), {})

    previous_kind = fields.get("_kind")
    if previous_kind is not None and previous_kind != folder_kind:
        print(
            "Nextcloud policy refused: conflicting folder storage groups "
            f"for account {account}, folder {folder}.",
            file=sys.stderr,
        )
        raise SystemExit(78)

    fields["_kind"] = folder_kind
    fields[field] = value.strip()

if not accounts:
    if os.path.isdir(canonical_root):
        try:
            has_content = next(os.scandir(canonical_root), None) is not None
        except OSError as exc:
            print(
                f"Nextcloud policy refused: cannot inspect canonical root: {exc}",
                file=sys.stderr,
            )
            raise SystemExit(78)
        if has_content:
            print(
                "Nextcloud policy refused: unconfigured canonical root is not empty; "
                "remove or relocate stale local data before fresh VFS enrollment.",
                file=sys.stderr,
            )
            raise SystemExit(78)
    raise SystemExit(10)

if len(accounts) != 1:
    print(
        "Nextcloud policy refused: exactly one configured account is supported; "
        f"found {len(accounts)}.",
        file=sys.stderr,
    )
    raise SystemExit(78)

if not folders:
    raise SystemExit(11)

if len(folders) != 1:
    print(
        "Nextcloud policy refused: exactly one sync-folder connection is allowed; "
        f"found {len(folders)}.",
        file=sys.stderr,
    )
    raise SystemExit(78)

(account, _folder), fields = next(iter(folders.items()))
if account not in accounts:
    print(
        "Nextcloud policy refused: sync-folder ownership does not match the configured account.",
        file=sys.stderr,
    )
    raise SystemExit(78)

local_path = fields.get("localPath", "")
if not local_path:
    print("Nextcloud policy refused: sync folder has no localPath.", file=sys.stderr)
    raise SystemExit(78)

local_path = os.path.realpath(os.path.expanduser(local_path))
if local_path != canonical_root:
    print(
        "Nextcloud policy refused: configured local root is not the canonical root:\n"
        f"  configured: {local_path}\n"
        f"  required:   {canonical_root}",
        file=sys.stderr,
    )
    raise SystemExit(78)

if not os.path.isdir(canonical_root):
    print(
        "Nextcloud policy refused: configured canonical root is missing; "
        "refusing to recreate sync state implicitly.",
        file=sys.stderr,
    )
    raise SystemExit(78)

if fields.get("targetPath", "") != "/":
    print(
        "Nextcloud policy refused: the single sync connection must target remote root '/'.",
        file=sys.stderr,
    )
    raise SystemExit(78)

if fields.get("_kind") != "FoldersWithPlaceholders":
    print(
        "Nextcloud policy refused: classic sync folders are not allowed; "
        "GjallarOS requires an OpenVFS placeholder folder.",
        file=sys.stderr,
    )
    raise SystemExit(78)

if fields.get("virtualFilesMode", "").strip().lower() != "openvfs":
    print(
        "Nextcloud policy refused: virtualFilesMode must be 'openvfs'.",
        file=sys.stderr,
    )
    raise SystemExit(78)

PY
  '';

  dolphinPlace = pkgs.writeShellScript "nextcloud-dolphin-place" ''
    exec ${lib.getExe pkgs.python3} ${./dolphin-place.py} ${lib.escapeShellArg dolphinNextcloudIcons.places}
  '';

  enrollmentLauncher = pkgs.writeShellScriptBin "nextcloud-enroll" ''
    config="''${XDG_CONFIG_HOME:-$HOME/.config}/Nextcloud/nextcloud.cfg"
    root="$HOME/Nextcloud"
    logdir="''${XDG_STATE_HOME:-$HOME/.local/state}/nextcloud"

    ${pkgs.coreutils}/bin/install -d -m 0700 "$logdir"
    ${nextcloudConfigPolicy} "$config"

    policy_status=0
    ${nextcloudFolderPolicy} "$config" "$root" || policy_status=$?

    case "$policy_status" in
      0)
        ${dolphinPlace} || true
        exec ${lib.getExe client} --logdir "$logdir" --logflush "$@"
        ;;
      10)
        ;;
      11)
        echo "Nextcloud policy refused: account setup is incomplete; remove the stale local account state and re-enroll."
        exit 78
        ;;
      *)
        exit "$policy_status"
        ;;
    esac

    ${pkgs.coreutils}/bin/install -d -m 0700 "$root"
    ${dolphinPlace} || true

    if [ "$policy_status" -eq 10 ]; then
      auth_session="$$-$RANDOM-$RANDOM"
      export NEXTCLOUD_AUTH_SESSION="$auth_session"
      export NEXTCLOUD_BROWSER_OPENER=${nextcloudBrowserOpener}
      export NEXTCLOUD_BROWSER_CLOSER=${nextcloudBrowserCloser}

      if [ -n "''${HYPRLAND_INSTANCE_SIGNATURE:-}" ]; then
        workspace="$(${lib.getExe' pkgs.hyprland "hyprctl"} activeworkspace -j 2>/dev/null \
          | ${lib.getExe pkgs.python3} -c '
import json, sys
try:
    workspace = json.load(sys.stdin)
except Exception:
    raise SystemExit(0)
wid = workspace.get("id")
if isinstance(wid, int) and wid > 0:
    print(wid)
' 2>/dev/null || true)"
        if [ -n "$workspace" ]; then
          export NEXTCLOUD_WORKSPACE="$workspace"
        fi
      fi

      override_status=0
      ${lib.getExe client} \
        --overrideserverurl ${lib.escapeShellArg host} \
        --overridelocaldir "$root" \
        || override_status=$?

      if [ "$override_status" -ne 0 ]; then
        echo "Nextcloud enrollment defaults could not be written." >&2
        exit "$override_status"
      fi

      exec ${lib.getExe client} --logdir "$logdir" --logflush "$@"
    fi
  '';

  managedClient = pkgs.symlinkJoin {
    name = "nextcloud-client-managed";
    paths = [
      client
      openvfs
      pkgs.fuse3
    ];
    nativeBuildInputs = [ pkgs.makeWrapper ];

    postBuild = ''
      rm -f "$out/bin/nextcloud"
      makeWrapper \
        ${enrollmentLauncher}/bin/nextcloud-enroll \
        "$out/bin/nextcloud" \
        --prefix XDG_CONFIG_DIRS : "$out/etc/xdg"

      rm -f "$out"/etc/xdg/autostart/*nextcloud*.desktop

      desktop="$out/share/applications/com.nextcloud.desktopclient.nextcloud.desktop"

      if [ -L "$desktop" ]; then
        source="$(${pkgs.coreutils}/bin/readlink -f "$desktop")"
        rm "$desktop"
        cp "$source" "$desktop"
        chmod u+w "$desktop"
      fi

      if [ -f "$desktop" ]; then
        sed -i \
          "0,/^Exec=/{s|^Exec=.*|Exec=$out/bin/nextcloud|}" \
          "$desktop"
      fi
    '';
  };

  syncRunner = pkgs.writeShellScript "nextcloud-sync" ''
    config="''${XDG_CONFIG_HOME:-$HOME/.config}/Nextcloud/nextcloud.cfg"
    root="$HOME/Nextcloud"

    ${nextcloudConfigPolicy} "$config"

    policy_status=0
    ${nextcloudFolderPolicy} "$config" "$root" || policy_status=$?

    case "$policy_status" in
      0)
        ;;
      10)
        echo "Nextcloud account setup is incomplete; automatic enrollment is disabled."
        echo "Run nextcloud-enroll to configure it."
        exit 0
        ;;
      11)
        echo "Nextcloud folder setup is incomplete; automatic sync remains disabled until a clean VFS enrollment is completed."
        exit 0
        ;;
      *)
        exit "$policy_status"
        ;;
    esac

    ${dolphinPlace} || true

    exec ${managedClient}/bin/nextcloud --background
  '';
in
lib.mkIf config.gjallar.apps.nextcloud.enable {
  assertions = lib.optionals enable [
    {
      assertion = host != "";
      message = "Nextcloud integration requires nextcloudHost.";
    }
    {
      assertion = lib.hasPrefix "https://" host;
      message = "Nextcloud integration requires an HTTPS nextcloudHost.";
    }
  ];

  home.packages = lib.optionals enable [
    managedClient
    enrollmentLauncher
  ];


  home.activation.nextcloudAutostartOwnership = lib.mkIf enable (
    lib.hm.dag.entryAfter [ "writeBoundary" ] ''
      ${pkgs.coreutils}/bin/rm -f \
        "$HOME/.config/autostart/Nextcloud.desktop" \
        "$HOME/.config/autostart/com.nextcloud.desktopclient.nextcloud.desktop"
    ''
  );

  home.activation.dolphinNextcloudPlace = lib.mkIf enable (
    lib.hm.dag.entryAfter [ "writeBoundary" ] ''
      run ${dolphinPlace} || true
    ''
  );

  systemd.user.services.nextcloud-sync = lib.mkIf enable {
    Unit = {
      Description = "Nextcloud synchronization";
      After = [ "graphical-session.target" ];
      PartOf = [ "graphical-session.target" ];
    };

    Service = {
      Type = "simple";
      ExecStart = syncRunner;

      Restart = "no";

    };

    Install.WantedBy = [
      "graphical-session.target"
    ];
  };

}
