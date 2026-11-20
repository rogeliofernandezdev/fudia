// Package realtime reparte a las pantallas abiertas los avisos de cambio que
// PostgreSQL emite con pg_notify (migración 000138). Un aviso solo dice qué tema
// cambió en qué empresa y local; los datos se vuelven a leer por la API.
package realtime

import (
	"strings"
	"sync"
	"sync/atomic"
)

// Channel es el canal de pg_notify que escuchan las instancias de la API.
const Channel = "foods_live"

// Change es un aviso: el tema (orders, kitchen, payments…) cambió en el local.
// Un LocationID vacío afecta a todos los locales de la empresa.
type Change struct {
	OrganizationID, LocationID, Topic string
}

// ParseChange interpreta la carga «empresa:local:tema» de notify_live_change.
func ParseChange(payload string) (Change, bool) {
	parts := strings.Split(payload, ":")
	if len(parts) != 3 || parts[0] == "" || parts[2] == "" {
		return Change{}, false
	}
	return Change{OrganizationID: parts[0], LocationID: parts[1], Topic: parts[2]}, true
}

type subscriber struct {
	organizationID, locationID string
	topics                     chan string
}

// Hub mantiene los suscriptores por empresa y local. Publicar nunca bloquea: si
// un suscriptor lento llena su búfer, el aviso se descarta y la pantalla lo
// recupera con su consulta de respaldo.
type Hub struct {
	mu    sync.RWMutex
	subs  map[*subscriber]struct{}
	ready atomic.Bool
}

func NewHub() *Hub { return &Hub{subs: map[*subscriber]struct{}{}} }

// Ready indica si hay una conexión escuchando a PostgreSQL. Sin ella, la API
// rechaza nuevas suscripciones para que el navegador siga con su polling normal.
func (h *Hub) Ready() bool { return h.ready.Load() }

// SetReady cambia el estado de la escucha. Al perderla se cierran todas las
// suscripciones: los navegadores vuelven al polling hasta reconectar.
func (h *Hub) SetReady(ready bool) {
	if h.ready.Swap(ready) == ready || ready {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for s := range h.subs {
		close(s.topics)
		delete(h.subs, s)
	}
}

// Subscribe devuelve los temas que cambian en el local y la función para dejar
// de escuchar. El canal se cierra si se pierde la conexión con PostgreSQL.
func (h *Hub) Subscribe(organizationID, locationID string) (<-chan string, func()) {
	s := &subscriber{organizationID: organizationID, locationID: locationID, topics: make(chan string, 64)}
	h.mu.Lock()
	if !h.ready.Load() {
		// La escucha se perdió entre la comprobación y la suscripción.
		h.mu.Unlock()
		close(s.topics)
		return s.topics, func() {}
	}
	h.subs[s] = struct{}{}
	h.mu.Unlock()
	return s.topics, func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if _, ok := h.subs[s]; ok {
			delete(h.subs, s)
			close(s.topics)
		}
	}
}

// Publish entrega el aviso a los suscriptores de su empresa y local.
func (h *Hub) Publish(c Change) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	for s := range h.subs {
		if s.organizationID != c.OrganizationID || (c.LocationID != "" && s.locationID != c.LocationID) {
			continue
		}
		select {
		case s.topics <- c.Topic:
		default:
		}
	}
}

// Subscribers cuenta las pantallas conectadas (para registros y pruebas).
func (h *Hub) Subscribers() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.subs)
}
