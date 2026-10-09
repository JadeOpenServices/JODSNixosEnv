"""Add ~/Nextcloud to the Places panel (KIO user-places.xbel).

Runs at activation, at enrollment and at each login. A fresh account has no
user-places.xbel until a KDE file view first opens, so the activation alone
never added the entry (2026-10-09). A missing file gets the Home entry KIO
itself writes plus Nextcloud and no kde_places_version, so KIO appends the
rest of its defaults and skips the URLs already present.
"""
import os
import sys
import time
from urllib.parse import quote
from xml.sax.saxutils import quoteattr

icon = sys.argv[1]
home = os.path.expanduser("~")
root = os.path.join(home, "Nextcloud")
data = os.environ.get("XDG_DATA_HOME") or os.path.join(home, ".local", "share")
places = os.path.join(data, "user-places.xbel")

# Before enrollment the folder does not exist; a dead entry helps nobody.
if not os.path.isdir(root):
    raise SystemExit(0)


def url(path):
    return "file://" + quote(path, safe="/")


def bookmark(path, title, icon_name, ident, system=False):
    extra = "\n    <isSystemItem>true</isSystemItem>" if system else ""
    return (
        f" <bookmark href={quoteattr(url(path))}>\n"
        f"  <title>{title}</title>\n"
        "  <info>\n"
        '   <metadata owner="http://freedesktop.org">\n'
        f"    <bookmark:icon name={quoteattr(icon_name)}/>\n"
        "   </metadata>\n"
        '   <metadata owner="http://www.kde.org">\n'
        f"    <ID>{ident}</ID>{extra}\n"
        "   </metadata>\n"
        "  </info>\n"
        " </bookmark>\n"
    )


now = int(time.time())
entry = bookmark(root, "Nextcloud", icon, f"{now}/900")

try:
    with open(places, encoding="utf-8") as handle:
        text = handle.read()
except FileNotFoundError:
    text = None

if text is None:
    text = (
        '<?xml version="1.0" encoding="UTF-8"?>\n'
        "<!DOCTYPE xbel>\n"
        '<xbel xmlns:bookmark="http://www.freedesktop.org/standards/desktop-bookmarks"'
        ' xmlns:kdepriv="http://www.kde.org/kdepriv"'
        ' xmlns:mime="http://www.freedesktop.org/standards/shared-mime-info">\n'
        + bookmark(home, "Home", "user-home", f"{now}/0", system=True)
        + entry
        + "</xbel>\n"
    )
else:
    if f"href={quoteattr(url(root))}" in text:
        raise SystemExit(0)
    # Next to the other local places: before Network, else before Trash,
    # else at the end.
    for anchor in ('<bookmark href="remote:/">', '<bookmark href="trash:/">', "</xbel>"):
        at = text.find(anchor)
        if at == -1:
            continue
        at = text.rfind("\n", 0, at) + 1
        text = text[:at] + entry + text[at:]
        break
    else:
        print(f"{places}: not an XBEL file, Nextcloud place not added", file=sys.stderr)
        raise SystemExit(1)

os.makedirs(data, exist_ok=True)
tmp = f"{places}.gjallar-nextcloud.{os.getpid()}"
with open(tmp, "w", encoding="utf-8") as handle:
    handle.write(text)
os.chmod(tmp, 0o600)
os.replace(tmp, places)
