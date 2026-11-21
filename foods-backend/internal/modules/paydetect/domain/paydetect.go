package domain

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Fudia Pay: dispositivos del local que detectan notificaciones de pagos
// recibidos por Yape y Plin y las reportan a Caja. El backend resuelve la
// empresa y el local del dispositivo; una notificación detectada nunca
// confirma un cobro por sí sola.

const (
	// Plataformas del contrato de dispositivos: el backend y los contratos
	// son independientes del sistema operativo; en esta etapa solo hay Android.
	PlatformAndroid = "ANDROID"
	PlatformIOS     = "IOS"

	// Proveedores con parser propio en la aplicación Android. El backend solo
	// acepta eventos ya normalizados por esos parsers.
	ProviderYape = "yape"
	ProviderPlin = "plin"

	// Estados de una notificación, avanzados solo por Caja.
	NotificationDetected   = "detected"
	NotificationAssociated = "associated"
	NotificationConfirmed  = "confirmed"
	NotificationDismissed  = "dismissed"

	// PairingCodeMinutes es la vigencia del código temporal de vinculación.
	PairingCodeMinutes = 10
	// PairingCodeRetentionDays: los códigos vencidos se borran pasado este plazo.
	PairingCodeRetentionDays = 1
	// DeviceNameLimit acota el nombre reconocible del dispositivo.
	DeviceNameLimit = 60
	// SenderNameLimit acota el remitente reportado por la notificación.
	SenderNameLimit = 80
	// EventKeyMin y EventKeyMax acotan la identidad estable del evento,
	// que la app construye a partir de la notificación de origen.
	EventKeyMin        = 8
	EventKeyMax        = 200
	OperationNumberMax = 64
)

// Device es un teléfono del local vinculado a Fudia Pay. El token nunca vive
// aquí: solo los últimos dígitos para reconocerlo.
type Device struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Platform    string `json:"platform"`
	Model       string `json:"model,omitempty"`
	TokenSuffix string `json:"tokenSuffix"`
	Active      bool   `json:"active"`
	LastSeenAt  string `json:"lastSeenAt,omitempty"`
	CreatedAt   string `json:"createdAt"`
	RevokedAt   string `json:"revokedAt,omitempty"`
}

// ValidPlatform verifica la plataforma declarada por el dispositivo.
func ValidPlatform(platform string) bool {
	return platform == PlatformAndroid || platform == PlatformIOS
}

// PairingCode es el código temporal de un solo uso que Caja muestra como QR.
// Caduca solo y nunca contiene credenciales permanentes.
type PairingCode struct {
	ID             string
	OrganizationID string
	LocationID     string
	Code           string
	CreatedBy      string
	ExpiresAt      time.Time
	UsedAt         *time.Time
	DeviceID       string
	CreatedAt      time.Time
	// ScannedAt: la app leyó el QR y está canjeando el código.
	ScannedAt *time.Time
}

// Used dice si el código ya se canjeó por un dispositivo.
func (c PairingCode) Used() bool {
	return c.UsedAt != nil
}

// Expired dice si el código ya no puede canjearse.
func (c PairingCode) Expired(now time.Time) bool {
	return now.After(c.ExpiresAt)
}

// Redeemable dice si el código puede activar un dispositivo ahora mismo.
func (c PairingCode) Redeemable(now time.Time) bool {
	return !c.Used() && !c.Expired(now)
}

// Event es lo que el dispositivo reporta en su solicitud de sincronización.
type Event struct {
	Provider        string    `json:"provider"`
	EventKey        string    `json:"eventKey"`
	OperationNumber string    `json:"operationNumber,omitempty"`
	SenderName      string    `json:"senderName"`
	Amount          float64   `json:"amount"`
	OccurredAt      time.Time `json:"occurredAt"`
}

// Notification es un pago detectado, tal como Caja lo lee. El monto viaja como
// texto decimal, como el resto de montos del sistema.
type Notification struct {
	ID              string `json:"id"`
	DeviceID        string `json:"deviceId"`
	DeviceName      string `json:"deviceName"`
	Provider        string `json:"provider"`
	EventKey        string `json:"eventKey"`
	OperationNumber string `json:"operationNumber,omitempty"`
	SenderName      string `json:"senderName"`
	Amount          string `json:"amount"`
	Currency        string `json:"currency"`
	OccurredAt      string `json:"occurredAt"`
	Status          string `json:"status"`
	OrderID         string `json:"orderId,omitempty"`
	PaymentID       string `json:"paymentId,omitempty"`
	ConfirmedBy     string `json:"confirmedBy,omitempty"`
	ConfirmedAt     string `json:"confirmedAt,omitempty"`
	CreatedAt       string `json:"createdAt"`
}

