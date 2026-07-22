import Clutter from 'gi://Clutter';
import Gio from 'gi://Gio';
import GLib from 'gi://GLib';
import GObject from 'gi://GObject';
import St from 'gi://St';

import { Extension } from 'resource:///org/gnome/shell/extensions/extension.js';
import * as Main from 'resource:///org/gnome/shell/ui/main.js';
import * as PanelMenu from 'resource:///org/gnome/shell/ui/panelMenu.js';
import * as PopupMenu from 'resource:///org/gnome/shell/ui/popupMenu.js';

const CONFIG = {
    refreshIntervalSec: 1,
    defaultPingHost: '8.8.8.8',
    pingTimeoutSec: 2,
    onDemandPingCount: 5,
    smoothingAlpha: 0.5,
};

const HOSTNAME_RE = /^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,63}$/;
const IPV4_RE = /^(\d{1,3}\.){3}\d{1,3}$/;

function isValidHost(s) {
    if (!s)
        return false;
    s = s.trim();
    if (s === '')
        return false;
    if (IPV4_RE.test(s))
        return s.split('.').every(part => Number(part) <= 255);
    if (s.includes(':'))
        return /^[0-9a-fA-F:]+$/.test(s);
    return HOSTNAME_RE.test(s);
}

function readIfaceStats(iface) {
    let contents;
    try {
        [, contents] = GLib.file_get_contents('/proc/net/dev');
    } catch (e) {
        return null;
    }
    const text = new TextDecoder('utf-8').decode(contents);
    for (const line of text.split('\n')) {
        if (!line.includes(':'))
            continue;
        const [namePart, rest] = line.split(':');
        const name = namePart.trim();
        if (name !== iface)
            continue;
        const fields = rest.trim().split(/\s+/);
        if (fields.length < 9)
            continue;
        const rx = parseInt(fields[0], 10);
        const tx = parseInt(fields[8], 10);
        if (Number.isNaN(rx) || Number.isNaN(tx))
            continue;
        return { rx, tx, time: GLib.get_monotonic_time() };
    }
    return null;
}

function defaultInterface() {
    let contents;
    try {
        [, contents] = GLib.file_get_contents('/proc/net/route');
    } catch (e) {
        return null;
    }
    const text = new TextDecoder('utf-8').decode(contents);
    const lines = text.split('\n').slice(1);
    for (const line of lines) {
        const fields = line.trim().split(/\s+/);
        if (fields.length < 2)
            continue;
        if (fields[1] === '00000000')
            return fields[0];
    }
    return null;
}

function computeRatesBytesPerSec(prev, cur) {
    if (!prev || !cur)
        return { down: 0, up: 0 };
    const dtSeconds = (cur.time - prev.time) / 1000000;
    if (dtSeconds <= 0)
        return { down: 0, up: 0 };
    if (cur.rx < prev.rx || cur.tx < prev.tx)
        return { down: 0, up: 0 };
    const downBps = (cur.rx - prev.rx) / dtSeconds;
    const upBps = (cur.tx - prev.tx) / dtSeconds;
    return { down: downBps, up: upBps };
}

function formatBytesPerSec(bytesPerSec) {
    const KB = 1024;
    const MB = KB * 1024;
    const GB = MB * 1024;

    if (bytesPerSec < KB) {
        const whole = Math.max(0, Math.round(bytesPerSec));
        return String(whole).padStart(3, ' ') + ' B/s';
    }
    if (bytesPerSec < MB)
        return (bytesPerSec / KB).toFixed(1).padStart(5, ' ') + ' KB/s';
    if (bytesPerSec < GB)
        return (bytesPerSec / MB).toFixed(1).padStart(5, ' ') + ' MB/s';
    return (bytesPerSec / GB).toFixed(1).padStart(5, ' ') + ' GB/s';
}

function formatMbps(bytesPerSec) {
    const mbps = (bytesPerSec * 8) / 1000000;
    return mbps.toFixed(1).padStart(5, ' ') + ' Mbps';
}

function formatSpeed(bytesPerSec, useMbps) {
    return useMbps ? formatMbps(bytesPerSec) : formatBytesPerSec(bytesPerSec);
}

function formatPing(pingMs, ok) {
    if (!ok)
        return 'timeout';
    return String(Math.round(pingMs)).padStart(3, ' ') + ' ms';
}

const TIME_RE = /time=([\d.]+)\s*ms/g;
const SUMMARY_RE = /(\d+) packets transmitted, (\d+)( packets)? received/;
const RTT_RE = /=\s*([\d.]+)\/([\d.]+)\/([\d.]+)/;

