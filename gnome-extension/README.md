# warp-speed — GNOME Shell extension

A top-bar indicator version of warp-speed: shows live download/upload speed
and ping right in the GNOME top panel, with a dropdown to test ping
against any domain or IP address.

This is a separate implementation from the Go TUI in the rest of this
repo — GNOME Shell extensions are written in GJS (GNOME's JavaScript
bindings), not Go, since they run inside the Shell's own process.

## Before you publish: rename the UUID

The extension's directory name and `metadata.json` both currently use the
placeholder UUID `warp-speed@ah-asif.github.io`. GNOME Shell
extension UUIDs are conventionally styled like an email address / reverse
domain and must be **globally unique** on extensions.gnome.org. Replace
`ah-asif` in **both places** with something identifying you:

```bash
cd gnome-extension
OLD="warp-speed@ah-asif.github.io"
NEW="warp-speed@yourusername.github.io"   # <-- pick your own
mv "$OLD" "$NEW"
sed -i "s/$OLD/$NEW/" "$NEW/metadata.json"
```

Also update the `url` field in `metadata.json` to point at your repo.

## 1. Install it locally to test

```bash
cp -r gnome-extension/warp-speed@yourusername.github.io \
      ~/.local/share/gnome-shell/extensions/
```

**On X11:** press `Alt`+`F2`, type `r`, press Enter to restart GNOME
Shell, then:

```bash
gnome-extensions enable warp-speed@yourusername.github.io
```

**On Wayland:** you can't restart the Shell in-place, so log out and log
back in first, then run the same `enable` command.

Watch for errors while developing with:

```bash
journalctl -f -o cat /usr/bin/gnome-shell
```

You should see a label like `↓0.0M ↑0.0M` appear in the top bar. Click it
to see the full stats and the ping-test entry box.

## 2. Package it

GNOME ships a tool that validates `metadata.json` and zips the extension
correctly:

```bash
cd ~/.local/share/gnome-shell/extensions/warp-speed@yourusername.github.io
gnome-extensions pack --force
```

This produces `warp-speed@yourusername.github.io.shell-extension.zip` in
the current directory — this is the exact file extensions.gnome.org
expects.

## 3. Publish to extensions.gnome.org

1. Create an account at <https://extensions.gnome.org> (or log in with
   GNOME/Fedora/Ubuntu SSO).
2. Go to **Upload extension** and upload the `.zip` from step 2.
3. Fill in the listing: description, screenshot(s) of the top-bar label
   and the dropdown menu, and a link back to your GitHub repo.
4. Submit for review.

Review is done manually by GNOME's extension review team and can take
anywhere from a few days to a few weeks depending on queue length.
Common rejection reasons to check for before submitting:
- No `eval()`, no dynamic `imports.*` outside the standard pattern
  (this extension doesn't use either).
- No unexplained network access — this extension only shells out to the
  local `ping` binary, which is fine, but mention it in your listing
  description since reviewers do check for it.
- The UUID directory name must exactly match `metadata.json`'s `uuid`.

## 4. Let people install it in the meantime (or instead)

While waiting on review — or if you don't want to go through it at all —
people can install straight from GitHub:

```bash
git clone https://github.com/yourusername/warp-speed.git
cp -r warp-speed/gnome-extension/warp-speed@yourusername.github.io \
      ~/.local/share/gnome-shell/extensions/
# then restart the Shell (Alt+F2, r, Enter on X11; log out/in on Wayland)
gnome-extensions enable warp-speed@yourusername.github.io
```

## 5. Shipping updates

Bump nothing version-specific is required in `metadata.json` for Shell
extensions (there's no semver field), but do keep the `shell-version`
array current as new GNOME releases come out. To push an update to
extensions.gnome.org, re-run `gnome-extensions pack --force` and upload
the new zip as a new version of the same listing.

## Notes / limitations

- Ping uses the system `ping` binary via `Gio.Subprocess`, so no special
  permissions are needed (unlike the Go version's raw ICMP socket).
- Default ping host, refresh interval, and probe count are constants at
  the top of `extension.js` (`CONFIG`). There's no preferences UI yet —
  if you want one, it would use `prefs.js` + a compiled GSettings schema,
  which is a reasonable next step but adds real complexity for a first
  release.
