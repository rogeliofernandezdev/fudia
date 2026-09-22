# Comandas y KDS

## Propósito

Permitir que cocina identifique qué preparar, qué está atrasado y qué puede
entregarse sin navegar por pantallas secundarias.

## Jerarquía

1. Encabezado con estado de conexión en vivo.
2. Resumen: por preparar, en preparación, listos y requieren atención.
3. Selector de estación: todas, caliente, fría o barra.
4. Colas Por preparar → En preparación → Listos para entregar.

Las superficies son neutras. Azul, violeta, verde y naranja comunican estado;
no decoran tarjetas completas. Cada carril conserva borde y cuerpo neutros. La cabecera completa usa un fondo semántico claramente distinguible: azul `primary-600`, violeta `digital-500` o verde `brand-600`. Azul y violeta usan título/descripción en blanco; el verde usa `brand-700` para texto. Icono y contador se apoyan en superficie blanca para mantener contraste. No se usan franjas superiores ni barras laterales de color.
Los nombres de mesa se muestran en mayúsculas para facilitar el escaneo rápido.
Las sombras internas se evitan y los radios se mantienen entre 6 y 8 px.

## Orden y tiempo

- Cada cola ordena las comandas de mayor a menor urgencia.
- Tiempo, mesa o canal y correlativo permanecen visibles en el encabezado.
- Las observaciones de cocina se distinguen del nombre del producto.
- Cada comanda presenta una sola acción primaria según su estado.

## Menús y combos en cocina

Una línea de pedido puede representar un producto simple o un menú/combo configurado.
El contrato de órdenes expone `itemType` y, cuando `itemType="combo"`, una colección
`selections` con `groupName`, `name` y `surcharge`. El KDS debe renderizar el
nombre del menú como línea principal y cada selección debajo, agrupada por la parte
del menú que originó la elección.

Ejemplo visual:

```text
1× Menú Ejecutivo
   Entrada: Papa a la huancaína
   Fondo: Lomo saltado
   Bebida: Chicha
   Nota: sin cebolla
```

Las selecciones son una instantánea de la venta. Cocina no vuelve a consultar la
definición actual del combo para reconstruir el ticket, porque el menú puede cambiar
después de registrada la comanda. Los tickets deben usar exclusivamente la composición
persistida en la orden.

## Scroll

- Cada columna tiene scroll vertical independiente y altura ligada al viewport.
- El encabezado de la columna queda fuera de la región desplazable.
- Las tarjetas no se comprimen para intentar mostrar toda la cola.
- El scroll usa `overscroll-behavior` para no arrastrar accidentalmente la página.
- El resumen superior no usa scroll horizontal: se adapta de cuatro a dos columnas.

## Responsive

- Escritorio amplio: tres columnas simultáneas.
- Tablet y ventanas estrechas: una columna por bloque con lista desplazable.
- Hasta 900 px: se oculta el resumen redundante, aparecen pestañas compactas de
  estado y solo la cola seleccionada permanece visible. El encabezado de columna
  también se oculta porque la pestaña ya comunica estado y cantidad.
- Hasta 620 px: selector de estación a ancho completo y controles táctiles de 44 px.

## Estados y accesibilidad

- El estado activo de las pestañas usa `aria-pressed`.
- Los contadores incluyen texto además del color.
- Los tiempos críticos mantienen contraste AA.
- Estados vacíos muestran “Sin tickets”.
- Reducir movimiento respeta `prefers-reduced-motion`.

## Verificación

Probar colas vacías y extensas, cambio de estado, orden por urgencia, selector de
estación, scroll independiente y navegación mediante teclado a 390 px, tablet
horizontal y escritorio.
