# Wizard de venta

## Propósito

El recorrido de venta debe ser visible, continuo y consistente desde la toma del
pedido hasta la liberación de la mesa. Las pantallas comparten contexto,
progreso, geometría y jerarquía de acciones.

## Recorrido

1. **Pedido**: pedidos simultáneos, categorías, catálogo, cantidades y resumen.
2. **Personalizar**: modificadores obligatorios, extras, exclusiones y notas.
3. **Pago**: medio, división, importe recibido, vuelto y confirmación.
4. **Comprobante**: boleta o factura, identificación, impresión y envío digital.

## Navegación

- Caja abre Pedido mediante Nueva venta.
- Una cuenta pendiente abre Pago.
- Editar un producto abre Personalizar.
- Confirmar el pago abre Comprobante.
- Finalizar libera la mesa y vuelve a Mesas.
- La ubicación actual permanece visible con aria-current. La ruta por sí sola
  no demuestra que un pago, guardado o paso anterior se haya completado.
- No depender del botón atrás del navegador ni esconder etapas en enlaces pequeños.

## Invariantes visuales

- Mostrar siempre mesa o canal y correlativo.
- Usar texto, número o icono para distinguir actual, completado y pendiente.
- Mantener una sola acción primaria por paso.
- Las acciones secundarias no compiten en color con la acción primaria.
- En escritorio, contenido y resumen permanecen en dos columnas cuando sean legibles.
- En móvil, cada etapa utiliza una columna, sin paneles fijos ni desbordamiento.
- Montos, tiempos, cantidades y correlativos usan cifras tabulares.
- Todos los controles táctiles miden al menos 44 por 44 px.
- El wizard usa iconos y etiquetas sin líneas conectoras entre pasos.
- Usar una barra compacta, sin porcentaje ficticio ni mensajes de guardado
  derivados únicamente de la navegación. Los estados reales vienen del dominio.

## Verificación

Validar el recorrido completo a 390 px, tablet horizontal y escritorio. Comprobar
teclado, foco visible, ausencia de desbordamiento horizontal y conservación del
contexto al avanzar o volver entre pasos.
