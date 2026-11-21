package postgres

import (
	"strings"
	"testing"

	"github.com/foods-platform/foods-backend/internal/modules/paydetect/domain"
)

func TestNotificationWhereSearch(t *testing.T) {
	where, args := notificationWhere(scope{}, domain.NotificationFilter{Query: "25,5"})
	if !strings.Contains(where, "translate(lower(n.sender_name),'áéíóúüñ','aeiouun') LIKE '%'||translate(lower($3),'áéíóúüñ','aeiouun')||'%'") || !strings.Contains(where, "OR n.amount::text LIKE $4||'%'") {
		t.Fatalf("búsqueda: %s", where)
	}
	if len(args) != 4 || args[3] != "25.5" {
		t.Fatalf("args: %#v", args)
	}
	where, args = notificationWhere(scope{}, domain.NotificationFilter{Query: "Rosa"})
	if strings.Contains(where, "n.amount") || len(args) != 3 {
		t.Fatalf("un nombre no busca por monto: %s %#v", where, args)
	}
}
