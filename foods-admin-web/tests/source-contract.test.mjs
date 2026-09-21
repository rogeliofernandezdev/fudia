import test from "node:test";
import assert from "node:assert/strict";
import {existsSync,readFileSync,readdirSync,statSync} from "node:fs";
import {join} from "node:path";

const root=process.cwd();
const forbiddenRegionalLocale=["es","PE"].join("-");
const read=p=>readFileSync(join(root,p),"utf8");
const walk=dir=>readdirSync(join(root,dir)).flatMap(name=>{
  const p=join(dir,name);return statSync(join(root,p)).isDirectory()?walk(p):[p];
});

test("la arquitectura no usa un contenedor generico de features",()=>{
  assert.equal(existsSync(join(root,"src/components")),false);
  for(const moduleName of ["auth","configuration","context","customers","dashboard","identity","menu","modules","operations","organizations","platform","public-menu","sales","supply"]){
    assert.equal(existsSync(join(root,`src/modules/${moduleName}/index.ts`)),true,`falta index publico: ${moduleName}`);
  }
});

test("app no contiene CSS de negocio",()=>{
  const css=walk("src/app").filter(p=>p.endsWith(".css"));
  assert.deepEqual(css,[]);
});

test("todo CSS tiene owner explicito",()=>{
  const css=walk("src").filter(p=>p.endsWith(".css"));
  const invalid=css.filter(p=>![
    "src/styles/","src/design-system/styles/","src/shell/styles/","src/providers/styles/","src/modules/"
  ].some(prefix=>p.startsWith(prefix)));
  assert.deepEqual(invalid,[]);
});

