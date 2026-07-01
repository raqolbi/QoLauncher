package viewer

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"os"
	"strconv"
)

func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/logs", http.StatusFound)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
		"viewer": "enabled",
	})
}

func (s *Server) handleLogsList(w http.ResponseWriter, r *http.Request) {
	logs, err := ListLogs(s.dir)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": logs})
}

func (s *Server) handleLogsListHTML(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("format") == "json" {
		s.handleLogsList(w, r)
		return
	}

	logs, err := ListLogs(s.dir)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, htmlPageStart("QoLauncher Logs"))
	fmt.Fprint(w, `<h1>Log Files</h1><table><thead><tr><th>Date</th><th>Size</th><th>Actions</th></tr></thead><tbody>`)
	for _, log := range logs {
		size := formatBytes(log.SizeBytes)
		fmt.Fprintf(w,
			`<tr><td>%s</td><td>%s</td><td><a href="/logs/%s?format=html">View</a> · <a href="/logs/%s/download">Download</a></td></tr>`,
			html.EscapeString(log.Date),
			html.EscapeString(size),
			html.EscapeString(log.Name),
			html.EscapeString(log.Name),
		)
	}
	fmt.Fprint(w, `</tbody></table>`)
	fmt.Fprint(w, htmlPageEnd())
}

func (s *Server) handleLogView(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("filename")
	path, err := LogFilePath(s.dir, name)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	format := r.URL.Query().Get("format")
	if format == "" {
		format = "html"
	}

	if format == "raw" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(data)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, htmlPageStart(html.EscapeString(name)))
	fmt.Fprintf(w, `<p><a href="/logs">← Back</a> · <a href="/logs/%s/download">Download</a></p>`, html.EscapeString(name))
	fmt.Fprint(w, `<pre>`)
	fmt.Fprint(w, html.EscapeString(string(data)))
	fmt.Fprint(w, `</pre>`)
	fmt.Fprint(w, htmlPageEnd())
}

func (s *Server) handleLogDownload(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("filename")
	path, err := LogFilePath(s.dir, name)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer f.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	_, _ = io.Copy(w, f)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return strconv.FormatInt(n, 10) + " B"
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func htmlPageStart(title string) string {
	return `<!DOCTYPE html><html><head><meta charset="utf-8"><title>` + title + `</title>` +
		`<style>
body{font-family:system-ui,sans-serif;margin:2rem;line-height:1.5;color:#222;background:#fafafa}
table{border-collapse:collapse;width:100%;max-width:960px;background:#fff}
th,td{border:1px solid #ddd;padding:.5rem .75rem;text-align:left}
th{background:#f0f0f0}
pre{background:#111;color:#eee;padding:1rem;overflow:auto;max-width:100%}
a{color:#0366d6}
</style></head><body>`
}

func htmlPageEnd() string {
	return `</body></html>`
}