function runPing(target, count, timeoutSec, cancellable, onDone) {
    let proc;
    try {
        proc = Gio.Subprocess.new(
            ['ping', '-c', String(count), '-W', String(timeoutSec), target],
            Gio.SubprocessFlags.STDOUT_PIPE | Gio.SubprocessFlags.STDERR_PIPE
        );
    } catch (e) {
        onDone({ sent: 0, received: 0, ok: false, error: String(e) });
        return;
    }

    proc.communicate_utf8_async(null, cancellable, (source, res) => {
        let stdout = '';
        try {
            [, stdout] = source.communicate_utf8_finish(res);
        } catch (e) {
            onDone({ sent: 0, received: 0, ok: false, error: String(e) });
            return;
        }

        const result = { sent: 0, received: 0, minMs: 0, avgMs: 0, maxMs: 0, lastMs: -1, ok: false };

        const summary = SUMMARY_RE.exec(stdout);
        if (summary) {
            result.sent = parseInt(summary[1], 10);
            result.received = parseInt(summary[2], 10);
        }

        const rtt = RTT_RE.exec(stdout);
        if (rtt) {
            result.minMs = parseFloat(rtt[1]);
            result.avgMs = parseFloat(rtt[2]);
            result.maxMs = parseFloat(rtt[3]);
        }

        const timeMatches = [...stdout.matchAll(TIME_RE)];
        if (timeMatches.length > 0) {
            result.lastMs = parseFloat(timeMatches[timeMatches.length - 1][1]);
            result.ok = true;
        }

        if (!result.ok && result.sent === 0)
            result.error = 'No reply / could not resolve host';

        onDone(result);
    });
}

const PingEntryItem = GObject.registerClass(
class PingEntryItem extends PopupMenu.PopupBaseMenuItem {
    _init(onSubmit) {
        super._init({ activate: false, hover: false, can_focus: false });

        this._entry = new St.Entry({
            hint_text: 'Enter domain or IP address',
            can_focus: true,
            x_expand: true,
            style_class: 'warp-speed-ping-entry',
        });

        this._entry.clutter_text.connect('key-press-event', (actor, event) => {
            const symbol = event.get_key_symbol();
            if (symbol === Clutter.KEY_Return || symbol === Clutter.KEY_KP_Enter) {
                onSubmit(this._entry.get_text());
                this._entry.set_text('');
                return Clutter.EVENT_STOP;
            }
            return Clutter.EVENT_PROPAGATE;
        });

        this.add_child(this._entry);
    }
});

export default class WarpSpeedExtension extends Extension {
    enable() {
        this._cancellable = new Gio.Cancellable();
        this._iface = defaultInterface() || 'eth0';
        this._prevSample = readIfaceStats(this._iface);

        this._smoothedDown = null;
        this._smoothedUp = null;
        this._bgPingMs = -1;
        this._bgPingOk = false;
        this._useMbps = false;

        this._indicator = new PanelMenu.Button(0.0, this.metadata.name, false);

        this._label = new St.Label({
            text: 'warp-speed …',
            y_align: Clutter.ActorAlign.CENTER,
            style_class: 'warp-speed-panel-label',
        });
        this._indicator.add_child(this._label);

        this._indicator.connect('button-press-event', (actor, event) => {
            if (event.get_button() === 2) {
                this._toggleUnits();
                return Clutter.EVENT_STOP;
            }
            return Clutter.EVENT_PROPAGATE;
        });

        this._buildMenu();

        Main.panel.addToStatusArea(this.uuid, this._indicator);

        this._refresh();
        this._timeoutId = GLib.timeout_add_seconds(
            GLib.PRIORITY_DEFAULT,
            CONFIG.refreshIntervalSec,
            () => {
                this._refresh();
                return GLib.SOURCE_CONTINUE;
            }
        );
    }

    disable() {
        if (this._timeoutId) {
            GLib.source_remove(this._timeoutId);
            this._timeoutId = null;
        }
        if (this._cancellable) {
            this._cancellable.cancel();
            this._cancellable = null;
        }
        this._indicator?.destroy();
        this._indicator = null;
        this._label = null;
        this._downItem = null;
        this._upItem = null;
        this._pingItem = null;
        this._resultItem = null;
        this._unitSwitch = null;
    }