test("no quedan imports a components ni CSS antiguo de app",()=>{
  const source=walk("src").filter(p=>/\.(ts|tsx)$/.test(p));
  const bad=[];
  for(const p of source){
    const c=read(p);
    if(c.includes("@/components/")||/from ["'][.]{1,2}\/app\//.test(c)||/import ["'][^"']*\/app\/[^"']*\.css["']/.test(c))bad.push(p);
  }
  assert.deepEqual(bad,[]);
});

test("ninguna vista fija el locale regional de Peru",()=>{
  const source=walk("src").filter(p=>/\.(ts|tsx|js|jsx)$/.test(p));
  const bad=source.filter(p=>read(p).includes(forbiddenRegionalLocale));
  assert.deepEqual(bad,[]);
});

test("presentation no construye transporte HTTP",()=>{
  const presentation=walk("src/modules").filter(p=>p.includes("/presentation/")&&/\.(ts|tsx)$/.test(p));
  const bad=[];
  for(const p of presentation){
    const source=read(p);
    if(/\bapiFetch\b/.test(source)||/\bfetch\s*\(/.test(source))bad.push(p);
  }
  assert.deepEqual(bad,[]);
});

test("providers shell y pages no construyen transporte HTTP",()=>{
  const source=[
    ...walk("src/providers").filter(p=>/\.(ts|tsx)$/.test(p)),
    ...walk("src/shell").filter(p=>/\.(ts|tsx)$/.test(p)),
    ...walk("src/app").filter(p=>p.endsWith("/page.tsx")),
  ];
  const bad=[];
  for(const p of source){
    const content=read(p);
    if(/\bapiFetch\b/.test(content)||/\bfetch\s*\(/.test(content))bad.push(p);
  }
  assert.deepEqual(bad,[]);
});

test("los estilos especializados se cargan desde su owner",()=>{
  const owners={
    "src/shell/admin-shell.tsx":"./styles/shell.css",
    "src/providers/feedback-provider.tsx":"./styles/feedback.css",
    "src/design-system/confirm-dialog.tsx":"./styles/confirm-dialog.css",
    "src/design-system/remote-modal-skeleton.tsx":"./styles/remote-modal-skeleton.css",
    "src/design-system/maps/mapbox-location-provider.tsx":"../styles/location-map.css",
    "src/modules/dashboard/presentation/dashboard-view.tsx":"./dashboard.css",
    "src/modules/configuration/presentation/configuration-home-page.tsx":"./configuration.css",
    "src/modules/modules/presentation/modules-view.tsx":"./modules.css",
    "src/modules/platform/presentation/platform-onboarding-page.tsx":"./platform-onboarding.css",
    "src/modules/platform/presentation/platform-shell.tsx":"./platform-shell.css",
  };
  for(const [p,css] of Object.entries(owners))assert.ok(read(p).includes(css),`${p} no carga ${css}`);
});

test("los modales remotos muestran skeleton mientras esperan datos",()=>{
  const contracts={
    "src/modules/customers/presentation/customers-manager.tsx":["CustomerDetailSkeleton","RemoteModalSkeleton","draftLoading"],
    "src/modules/operations/orders/presentation/orders-manager.tsx":["OrderDetailSkeleton","aria-busy={loading}"],
    "src/modules/operations/salon/presentation/salon-manager.tsx":["OrderDetailSkeleton","aria-busy={loading}"],
    "src/modules/menu/combos/presentation/combos-page.tsx":["ComboDetailSkeleton","RemoteModalSkeleton","loadForEdit.isPending"],
    "src/modules/menu/products/presentation/catalog-manager.tsx":["RemoteModalSkeleton","categories.isLoading"],
    "src/modules/identity/presentation/users-roles-manager.tsx":["RemoteModalSkeleton","roles.isLoading||locations.isLoading","permissions.isLoading"],
    "src/modules/organizations/presentation/organization-admin.tsx":["RemoteModalSkeleton","profiles.isLoading"],
    "src/modules/supply/inventory/presentation/inventory-page.tsx":["RemoteModalSkeleton","products.isLoading"],
    "src/design-system/maps/mapbox-location-provider.tsx":["mapLoading","location-map-canvas-skeleton","location-map-suggestions-loading"],
  };
  for(const [p,needles] of Object.entries(contracts)){
    const source=read(p);
    for(const needle of needles)assert.ok(source.includes(needle),`${p} debe incluir ${needle}`);
  }
});

test("producto e inventario mantienen una sola fuente de verdad",()=>{
  const productTypes=read("src/modules/menu/products/domain/types.ts");
  assert.ok(productTypes.includes('ProductType="prepared"|"retail"'));
  assert.ok(productTypes.includes('productType:ProductType'));
  assert.ok(productTypes.includes('QuantityControl="none"|"portions"|"inventory"'));
  for(const legacy of ["stockMode","defaultDailyQuota","dailyQuota"])assert.equal(productTypes.includes(legacy),false,legacy);

  const productDialog=read("src/modules/menu/products/presentation/product-dialog.tsx");
  for(const label of ["Sin control","Porciones preparadas","Inventario físico"])assert.ok(productDialog.includes(label),label);
  assert.equal(productDialog.includes("Cupo diario predeterminado"),false);

  assert.equal(existsSync(join(root,"src/modules/supply/presentation/inventory-page.tsx")),false);
  const inventory=read("src/modules/supply/inventory/presentation/inventory-page.tsx");
  assert.ok(inventory.includes("Registrar ajuste"),"Inventario expone únicamente la acción de ajuste manual");
  assert.ok(inventory.includes("listInventory"),"Inventario consulta las existencias reales del local");
  assert.ok(inventory.includes("createInventoryAdjustment"),"Inventario registra ajustes mediante su endpoint específico");
  assert.equal(inventory.includes("Nueva entrada"),false,"Inventario ya no presenta el flujo genérico de entradas");
  assert.equal(inventory.includes("Nuevo producto"),false,"Inventario no crea productos vendibles");
  assert.equal(inventory.includes("Nuevo insumo"),false,"Inventario no crea insumos");
  assert.equal(inventory.includes("listInventoryCategories"),false,"Inventario no consulta categorías para altas rápidas");

  assert.equal(existsSync(join(root,"src/modules/supply/inventory/presentation/inventory-entry-dialog.tsx")),false,"Se elimina el formulario anterior de entradas");
  assert.equal(existsSync(join(root,"src/modules/supply/inventory/domain/inventory-entry-schema.ts")),false,"Se elimina el esquema anterior de entradas");

  const dialog=read("src/modules/supply/inventory/presentation/inventory-adjustment-dialog.tsx");
  const adjustmentSchema=read("src/modules/supply/inventory/domain/inventory-adjustment-schema.ts");
  assert.ok(dialog.includes("useForm<InventoryAdjustmentDraft>"),"El ajuste usa React Hook Form");
  assert.ok(dialog.includes("resolver:inventoryAdjustmentResolver"),"El ajuste delega validación al resolver Zod");
  assert.ok(dialog.includes("STOCK ACTUAL"),"El formulario muestra la existencia actual");
  assert.ok(dialog.includes("UNIDAD BASE"),"El formulario muestra la unidad base");
  assert.ok(dialog.includes("Tipo de movimiento"),"El ajuste solicita entrada o salida");
  assert.ok(dialog.includes('value="entry"'),"El ajuste permite entrada");
  assert.ok(dialog.includes('value="exit"'),"El ajuste permite salida");
  for(const reason of ["Ajuste por sobrante","Ajuste por faltante","Merma","Vencimiento","Otro motivo de salida"])assert.ok(dialog.includes(reason),reason);
  assert.ok(dialog.includes("La salida no puede superar el stock actual."),"La interfaz evita saldos negativos antes de enviar");
  assert.ok(dialog.includes("Observación opcional"),"La observación permanece opcional");
  assert.ok(dialog.includes('busy?"Guardando…":"Guardar"'),"El footer conserva el texto estándar de guardado");
  assert.ok(adjustmentSchema.includes('movementType:z.enum(["entry","exit"])'),"El esquema restringe los tipos de ajuste");
  assert.ok(adjustmentSchema.includes("surplus_adjustment"),"El esquema valida el motivo de sobrante");
  assert.ok(adjustmentSchema.includes("shortage_adjustment"),"El esquema valida el motivo de faltante");
  assert.ok(adjustmentSchema.includes("expiration"),"El esquema valida vencimiento");

  assert.equal(inventory.includes("Buscar producto o SKU"),false,"El buscador visible de Inventario no expone SKU");
  assert.equal(inventory.includes("item.categoryName"),false,"Inventario no repite la categoría debajo del artículo");
  assert.equal(inventory.includes("Actualizado"),false,"Inventario no muestra metadatos de fecha en la tabla principal");

  const kardex=read("src/modules/supply/inventory/presentation/kardex-page.tsx");
  const regionalFormat=read("src/shared/i18n/regional-format.ts");
  const sessionApi=read("src/shared/session/session-api.ts");
  assert.ok(kardex.includes("formatRegionalDateTime"),"Kárdex usa el formateador regional compartido");
  assert.ok(kardex.includes("location?.country"),"Kárdex toma el país del local activo");
  assert.ok(kardex.includes("location?.timezone"),"Kárdex toma la zona horaria del local activo");
  assert.equal(kardex.includes(forbiddenRegionalLocale),false,"Kárdex no fija Perú como región");
  for(const column of ["MOTIVO","STOCK ANTERIOR","STOCK RESULTANTE","USUARIO"])assert.ok(kardex.includes(column),column);
  assert.ok(kardex.includes("item.reason"),"Kárdex expone el motivo del ajuste");
  assert.ok(kardex.includes("item.balanceBefore"),"Kárdex expone el saldo anterior");
  assert.ok(kardex.includes("item.balanceAfter"),"Kárdex expone el saldo resultante");
  assert.ok(kardex.includes("item.createdByName"),"Kárdex expone el usuario que registró el movimiento");
  assert.equal(kardex.includes("item.sourceId.slice"),false,"Kárdex no expone fragmentos de UUID");
  assert.equal(kardex.includes("item.productId.slice"),false,"Kárdex no expone UUID de producto");
  assert.ok(kardex.includes("item.itemName"),"Kárdex funciona también para insumos sin ProductId");
  assert.equal(inventory.includes(forbiddenRegionalLocale),false,"Inventario no fija Perú como región");
  assert.ok(regionalFormat.includes("timeZone:context.timeZone||options.timeZone"),"El formateador aplica la zona horaria operativa");
  assert.ok(sessionApi.includes("country:string;timezone:string"),"La sesión expone país y zona horaria del local");

  const inventoryApi=read("src/modules/supply/inventory/infrastructure/inventory-api.ts");
  assert.ok(inventoryApi.includes('"inventory/adjustments"'),"Inventario usa el endpoint de ajustes");
  assert.equal(inventoryApi.includes('"inventory/entries"'),false,"El frontend ya no usa el endpoint de entradas");
  assert.ok(inventoryApi.includes("movementType:draft.movementType"),"El ajuste envía el tipo de movimiento");
  assert.ok(inventoryApi.includes("reason:draft.reason"),"El ajuste envía el motivo");
  assert.ok(inventoryApi.includes("quantity:Number(draft.quantity)"),"El ajuste envía cantidad numérica");
  assert.ok(inventoryApi.includes("observation:draft.observation.trim()"),"El ajuste envía la observación");
  const availability=read("src/modules/menu/availability/presentation/product-availability-manager.tsx");
  assert.ok(availability.includes('quantityControl==="portions"'));
  assert.ok(availability.includes('quantityControl==="inventory"'));
  assert.ok(availability.includes("La existencia se actualiza únicamente desde Inventario"));
});

test("compras concentra orden recepcion y altas de abastecimiento",()=>{
  const purchases=read("src/modules/supply/purchases/presentation/purchases-page.tsx");
  const purchasesCss=read("src/modules/supply/purchases/presentation/purchases.css");
  const itemDialog=read("src/modules/supply/purchases/presentation/purchase-item-dialog.tsx");
  const receiptDialog=read("src/modules/supply/purchases/presentation/purchase-receipt-dialog.tsx");
  const api=read("src/modules/supply/purchases/infrastructure/purchases-api.ts");
  const productApi=read("src/modules/menu/products/infrastructure/products-api.ts");
  const productImageApi=read("src/shared/api/product-image.ts");
  const inventory=read("src/modules/supply/inventory/presentation/inventory-page.tsx");

  assert.ok(purchases.includes("PurchaseItemDialog"),"La alta de artículo vive dentro del flujo existente de Compras");
  assert.ok(purchasesCss.includes(".purchase-order-modal>.modal-busy{inset:0"),"Guardar una OC bloquea todo el modal");
  assert.ok(purchases.includes('inert={busy}'),"Guardar una OC desactiva interacción de teclado y puntero en el formulario");
  assert.ok(purchasesCss.includes(".purchase-order-modal{display:flex;flex-direction:column"),"El scroll de la OC no desplaza el overlay de guardado");
  assert.ok(purchasesCss.includes("overflow-y:auto;overscroll-behavior:contain"),"El scroll queda contenido en el cuerpo de la OC");
  assert.ok(purchases.includes("PurchaseReceiptDialog"),"La recepción vive dentro del mismo módulo Compras");
  assert.ok(purchases.includes('openDetail(order.id,"view")'),"El ojo abre el detalle en modo solo lectura");
  assert.ok(purchases.includes('openDetail(order.id,"review")'),"Revisar orden abre un modo operativo distinto");
  assert.ok(purchases.includes('mode==="review"&&(canManage||canReceive)'),"Las acciones de workflow solo existen en modo revisión");
  assert.ok(purchases.includes('mode==="view"?"DETALLE DE ORDEN":"REVISAR ORDEN"'),"El modal comunica claramente si es consulta o revisión");
  assert.ok(purchases.includes("Ir a Recepciones"),"Una orden aprobada deriva al workspace de Recepciones");
  assert.ok(purchases.includes('tab==="receipts"'),"Compras tiene una vista independiente de Recepciones");
  assert.ok(purchases.includes('queryKey:["purchase-orders","receivable-summary"]'),"El contador de Recepciones se consulta aunque la pestaña no esté activa");
  assert.ok(purchases.includes('status:"receivable",page:1,pageSize:1'),"El contador usa la cola de órdenes recibibles");
  assert.ok(purchases.includes('invalidateQueries({queryKey:["purchase-orders","receipts"]})'),"Entrar a Recepciones refresca su listado");
  assert.ok(purchases.includes("<span>Recepciones</span>"),"La navegación expone Recepciones como workspace propio");
  assert.ok(purchases.includes('"receivable"'),"La cola de Recepciones consulta solo órdenes recibibles");
  assert.ok(purchases.includes("No hay mercadería pendiente de recibir"),"Recepciones tiene estado vacío propio");
  assert.ok(purchases.includes("Solo aparecen órdenes aprobadas con cantidades pendientes."),"Recepciones explica su responsabilidad");
  assert.equal(purchases.includes("PurchaseFlowSteps"),false,"La navegación por pestañas evita repetir un stepper dentro del módulo");
  assert.equal(receiptDialog.includes("PurchaseFlowSteps"),false,"El modal de recepción no repite el flujo completo");
  assert.ok(purchases.includes("partially_received"),"Compras representa una recepción parcial sin cerrar la orden");
  for(const column of ["SOLICITADO","RECIBIDO","PENDIENTE"])assert.ok(purchases.includes(column),column);
  assert.ok(purchases.includes("Agregar artículo"),"La OC expone un único punto claro para agregar artículos");
  assert.ok(purchases.includes("fields.length>0&&<Button"),"El botón superior aparece solo cuando ya existen líneas");
  assert.equal(purchases.includes("Agregar primer artículo"),false,"El estado vacío no duplica el CTA con otra etiqueta");
  assert.ok(purchases.includes("Guardar borrador"),"Guardar la OC comunica que aún no hay recepción ni movimiento de stock");
  assert.ok(purchases.includes("Busca un artículo existente o crea uno nuevo para incluirlo en la orden."),"La OC usa una ayuda breve en el estado vacío");
  assert.equal(purchases.includes("Aún no agregaste artículos"),false,"El estado vacío no repite el título de la sección");
  assert.equal(purchases.includes("Cambiar artículo"),false,"La OC evita una acción redundante que podría conservar datos del artículo anterior");
  assert.ok(purchases.includes("purchase-line-remove"),"Cada línea conserva una única acción explícita para quitar el artículo");

  assert.ok(itemDialog.includes("Artículo existente"),"El selector muestra explícitamente la ruta de artículo existente");
  assert.ok(itemDialog.includes("Escribe el nombre del producto o insumo"),"La búsqueda de artículos vive dentro del selector");
  assert.ok(itemDialog.includes("Nuevo producto vendible"),"Compras conserva el alta de mercadería vendible");
  assert.ok(itemDialog.includes("Nuevo insumo"),"Compras conserva el alta de insumo no vendible");
  assert.ok(itemDialog.includes("Stock inicial: 0"),"Crear el artículo desde Compras no mueve inventario");
  assert.equal(itemDialog.includes("quantityControl"),false,"El formulario no expone detalles técnicos de control de cantidad");
  assert.ok(itemDialog.includes("no se vuelve vendible"),"El insumo no se convierte automáticamente en Producto");
  assert.ok(itemDialog.includes("Imagen del producto (opcional)"),"El alta vendible migra la imagen del producto");
  assert.ok(itemDialog.includes('accept="image/png,image/jpeg,image/webp"'),"La imagen conserva los formatos permitidos");
  assert.ok(itemDialog.includes("máximo 5 MB"),"La imagen conserva el límite de tamaño");

  assert.ok(receiptDialog.includes("Registra únicamente lo que llegó."),"Recepción registra cantidades reales");
  assert.equal(receiptDialog.includes("actualizará Inventario y quedarán registradas"),false,"El modal evita mensajes redundantes al final");
  assert.ok(receiptDialog.includes("pendingQuantity"),"Recepción parte de lo pendiente por línea");
  assert.ok(receiptDialog.includes("Confirmar recepción"),"Solo confirmar recepción dispara la entrada real");

  assert.ok(api.includes('"purchase-inventory-items"'),"Compras usa su propio límite para crear artículos");
  assert.ok(api.includes("purchase-orders/"),"Recepción permanece vinculada a la orden existente");
  assert.ok(api.includes("purchaseOrderItemId"),"La recepción identifica cada línea de la orden");
  assert.ok(api.includes("uploadProductImage"),"Compras reutiliza la carga compartida de imagen");
  assert.ok(productApi.includes("uploadProductImage"),"Productos usa la misma carga compartida de imagen");
  assert.ok(productImageApi.includes("/api/admin/products/"),"La carga de imagen mantiene un solo endpoint compartido");
  assert.equal(api.includes('"inventory/entries"'),false,"Compras no usa la antigua entrada manual de Inventario");

  assert.equal(inventory.includes("Nuevo producto vendible"),false,"Inventario no recupera creación de productos");
  assert.equal(inventory.includes("Nuevo insumo"),false,"Inventario no recupera creación de insumos");
});

test("caja separa cajas fisicas de sus turnos",()=>{
  const page=read("src/modules/operations/cash/presentation/cash-page.tsx");
  const dialogs=read("src/modules/operations/cash/presentation/cash-dialogs.tsx");
  const advancedDialogs=read("src/modules/operations/cash/presentation/cash-advanced-dialogs.tsx");
  const api=read("src/modules/operations/cash/infrastructure/cash-api.ts");
  const schema=read("src/modules/operations/cash/domain/cash-schema.ts");
  const types=read("src/modules/operations/cash/domain/types.ts");
  const route=read("src/app/(admin)/caja/page.tsx");

  assert.ok(route.includes("CashPage"),"/caja compone el módulo operativo real");
  assert.ok(page.includes(">Cajas<"),"La vista principal administra cajas registradas");
  assert.ok(page.includes(">Turnos<"),"Caja expone historial de turnos");
  assert.ok(page.includes("Aún no hay cajas registradas"),"Sin cajas se explica que primero debe registrarse una");
  assert.ok(page.includes("Registrar caja"),"El primer paso funcional es registrar una caja");
  assert.ok(page.includes("Iniciar turno"),"El turno se inicia desde una caja existente");
  assert.ok(page.includes("cash-overview"),"Cajas muestra un resumen operativo");
  assert.ok(page.includes("EFECTIVO ESPERADO"),"El resumen agrega el efectivo esperado de turnos abiertos");
  assert.ok(page.includes("Editar caja"),"Las cajas pueden renombrarse sin perder historial");
  assert.ok(page.includes("toggleStatus"),"Las cajas pueden activarse o desactivarse");
  assert.ok(page.includes("businessDate(item.businessDate)"),"El historial usa día operativo");
  assert.ok(page.includes("shiftTarget"),"El turno conserva como objetivo la caja elegida");
  assert.ok(page.includes("cashRegisterName"),"El historial identifica la caja de cada turno");
  assert.ok(page.includes('movement(shift,"income")'),"Un turno abierto permite ingresos manuales");
  assert.ok(page.includes('movement(shift,"expense")'),"Un turno abierto permite egresos manuales");
  assert.ok(page.includes("CloseCashShiftDialog"),"El cierre se realiza mediante arqueo");
  assert.ok(page.includes('RowActionButton action="view"'),"El historial abre detalle solo lectura");

  assert.ok(dialogs.includes("CashRegisterDialog"),"Registrar caja tiene un formulario propio");
  assert.ok(dialogs.includes("Estás iniciando un turno en"),"El modal de turno deja clara la caja seleccionada");
  assert.ok(dialogs.includes('cashRegister.name.toUpperCase()'),"El turno muestra el nombre de su caja");
  assert.ok(dialogs.includes("EFECTIVO ESPERADO"),"El arqueo muestra el esperado antes de cerrar");
  assert.ok(dialogs.includes("Efectivo contado"),"El arqueo solicita el efectivo contado");
  assert.ok(dialogs.includes("DIFERENCIA"),"El cierre calcula sobrante o faltante");
  assert.ok(dialogs.includes("ORIGEN"),"El detalle de turno muestra el origen de cada movimiento");
  assert.ok(dialogs.includes("Cierre ciego activo"),"El arqueo soporta cierre ciego real");
  assert.ok(dialogs.includes("Por denominaciones"),"El arqueo permite conteo por denominaciones");
  assert.ok(dialogs.includes("denominationsForCurrency"),"Las denominaciones se adaptan a la moneda configurada");
  assert.ok(advancedDialogs.includes("CashTeamDialog"),"Caja administra el equipo asignado al turno");
  assert.ok(advancedDialogs.includes("CashOperationDialog"),"Caja separa retiros depósitos y transferencias de movimientos manuales");
  assert.ok(advancedDialogs.includes("Transferencia entre cajas"),"Las transferencias de efectivo tienen un flujo explícito");

  assert.ok(api.includes('"cash-registers"'),"Frontend consulta y crea cajas del local");
  assert.ok(api.includes("updateCashRegister"),"Frontend permite renombrar cajas");
  assert.ok(api.includes("setCashRegisterActive"),"Frontend administra disponibilidad de cajas");
  assert.ok(api.includes("cashRegisterId"),"Abrir turno envía explícitamente la caja seleccionada");
  assert.ok(api.includes("/movements"),"Caja registra movimientos contra el turno");
  assert.ok(api.includes("/close"),"Caja cierra el turno mediante endpoint dedicado");
  assert.ok(api.includes("listCashShiftUsers"),"Caja consulta el equipo de cada turno");
  assert.ok(api.includes("assignCashShiftUser"),"Caja asigna usuarios al turno");
  assert.ok(api.includes("createCashOperation"),"Caja registra operaciones especiales atómicas");
  assert.ok(schema.includes("cashRegisterResolver"),"El alta de caja valida su formulario");
  assert.ok(schema.includes("cashMovementResolver"),"Movimientos usan validación de formulario");
  assert.ok(schema.includes("closeCashShiftResolver"),"El arqueo usa validación de formulario");
  assert.ok(types.includes("openShift:CashShift|null"),"Cada caja conoce si tiene un turno abierto");
  assert.ok(types.includes("businessDate:string"),"El turno conserva día operativo");
  assert.ok(types.includes('sourceType:"manual"'),"Los movimientos están preparados para orígenes automáticos");
  assert.ok(types.includes("expectedVisible:boolean"),"El backend controla si el esperado puede mostrarse");
  assert.ok(types.includes("blindClose:boolean"),"Cada caja conserva su configuración de cierre ciego");
  assert.ok(types.includes("CashShiftUser"),"El turno soporta múltiples usuarios");
  assert.ok(types.includes("CashCountLine"),"El cierre conserva conteo por denominaciones");
});

test("pos cobra pedidos y sincroniza efectivo con caja",()=>{
  const page=read("src/modules/operations/pos/presentation/pos-page.tsx");
  const dialogs=read("src/modules/operations/pos/presentation/pos-dialogs.tsx");
  const api=read("src/modules/operations/pos/infrastructure/pos-api.ts");
  const route=read("src/app/(admin)/pos/page.tsx");

  assert.ok(route.includes("POSPage"),"/pos ya no es un placeholder");
  assert.equal(route.includes("ComingSoonPage"),false,"Punto de venta tiene implementación operativa");
  assert.ok(page.includes("Por cobrar"),"POS conserva pagos parciales en la cola de cobro");
  assert.ok(page.includes("getCurrentCashShift"),"POS exige un turno asignado para operar");
  assert.ok(page.includes("No estás asignado a una caja abierta"),"POS explica claramente cuando falta turno");
  assert.ok(page.includes("PaymentDialog"),"POS registra cobros desde la bandeja");
  assert.ok(page.includes("RefundDialog"),"POS permite devolver pagos existentes");
  assert.ok(dialogs.includes("incrementará automáticamente el efectivo esperado"),"El cobro en efectivo comunica su impacto en Caja");
  assert.ok(dialogs.includes("La devolución saldrá del efectivo esperado"),"La devolución en efectivo comunica su impacto en Caja");
  assert.ok(api.includes('apiFetch<Payment>("payments"'),"POS registra pagos en el límite oficial");
  assert.ok(api.includes("/refund"),"POS registra devoluciones ligadas al pago original");
});

test("combos conserva la misma tabla en movil y el shell no desborda",()=>{
  const combos=read("src/modules/menu/combos/presentation/combos-page.tsx");
  assert.ok(combos.includes('className="table-wrap hover-scroll"'));
  assert.equal(combos.includes("combo-mobile-cards"),false);
  assert.equal(combos.includes("\\n    {draft&&<ComboWizard"),false);
  const comboCss=read("src/modules/menu/combos/presentation/combo-wizard.css");
  assert.ok(comboCss.includes("container-name: combo-list"));
  assert.ok(comboCss.includes("@container combo-list (width <= 820px)"));
  assert.ok(comboCss.includes(".standardized-management.combo-list .table-wrap"));
  assert.ok(comboCss.includes("overflow-x: auto"));
  assert.ok(comboCss.includes("min-width: 720px"));
  const shell=read("src/shell/styles/shell.css");
  assert.ok(shell.includes("height:100dvh"));
  assert.ok(shell.includes(".sidebar.open{z-index:90"));
  assert.ok(shell.includes(".sidebar-scrim{backdrop-filter:blur(2px);z-index:80"));
  assert.ok(shell.includes(".sidebar{z-index:90"));
  assert.ok(shell.includes("container-name:admin-main"));
  assert.ok(shell.includes("@container admin-main (width<=1040px)"));
  assert.ok(shell.includes(".platform-link{width:44px"));
  const context=read("src/modules/context/presentation/context-switcher.css");
  assert.ok(context.includes("@container admin-main (width<=1040px)"));
  assert.ok(context.includes("text-overflow:ellipsis"));
  const account=read("src/shell/styles/account-menu.css");
  assert.ok(account.includes("@container admin-main (width<=1040px)"));
  assert.ok(account.includes(".account-popover>header small{overflow:hidden"));
  assert.ok(account.includes("@media(max-width:820px)"));
  assert.ok(account.includes(".account-trigger-copy,.account-trigger>svg{display:none!important}"));
  assert.ok(account.includes(".account-menu{width:44px;min-width:44px;max-width:44px"));
  const globals=read("src/styles/globals.css");
  assert.ok(globals.includes(".page-header>.button{flex:0 0 auto;white-space:nowrap}"));
  const nav=read("src/shell/styles/navigation-state.css");
  assert.ok(nav.includes(".admin-shell[data-sidebar=collapsed] .sidebar{width:min(320px,86vw)"));
});

test("el root layout carga solo la base global",()=>{
  const c=read("src/app/layout.tsx");
  const cssImports=[...c.matchAll(/import ["']([^"']+\.css)["']/g)].map(m=>m[1]);
  assert.deepEqual(cssImports,["@/styles/globals.css"]);
});

test("las rutas principales componen modulos",()=>{
  const expected={
    "src/app/(admin)/pedidos/page.tsx":"@/modules/operations",
    "src/app/(admin)/salon/page.tsx":"@/modules/operations",
    "src/app/(admin)/cocina/page.tsx":"@/modules/operations",
    "src/app/(admin)/mesas/page.tsx":"@/modules/operations",
    "src/app/(admin)/pos/page.tsx":"@/modules/operations",
    "src/app/(admin)/caja/page.tsx":"@/modules/operations",
    "src/app/(admin)/productos/page.tsx":"@/modules/menu",
    "src/app/(admin)/combos/page.tsx":"@/modules/menu",
    "src/app/(admin)/clientes/page.tsx":"@/modules/customers",
    "src/app/(admin)/locales/page.tsx":"@/modules/organizations",
    "src/app/(admin)/configuracion/usuarios/page.tsx":"@/modules/identity",
    "src/app/(admin)/inventario/page.tsx":"@/modules/supply",
    "src/app/(admin)/kardex/page.tsx":"@/modules/supply",
    "src/app/(admin)/compras/page.tsx":"@/modules/supply"
  };
  for(const [p,dependency] of Object.entries(expected))assert.ok(read(p).includes(dependency),p);
});

test("no se versionan secretos locales ni artefactos temporales en la raiz admin",()=>{
  assert.equal(existsSync(join(root,".env.local")),false);
  const rootEntries=readdirSync(root);
  assert.deepEqual(rootEntries.filter(name=>/^__(diag|shot|sheet|comanda)/.test(name)),[]);
});

test("se preservan contratos visuales base",()=>{
  const css=read("src/styles/globals.css").replace(/\s+/g,"");
  for(const token of ["--brand-700","--ops-700","--digital-700","--primary-600","--control-height"])assert.ok(css.includes(token),token);
  const shell=read("src/shell/admin-shell.tsx");
  for(const label of ["Reportes","Punto de venta","Carta y productos","Inventario","Compras","CONFIGURACIÓN"])assert.ok(shell.includes(label),label);
});
