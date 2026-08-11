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
<title>Network Lifeline</title>
<style>
  body { font-family: system-ui, sans-serif; margin: 2rem; background: #0f1117; color: #e6e6e6; }
  h1 { font-size: 1.25rem; margin-bottom: 0.25rem; }
  #updated { color: #888; font-size: 0.85rem; margin-bottom: 1rem; }
  table { border-collapse: collapse; width: 100%; max-width: 900px; }
  th, td { text-align: left; padding: 0.5rem 0.75rem; border-bottom: 1px solid #2a2d36; }
  th { color: #999; font-weight: 600; text-transform: uppercase; font-size: 0.75rem; letter-spacing: 0.03em; }
  tr:hover td { background: #171a21; }
  .status-online { color: #4ade80; font-weight: 600; }
  .status-offline { color: #f87171; font-weight: 600; }
</style>
</head>
<body>
<h1>Network Lifeline</h1>
<div id="updated">loading&hellip;</div>
<table>
  <thead><tr><th>IP</th><th>Hostname</th><th>Status</th><th>Last Seen</th></tr></thead>
  <tbody id="devices"></tbody>
</table>
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
    const devices = await res.json();

    const tbody = document.getElementById('devices');
    tbody.innerHTML = '';
    for (const d of (devices || [])) {
      const statusClass = d.Status === 'online' ? 'status-online' : 'status-offline';
      const lastSeen = d.LastSeen ? new Date(d.LastSeen).toLocaleString() : '—';
      const tr = document.createElement('tr');
      tr.innerHTML =
        '<td>' + escapeHTML(d.IP) + '</td>' +
        '<td>' + escapeHTML(d.Hostname || '—') + '</td>' +
        '<td class="' + statusClass + '">' + escapeHTML(d.Status) + '</td>' +
        '<td>' + escapeHTML(lastSeen) + '</td>';
      tbody.appendChild(tr);
    }

    updated.textContent = (devices || []).length + ' device(s) — last updated ' + new Date().toLocaleTimeString();
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