// NormalizeEvent valida y normaliza un evento reportado por un dispositivo.
// Si el contenido no alcanza la confianza mínima para registrarse como pago
// detectado, devuelve el error que la app debe corregir; el backend no
// infiere un pago a partir de datos ambiguos.
var operationNumberPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]{0,63}$`)

func NormalizeEvent(event Event) (Event, error) {
	provider := strings.ToLower(strings.TrimSpace(event.Provider))
	if provider != ProviderYape && provider != ProviderPlin {
		return Event{}, fmt.Errorf("solo se aceptan notificaciones de Yape y Plin")
	}
	key := strings.TrimSpace(event.EventKey)
	if length := len(key); length < EventKeyMin || length > EventKeyMax {
		return Event{}, fmt.Errorf("la notificación no tiene una identidad estable para deduplicar")
	}
	if !(event.Amount > 0) || event.Amount >= 1e7 {
		return Event{}, fmt.Errorf("el monto de la notificación no es válido")
	}
	sender := []rune(strings.TrimSpace(event.SenderName))
	operationNumber := strings.TrimSpace(event.OperationNumber)
	if operationNumber != "" && !operationNumberPattern.MatchString(operationNumber) {
		return Event{}, fmt.Errorf("el número de operación debe tener hasta %d letras, números o guiones", OperationNumberMax)
	}
	if length := len(sender); length > SenderNameLimit {
		sender = sender[:SenderNameLimit]
	}
	return Event{
		Provider:        provider,
		EventKey:        key,
		OperationNumber: operationNumber,
		SenderName:      string(sender),
		Amount:          event.Amount,
		OccurredAt:      event.OccurredAt.UTC().Truncate(time.Second),
	}, nil
}

// FormatAmount deja el monto como texto decimal de dos decimales.
func FormatAmount(amount float64) string {
	return strconv.FormatFloat(amount, 'f', 2, 64)
}

// DeviceSession es lo que el backend resuelve desde el token del dispositivo:
// su tenencia y la moneda del restaurante. El dispositivo nunca declara estos
// datos; el backend los deriva de la vinculación.
type DeviceSession struct {
	DeviceID         string
	DeviceName       string
	OrganizationID   string
	OrganizationName string
	LocationID       string
	LocationName     string
	Currency         string
	Active           bool
	RevokedAt        *time.Time
}

// Usable dice si el dispositivo puede reportar notificaciones.
func (d DeviceSession) Usable() bool { return d.Active && d.RevokedAt == nil }

// PairInput es lo que la app declara al activar la vinculación: la plataforma
// y el modelo del teléfono. Ni empresa ni local ni identificadores: el backend
// los resuelve desde el código.
type PairInput struct {
	Platform string `json:"platform"`
	Model    string `json:"model"`
}

// Activation es la respuesta del canje del código: el token en claro se
// entrega una sola vez, aquí, y nunca se vuelve a leer del backend.
type Activation struct {
	Device           Device `json:"device"`
	Token            string `json:"token"`
	OrganizationName string `json:"organizationName"`
	LocationName     string `json:"locationName"`
}

// NotificationFilter acota el listado que Caja consulta.
type NotificationFilter struct {
	Status   string
	Provider string
	// Query busca por remitente (contiene, sin distinguir tildes ni
	// mayúsculas) o por monto (empieza con lo escrito). Las billeteras no
	// envían número de operación de forma fiable, así que no se busca por él.
	Query string
	Limit int
	// Offset salta las filas de las páginas anteriores (página 1: 0).
	Offset int
}

// Device statuses that filter the list of a location: active ones report
// payments; revoked ones remain as history.
const (
	DeviceStatusActive  = "active"
	DeviceStatusRevoked = "revoked"
)

// DeviceFilter pagina los dispositivos de un local por estado (vacío: todos).
type DeviceFilter struct {
	Status string
	Limit  int
	Offset int
}

// DevicePage es una página de dispositivos con el total que cumple el filtro.
type DevicePage struct {
	Items []Device `json:"items"`
	Total int      `json:"total"`
}

// NotificationPage es una página de pagos detectados con el total que cumple
// los filtros, para paginar en Caja sin traer el historial completo.
type NotificationPage struct {
	Items []Notification `json:"items"`
	Total int            `json:"total"`
}

// EventOutcome es el resultado por evento de un reporte: registrado ahora o
// duplicado (la idempotencia lo conserva tal cual).
type EventOutcome struct {
	EventKey string `json:"eventKey"`
	Status   string `json:"status"` // registered | duplicate
}

// ReportResult resume la sincronización de un dispositivo.
type ReportResult struct {
	Registered int            `json:"registered"`
	Duplicates int            `json:"duplicates"`
	Events     []EventOutcome `json:"events"`
}

// PairingDraft es el código generado por Caja antes de mostrarse como QR. El
// id permite a Caja seguir el estado del código sin exponerlo de nuevo.
type PairingDraft struct {
	ID        string `json:"id"`
	Code      string `json:"code"`
	ExpiresAt string `json:"expiresAt"`
}

// Estados de un código de vinculación, como los ve Caja mientras muestra el QR.
const (
	PairingPending = "pending" // esperando que la app lo escanee
	PairingScanned = "scanned" // la app lo leyó y está activando el dispositivo
	PairingLinked  = "linked"  // el dispositivo quedó vinculado
	PairingExpired = "expired" // venció sin usarse
)

// PairingScanModelLimit acota el modelo que la app declara al escanear.
const PairingScanModelLimit = 120

// PairingState es lo que Caja lee del código que está mostrando.
type PairingState struct {
	ExpiresAt  time.Time
	UsedAt     *time.Time
	ScannedAt  *time.Time
	Model      string
	DeviceName string
}

// PairingStatus es la respuesta para Caja: el estado y, cuando corresponde, el
// modelo que escaneó o el nombre del dispositivo vinculado.
type PairingStatus struct {
	Status     string `json:"status"`
	Model      string `json:"model,omitempty"`
	DeviceName string `json:"deviceName,omitempty"`
	ExpiresAt  string `json:"expiresAt"`
}

// StatusOf decide el estado con el reloj de la aplicación: vinculado gana
// siempre; un código escaneado sigue «escaneado» hasta que vence.
func (s PairingState) StatusOf(now time.Time) PairingStatus {
	out := PairingStatus{ExpiresAt: s.ExpiresAt.Format(time.RFC3339)}
	switch {
	case s.UsedAt != nil:
		out.Status, out.DeviceName = PairingLinked, s.DeviceName
	case now.After(s.ExpiresAt):
		out.Status = PairingExpired
	case s.ScannedAt != nil:
		out.Status, out.Model = PairingScanned, s.Model
	default:
		out.Status = PairingPending
	}
	return out
}