    _buildMenu() {
        const menu = this._indicator.menu;

        this._ifaceItem = new PopupMenu.PopupMenuItem('Interface: ' + this._iface, {
            reactive: false,
            can_focus: false,
        });
        menu.addMenuItem(this._ifaceItem);

        this._downItem = new PopupMenu.PopupMenuItem('\u2193 Download: \u2014', {
            reactive: false,
            can_focus: false,
        });
        menu.addMenuItem(this._downItem);

        this._upItem = new PopupMenu.PopupMenuItem('\u2191 Upload: \u2014', {
            reactive: false,
            can_focus: false,
        });
        menu.addMenuItem(this._upItem);

        this._pingItem = new PopupMenu.PopupMenuItem(
            '\u26a1 Ping (' + CONFIG.defaultPingHost + '): \u2014',
            { reactive: false, can_focus: false }
        );
        menu.addMenuItem(this._pingItem);

        menu.addMenuItem(new PopupMenu.PopupSeparatorMenuItem());

        this._unitSwitch = new PopupMenu.PopupSwitchMenuItem(
            'Show Mbps (bits) instead of MB/s (bytes)',
            this._useMbps
        );
        this._unitSwitch.connect('toggled', (item, state) => {
            this._useMbps = state;
            this._updatePanelLabel();
            this._updateMenuStats();
        });
        menu.addMenuItem(this._unitSwitch);

        menu.addMenuItem(new PopupMenu.PopupSeparatorMenuItem());

        const entryItem = new PingEntryItem(target => this._onTestPing(target));
        menu.addMenuItem(entryItem);

        this._resultItem = new PopupMenu.PopupMenuItem('', {
            reactive: false,
            can_focus: false,
        });
        this._resultItem.label.set_style('white-space: normal;');
        menu.addMenuItem(this._resultItem);
    }

    _toggleUnits() {
        this._useMbps = !this._useMbps;
        if (this._unitSwitch)
            this._unitSwitch.setToggleState(this._useMbps);
        this._updatePanelLabel();
        this._updateMenuStats();
    }

    _onTestPing(target) {
        target = (target || '').trim();
        if (!isValidHost(target)) {
            this._resultItem.label.text = '"' + target + '" doesn\'t look like a valid domain or IP.';
            return;
        }

        this._resultItem.label.text = 'Pinging ' + target + ' \u2026';

        runPing(target, CONFIG.onDemandPingCount, CONFIG.pingTimeoutSec, this._cancellable, result => {
            if (!this._resultItem)
                return;

            if (!result.ok || result.received === 0) {
                this._resultItem.label.text =
                    target + ': unreachable (' + result.received + '/' + result.sent + ' received)' +
                    (result.error ? ' \u2014 ' + result.error : '');
                return;
            }

            const lossPct = result.sent > 0
                ? Math.round(100 * (result.sent - result.received) / result.sent)
                : 0;

            this._resultItem.label.text =
                target + ': sent ' + result.sent + ', received ' + result.received + ', loss ' + lossPct + '%\n' +
                'min/avg/max: ' + result.minMs.toFixed(1) + '/' + result.avgMs.toFixed(1) + '/' + result.maxMs.toFixed(1) + ' ms';
        });
    }

    _updatePanelLabel() {
        if (!this._label)
            return;
        const downStr = formatSpeed(this._smoothedDown ?? 0, this._useMbps);
        const upStr = formatSpeed(this._smoothedUp ?? 0, this._useMbps);
        const pingStr = formatPing(this._bgPingMs, this._bgPingOk);

        this._label.text = downStr + ' \u2193 | ' + upStr + ' \u2191 | ' + pingStr + ' \u26a1';
    }

    _updateMenuStats() {
        if (this._downItem)
            this._downItem.label.text = '\u2193 Download: ' + formatSpeed(this._smoothedDown ?? 0, this._useMbps);
        if (this._upItem)
            this._upItem.label.text = '\u2191 Upload: ' + formatSpeed(this._smoothedUp ?? 0, this._useMbps);
        if (this._pingItem) {
            this._pingItem.label.text = this._bgPingOk
                ? '\u26a1 Ping (' + CONFIG.defaultPingHost + '): ' + this._bgPingMs.toFixed(0) + ' ms'
                : '\u26a1 Ping (' + CONFIG.defaultPingHost + '): timeout';
        }
    }

    _refresh() {
        const cur = readIfaceStats(this._iface);
        const rates = computeRatesBytesPerSec(this._prevSample, cur);
        this._prevSample = cur;

        const alpha = CONFIG.smoothingAlpha;
        this._smoothedDown = (this._smoothedDown == null)
            ? rates.down : alpha * rates.down + (1 - alpha) * this._smoothedDown;
        this._smoothedUp = (this._smoothedUp == null)
            ? rates.up : alpha * rates.up + (1 - alpha) * this._smoothedUp;

        this._updatePanelLabel();
        this._updateMenuStats();

        runPing(CONFIG.defaultPingHost, 1, CONFIG.pingTimeoutSec, this._cancellable, result => {
            if (!this._indicator)
                return;
            this._bgPingOk = result.ok;
            this._bgPingMs = result.ok ? result.lastMs : -1;
            this._updatePanelLabel();
            this._updateMenuStats();
        });
    }
}
