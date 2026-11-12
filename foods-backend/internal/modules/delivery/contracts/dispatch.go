package contracts

import (
	"context"

	"github.com/foods-platform/foods-backend/internal/platform/requestscope"
)

// OrderDispatch recibe de Orders los cambios de un pedido que afectan su
// despacho a las centrales de Delivery Fudia: confirmación, cancelación y
// preparación terminada. Es de mejor esfuerzo y se llama después del commit;
// un barrido periódico corrige lo que no se pudo aplicar.
type OrderDispatch interface {
	OrderChanged(ctx context.Context, s requestscope.Scope, orderID string)
}
