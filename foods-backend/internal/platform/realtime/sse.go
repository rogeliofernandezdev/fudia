package realtime

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/foods-platform/foods-backend/internal/platform/httpapi"
)

// Path es la ruta del canal en vivo. cmd/api la excluye del plazo de 30 s por
// petición y del aviso de petición lenta: es una conexión que queda abierta.
const Path = "/v1/admin/events"

// Events transmite por Server-Sent Events los temas que cambian en el local de
// la sesión. Agrupa los avisos que llegan juntos (Coalesce) en un solo evento
// «change», envía un comentario cada Heartbeat para que proxies y balanceadores
// no corten la conexión y la cierra tras MaxDuration o al apagar la API (Done);
// EventSource reconecta solo.
type Events struct {
	Hub         *Hub
	Done        <-chan struct{}
	Coalesce    time.Duration
	Heartbeat   time.Duration
	MaxDuration time.Duration
}

func (e Events) Serve(w http.ResponseWriter, r *http.Request) {
	s, ok := httpapi.Scope(w, r)
	if !ok {
		return
	}
	if s.OrganizationID == "" || s.LocationID == "" {
		httpapi.Fail(w, 403, "location_required", "Selecciona un local para continuar.")
		return
	}
	if e.Hub == nil || !e.Hub.Ready() {
		// Sin escucha a PostgreSQL no hay avisos: el navegador sigue con su polling.
		httpapi.Fail(w, 503, "live_unavailable", "Las actualizaciones en vivo no están disponibles.")
		return
	}
	controller := http.NewResponseController(w)
	// El servidor fija un plazo de escritura por petición; este flujo no termina.
	_ = controller.SetWriteDeadline(time.Time{})
	topics, cancel := e.Hub.Subscribe(s.OrganizationID, s.LocationID)
	defer cancel()

	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache, no-transform")
	header.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	write := func(chunk string) bool {
		if _, err := fmt.Fprint(w, chunk); err != nil {
			return false
		}
		return controller.Flush() == nil
	}
	if !write("retry: 5000\n\nevent: ready\ndata: {}\n\n") {
		return
	}

	coalesce, heartbeat, maxDuration := e.Coalesce, e.Heartbeat, e.MaxDuration
	if coalesce <= 0 {
		coalesce = 250 * time.Millisecond
	}
	if heartbeat <= 0 {
		heartbeat = 25 * time.Second
	}
	if maxDuration <= 0 {
		maxDuration = 15 * time.Minute
	}
	ping := time.NewTicker(heartbeat)
	defer ping.Stop()
	expire := time.NewTimer(maxDuration)
	defer expire.Stop()
	pending := map[string]bool{}
	var flush <-chan time.Time
	for {
		select {
		case <-r.Context().Done():
			return
		case <-e.Done:
			return
		case <-expire.C:
			return
		case topic, open := <-topics:
			if !open {
				return
			}
			pending[topic] = true
			if flush == nil {
				flush = time.After(coalesce)
			}
		case <-flush:
			flush = nil
			names := make([]string, 0, len(pending))
			for topic := range pending {
				names = append(names, topic)
			}
			sort.Strings(names)
			pending = map[string]bool{}
			data, _ := json.Marshal(map[string][]string{"topics": names})
			if !write("event: change\ndata: " + string(data) + "\n\n") {
				return
			}
		case <-ping.C:
			if !write(": ping\n\n") {
				return
			}
		}
	}
}
