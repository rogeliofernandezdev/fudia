# Flujo conversacional

## Inicio

El QR inicia WhatsApp con un texto que incluye FUDIA:<qr_token>. El token es hexadecimal de 32 caracteres y se procesa antes del LLM.

## Compra

1. Resolver la mesa.
2. Buscar productos mediante search_menu.
3. Agregar únicamente productId devueltos por Fudia.
4. Revalidar el producto al agregar.
5. Revisar carrito.
6. prepare_confirmation marca confirmación pendiente.
7. El cliente confirma explícitamente.
8. confirm_order solicita a foods-backend crear la orden.

## Invariantes

- Un nombre libre nunca se convierte directamente en ID.
- Un precio generado por el modelo nunca se persiste.
- Una confirmación inferida no crea pedidos.
- Un error de disponibilidad no borra el carrito.
