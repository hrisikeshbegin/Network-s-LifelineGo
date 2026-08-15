// Package web serves the local network dashboard: a JSON API exposing
// discovered devices and an auto-refreshing HTML page that displays them.
package web

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"

	"github.com/hrisikeshbegin/go-network-lifeline/internal/store"
)

// NewHandler returns an http.Handler serving the dashboard: a JSON API at
// /api/devices and an auto-refreshing HTML table at /.
func NewHandler(st store.Store) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/devices", devicesHandler(st))
	mux.HandleFunc("/", indexHandler)
	return mux
}

func devicesHandler(st store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		devices, err := st.ListDevices(r.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		sort.Slice(devices, func(i, j int) bool { return devices[i].IP < devices[j].IP })

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(devices); err != nil {
			log.Printf("web: encoding devices response: %v", err)
		}
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := w.Write([]byte(indexHTML)); err != nil {
		log.Printf("web: writing index page: %v", err)
	}
}

const indexHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Network Lifeline</title>
<style>
  :root {
    --page:       #0d0d0d;
    --surface:    #1a1a19;
    --surface-2:  #212120;
    --ink:        #ffffff;
    --ink-2:      #c3c2b7;
    --ink-muted:  #898781;
    --hairline:   #2c2c2a;
    --border:     rgba(255,255,255,0.10);
    --hover:      rgba(255,255,255,0.035);
    --good:       #0ca30c;
    --critical:   #d03b3b;
    --accent:     #3987e5;
  }

  * { box-sizing: border-box; }

  body {
    font-family: system-ui, -apple-system, "Segoe UI", sans-serif;
    margin: 0;
    background: var(--page);
    color: var(--ink);
    -webkit-font-smoothing: antialiased;
  }

  header {
    position: sticky;
    top: 0;
    z-index: 10;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.9rem 1.75rem;
    background: rgba(13,13,13,0.85);
    backdrop-filter: blur(6px);
    border-bottom: 1px solid var(--hairline);
  }

  .brand { display: flex; align-items: baseline; gap: 0.6rem; }
  .brand h1 { font-size: 1.05rem; font-weight: 600; margin: 0; letter-spacing: 0.01em; }
  .brand span { color: var(--ink-muted); font-size: 0.8rem; }

  .live {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    color: var(--ink-muted);
    font-size: 0.8rem;
  }
  .live-dot {
    width: 7px; height: 7px; border-radius: 50%;
    background: var(--good);
    box-shadow: 0 0 0 3px rgba(12,163,12,0.18);
    animation: pulse 2s ease-in-out infinite;
  }
  @keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.45; } }

  main { padding: 1.75rem; max-width: 1100px; margin: 0 auto; }

  .stats {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 1rem;
    margin-bottom: 1.75rem;
  }

  .stat-card {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    padding: 1rem 1.15rem;
  }
  .stat-label {
    display: flex;
    align-items: center;
    gap: 0.45rem;
    color: var(--ink-muted);
    font-size: 0.72rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    margin-bottom: 0.55rem;
  }
  .stat-value {
    font-size: 1.9rem;
    font-weight: 650;
    font-variant-numeric: tabular-nums;
    line-height: 1;
  }

  .dot {
    width: 8px; height: 8px; border-radius: 50%; flex: none;
    display: inline-block;
  }
  .dot-online  { background: var(--good); box-shadow: 0 0 0 2.5px rgba(12,163,12,0.16); }
  .dot-offline { background: var(--critical); box-shadow: 0 0 0 2.5px rgba(208,59,59,0.16); }
  .dot-total   { background: var(--accent); box-shadow: 0 0 0 2.5px rgba(57,135,229,0.16); }

  .panel {
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: 10px;
    overflow: hidden;
  }

  table { border-collapse: collapse; width: 100%; }
  thead th {
    background: var(--surface-2);
    text-align: left;
    padding: 0.65rem 1.1rem;
    color: var(--ink-muted);
    font-weight: 600;
    text-transform: uppercase;
    font-size: 0.7rem;
    letter-spacing: 0.05em;
    border-bottom: 1px solid var(--hairline);
  }
  tbody td {
    padding: 0.7rem 1.1rem;
    border-bottom: 1px solid var(--hairline);
    font-size: 0.9rem;
    color: var(--ink-2);
  }
  tbody tr:last-child td { border-bottom: none; }
  tbody tr:hover td { background: var(--hover); }

  td.ip { color: var(--ink); font-variant-numeric: tabular-nums; }
  td.last-seen { font-variant-numeric: tabular-nums; color: var(--ink-muted); }

  .status-cell { display: flex; align-items: center; gap: 0.5rem; }
  .status-online  { color: var(--good); font-weight: 600; }
  .status-offline { color: var(--critical); font-weight: 600; }

  .empty {
    padding: 3rem 1rem;
    text-align: center;
    color: var(--ink-muted);
    font-size: 0.9rem;
  }
