package realtime

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

func TestParseChange(t *testing.T) {
	c, ok := ParseChange("org:loc:orders")
	if !ok || c != (Change{OrganizationID: "org", LocationID: "loc", Topic: "orders"}) {
		t.Fatalf("unexpected change %#v %v", c, ok)
	}
	if c, ok = ParseChange("org::catalog"); !ok || c.LocationID != "" || c.Topic != "catalog" {
		t.Fatalf("company-wide change must parse, got %#v", c)
	}
	for _, bad := range []string{"", "org:loc", ":loc:orders", "org:loc:", "a:b:c:d"} {
		if _, ok := ParseChange(bad); ok {
			t.Fatalf("%q must be rejected", bad)
		}
	}
}

func TestHubDeliversOnlyToItsCompanyAndLocation(t *testing.T) {
	hub := NewHub()
	hub.SetReady(true)
	mine, cancelMine := hub.Subscribe("org", "loc")
	defer cancelMine()
	other, cancelOther := hub.Subscribe("org", "other")
	defer cancelOther()
	foreign, cancelForeign := hub.Subscribe("org-b", "loc")
	defer cancelForeign()

	hub.Publish(Change{OrganizationID: "org", LocationID: "loc", Topic: "orders"})
	hub.Publish(Change{OrganizationID: "org", Topic: "catalog"})
	if got := []string{<-mine, <-mine}; got[0] != "orders" || got[1] != "catalog" {
		t.Fatalf("unexpected topics %v", got)
	}
	if got := <-other; got != "catalog" {
		t.Fatalf("company-wide change must reach every location, got %q", got)
	}
	select {
	case topic := <-other:
		t.Fatalf("another location must not receive %q", topic)
	case topic := <-foreign:
		t.Fatalf("another company must not receive %q", topic)
	default:
	}
}

func TestHubClosesSubscriptionsWhenListenerIsLost(t *testing.T) {
	hub := NewHub()
	if _, open := <-mustSubscribe(hub); open {
		t.Fatal("subscribing without a listener must return a closed channel")
	}
	hub.SetReady(true)
	topics, cancel := hub.Subscribe("org", "loc")
	defer cancel()
	hub.SetReady(false)
	if _, open := <-topics; open {
		t.Fatal("losing the listener must close subscriptions")
	}
	if hub.Subscribers() != 0 {
		t.Fatalf("expected no subscribers, got %d", hub.Subscribers())
	}
}

func mustSubscribe(hub *Hub) <-chan string {
	topics, _ := hub.Subscribe("org", "loc")
	return topics
}

func TestEventsStreamsCoalescedChanges(t *testing.T) {
	hub := NewHub()
	hub.SetReady(true)
	events := Events{Hub: hub, Coalesce: 20 * time.Millisecond, Heartbeat: time.Hour}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := requestscope.With(r.Context(), requestscope.Scope{OrganizationID: "org", LocationID: "loc"})
		events.Serve(w, r.WithContext(ctx))
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 || !strings.HasPrefix(response.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("unexpected response %d %s", response.StatusCode, response.Header.Get("Content-Type"))
	}
	reader := bufio.NewReader(response.Body)
	readEvent := func() string {
		var lines []string
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				t.Fatalf("read stream: %v", err)
			}
			line = strings.TrimRight(line, "\n")
			if line == "" {
				if len(lines) > 0 {
					return strings.Join(lines, "|")
				}
				continue
			}
			lines = append(lines, line)
		}
	}
	if first := readEvent(); first != "retry: 5000" {
		t.Fatalf("unexpected first block %q", first)
	}
	if ready := readEvent(); ready != "event: ready|data: {}" {
		t.Fatalf("unexpected ready event %q", ready)
	}
	for hub.Subscribers() == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	hub.Publish(Change{OrganizationID: "org", LocationID: "loc", Topic: "orders"})
	hub.Publish(Change{OrganizationID: "org", LocationID: "loc", Topic: "kitchen"})
	hub.Publish(Change{OrganizationID: "org", LocationID: "loc", Topic: "orders"})
	if change := readEvent(); change != `event: change|data: {"topics":["kitchen","orders"]}` {
		t.Fatalf("changes must arrive grouped once, got %q", change)
	}
}

func TestEventsRejectsWithoutListener(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, Path, nil)
	request = request.WithContext(requestscope.With(request.Context(), requestscope.Scope{OrganizationID: "org", LocationID: "loc"}))
	Events{Hub: NewHub()}.Serve(recorder, request)
	if recorder.Code != 503 || !strings.Contains(recorder.Body.String(), "live_unavailable") {
		t.Fatalf("expected 503 live_unavailable, got %d %s", recorder.Code, recorder.Body.String())
	}
}
