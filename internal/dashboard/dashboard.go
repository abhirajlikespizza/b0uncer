package dashboard

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"

	"github.com/b0uncer/b0uncer/internal/logger"
)

var indexHTML []byte

func Start(html []byte) {
	indexHTML = html
	go serve()
}

func serve() {
	var ln net.Listener
	var err error
	port := 3456
	for _, p := range []int{3456, 3457, 3458} {
		ln, err = net.Listen("tcp", fmt.Sprintf(":%d", p))
		if err == nil {
			port = p
			break
		}
	}
	if ln == nil {
		fmt.Fprintln(os.Stderr, "b0uncer: dashboard could not bind to any port")
		return
	}
	fmt.Fprintf(os.Stderr, "b0uncer dashboard: http://localhost:%d\n", port)

	mux := http.NewServeMux()
	mux.HandleFunc("/", handleIndex)
	mux.HandleFunc("/api/events", handleEvents)
	mux.HandleFunc("/api/stats", handleStats)
	mux.HandleFunc("/api/clear", handleClear)

	http.Serve(ln, mux) //nolint:errcheck
}

func cors(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	cors(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML) //nolint:errcheck
}

func handleEvents(w http.ResponseWriter, r *http.Request) {
	cors(w)
	w.Header().Set("Content-Type", "application/json")
	events, err := logger.Recent(100)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(events) //nolint:errcheck
}

func handleStats(w http.ResponseWriter, r *http.Request) {
	cors(w)
	w.Header().Set("Content-Type", "application/json")
	stats, err := logger.GetStats()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(stats) //nolint:errcheck
}

func handleClear(w http.ResponseWriter, r *http.Request) {
	cors(w)
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := logger.Clear(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(`{"ok":true}`)) //nolint:errcheck
}