</style>
</head>
<body>
<header>
  <div class="brand">
    <h1>Network Lifeline</h1>
    <span>local device dashboard</span>
  </div>
  <div class="live">
    <span class="live-dot"></span>
    <span id="updated">loading&hellip;</span>
  </div>
</header>

<main>
  <div class="stats">
    <div class="stat-card">
      <div class="stat-label"><span class="dot dot-total"></span>Total devices</div>
      <div class="stat-value" id="stat-total">—</div>
    </div>
    <div class="stat-card">
      <div class="stat-label"><span class="dot dot-online"></span>Online</div>
      <div class="stat-value" id="stat-online">—</div>
    </div>
    <div class="stat-card">
      <div class="stat-label"><span class="dot dot-offline"></span>Offline</div>
      <div class="stat-value" id="stat-offline">—</div>
    </div>
  </div>

  <div class="panel">
    <table>
      <thead><tr><th>IP</th><th>Hostname</th><th>Status</th><th>Last Seen</th></tr></thead>
      <tbody id="devices"></tbody>
    </table>
    <div class="empty" id="empty" hidden>No devices discovered yet&hellip;</div>
  </div>
</main>

<script>
function escapeHTML(s) {
  const div = document.createElement('div');
  div.textContent = s == null ? '' : s;
  return div.innerHTML;
}

async function refresh() {
  const updated = document.getElementById('updated');
  try {
    const res = await fetch('/api/devices', { cache: 'no-store' });
    if (!res.ok) throw new Error('HTTP ' + res.status);
    const devices = await res.json() || [];

    const tbody = document.getElementById('devices');
    const empty = document.getElementById('empty');
    tbody.innerHTML = '';

    let online = 0;
    for (const d of devices) {
      if (d.Status === 'online') online++;
      const statusClass = d.Status === 'online' ? 'status-online' : 'status-offline';
      const dotClass = d.Status === 'online' ? 'dot-online' : 'dot-offline';
      const lastSeen = d.LastSeen ? new Date(d.LastSeen).toLocaleString() : '—';
      const tr = document.createElement('tr');
      tr.innerHTML =
        '<td class="ip">' + escapeHTML(d.IP) + '</td>' +
        '<td>' + escapeHTML(d.Hostname || '—') + '</td>' +
        '<td><span class="status-cell"><span class="dot ' + dotClass + '"></span>' +
          '<span class="' + statusClass + '">' + escapeHTML(d.Status) + '</span></span></td>' +
        '<td class="last-seen">' + escapeHTML(lastSeen) + '</td>';
      tbody.appendChild(tr);
    }

    empty.hidden = devices.length !== 0;
    document.getElementById('stat-total').textContent = devices.length;
    document.getElementById('stat-online').textContent = online;
    document.getElementById('stat-offline').textContent = devices.length - online;

    updated.textContent = 'updated ' + new Date().toLocaleTimeString();
  } catch (err) {
    updated.textContent = 'refresh failed: ' + err.message;
  }
}

refresh();
setInterval(refresh, 5000);
</script>
</body>
</html>
`
