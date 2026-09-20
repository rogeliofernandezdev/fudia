# Navegación

Inicio, Mesas, Punto de venta, Comandas, Pedidos y Caja. En móvil se usa
navegación inferior con cinco destinos de igual jerarquía, icono y nombre; Caja
se accede desde el menú hamburguesa. En escritorio, sidebar azul profundo con
selección azul operativa y scroll propio. El backend autoriza.

Cerrar sesión no forma parte de la navegación. La acción vive exclusivamente en
el desplegable del usuario de la barra superior, junto a su nombre y rol; elimina
la cookie de sesión, evita activaciones repetidas y redirige al acceso.

La gestión incluye Disponibilidad del menú. Esta vista opera sobre el local y
fecha activos y permite marcar un producto disponible, con pocas unidades o
agotado, además de ajustar el cupo diario cuando corresponda.

## Wizard de venta

El flujo operativo es visible y navegable en cuatro etapas: Pedido,
Personalizar, Pago y Comprobante. Caja enlaza a una nueva venta y las cuentas
pendientes abren Pago. La ubicación actual debe permanecer visible sin inferir
operaciones completadas a partir de la ruta. El wizard usa iconos y etiquetas sin
líneas conectoras para evitar superposiciones.

Las reglas completas del recorrido están en docs/10_SALES_WIZARD.md.

## Comandas / KDS

Comandas presenta resumen operativo, selector de estación y colas ordenadas por
urgencia. En escritorio muestra tres columnas; en móvil utiliza pestañas para
mostrar una cola por vez. Cada lista conserva scroll vertical independiente.
Las reglas completas están en `docs/11_KITCHEN_KDS.md`.
