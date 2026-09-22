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
  assert.ok(availability.includes("La existencia se actualiza desde Inventario y con las ventas."));
  assert.ok(availability.includes('can("menu.read")'),"Disponibilidad respeta permiso de lectura");
  assert.ok(availability.includes('can("menu.manage")'),"Disponibilidad separa permiso de escritura");
  assert.ok(availability.includes("function saveQuota"),"Guardar cupo tiene un flujo explícito");
  assert.ok(availability.includes("function setManualStatus"),"Cambiar estado no reutiliza el borrador de cupo");
  assert.ok(availability.includes("portionQuantity:null,kind:\"status\""),"El cambio de estado no persiste un cupo pendiente");
  assert.ok(availability.includes("Math.max(1,item.soldQuantity)"),"El cupo nunca baja de lo ya vendido");

  const recipes=read("src/modules/menu/recipes/presentation/recipes-page.tsx");
  const recipesApi=read("src/modules/menu/recipes/infrastructure/recipes-api.ts");
  assert.ok(recipes.includes('AsyncSelect from "react-select/async"'),"Recetas reutiliza el autocomplete asíncrono existente");
  assert.ok(recipes.includes("loadOptions={loadRecipeProductOptions}"),"Producto preparado busca opciones de forma asíncrona");
  assert.ok(recipes.includes("cacheOptions"),"El autocomplete reutiliza resultados recientes");
  assert.equal(recipes.includes('<Select value={draft.productId}'),false,"Producto preparado no vuelve a un select nativo");
  assert.ok(recipesApi.includes('products?status=active&q='),"La búsqueda de productos se delega al backend");
  assert.ok(recipesApi.includes('products?status=active&page=1&pageSize=10'),"La apertura inicial carga solo 10 productos");
  assert.ok(recipesApi.includes("pageSize=100"),"Las búsquedas usan páginas amplias para reunir coincidencias");
  assert.ok(recipesApi.includes("Math.ceil(first.total/100)"),"La búsqueda recorre todas las páginas hasta reunir todas las coincidencias");
  assert.ok(recipesApi.includes("rest.flatMap(page=>page.items)"),"La búsqueda combina todas las páginas del API");
  assert.ok(recipes.includes("search.length>0&&search.length<3"),"El autocomplete no consulta con uno o dos caracteres");
  assert.ok(recipes.includes("Escribe al menos 3 caracteres"),"La UI comunica el umbral mínimo de búsqueda");
  assert.ok(recipes.includes("label:product.name"),"El autocomplete muestra únicamente el nombre del producto");
  assert.equal(recipes.includes("product.sku?product.name"),false,"El autocomplete no muestra SKU");
  assert.ok(recipesApi.includes('inventory/products?page=1&pageSize=10'),"Insumos carga solo 10 opciones al abrir");
  assert.ok(recipesApi.includes('inventory/products?q='),"La búsqueda de insumos se delega al backend");
  assert.ok(recipesApi.includes("Math.ceil(first.total/100)"),"La búsqueda de insumos recorre todas las páginas necesarias");
  assert.ok(recipes.includes("loadRecipeIngredientOptions"),"Cada insumo usa búsqueda remota");
  assert.ok(recipes.includes('placeholder="Buscar insumo..."'),"La fila de insumo usa autocomplete");
  assert.ok(recipes.includes('inventoryItemId:""'),"Agregar insumo crea una fila vacía sin depender del catálogo cargado");
  assert.equal(recipes.includes("draft.items.length>=(inventory.data?.items.length??0)"),false,"No existe un máximo ligado al tamaño del catálogo cargado");
  assert.ok(recipes.includes('className="recipe-skeleton-table"'),"El skeleton de recetas usa la misma geometría de tabla");
  assert.ok(recipes.includes("<th>PRODUCTO</th><th>RENDIMIENTO</th><th>INSUMOS</th><th>ESTADO</th><th>ACCIONES</th>"),"El skeleton conserva las columnas reales");
  assert.equal(recipes.includes("recipe-list-skeleton-head"),false,"El skeleton no usa barras genéricas como cabecera");
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
  assert.ok(purchases.includes('mode==="review"&&(canManage||canApprove||canReceive)'),"Las acciones de workflow solo existen en modo revisión");
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

test("cocina mantiene jerarquia KDS y semantica de color",()=>{
  const kitchen=read("src/modules/operations/kitchen/presentation/kitchen-board.tsx");
  const css=read("src/modules/operations/kitchen/presentation/kitchen.css");
  assert.ok(kitchen.includes("function formatElapsed"),"Cocina formatea tiempos largos en unidades legibles");
  for(const label of ["A tiempo","Por vencer","Con demora","Listo"])assert.ok(kitchen.includes(label),label);
  assert.ok(kitchen.includes('kind="primary"'),"Las transiciones de cocina usan acción primaria azul");
  assert.equal(kitchen.includes('kind={next==="listo"?"success":"primary"}'),false,"Verde queda reservado al estado listo");
  assert.ok(kitchen.includes("kitchen-skeleton-head"),"El skeleton reproduce la tarjeta de comanda");
  assert.ok(kitchen.includes("kitchen-skeleton-items"),"El skeleton conserva líneas de productos");
  assert.ok(kitchen.includes('status:"preparando",label:"EN PREPARACIÓN",shortLabel:"Preparando",description:"Trabajo activo",icon:"cookingPot"'),"En preparación usa una olla humeante como señal visual");
  assert.ok(kitchen.includes('icon={next==="listo"?"check":"cookingPot"}'),"La acción Iniciar usa la misma metáfora de preparación");
  assert.ok(css.includes('.kitchen-lane{min-width:0;border:1px solid var(--line);border-radius:14px;background:var(--cloud-50)'),"Cada carril es un panel operativo neutro");
  assert.ok(css.includes('.kitchen-lane[data-status="confirmado"]>header{background:var(--kds-pending)}'),"Por preparar usa azul sólido del patrón");
  assert.ok(css.includes('.kitchen-lane[data-status="preparando"]>header{background:var(--kds-cooking)}'),"En preparación usa violeta sólido del patrón");
  assert.ok(css.includes('.kitchen-lane[data-status="listo"]>header{background:var(--kds-ready)}'),"Listo usa brand-600 como fondo");
  assert.ok(css.includes('.kitchen-lane[data-status="listo"]>header h2{color:var(--ink-950)}'),"Listo usa ink-950 para recuperar contraste del título");
  assert.ok(css.includes('.kitchen-lane[data-status="listo"]>header small{color:var(--ink-600);opacity:1}'),"Listo usa ink-600 sin opacidad para la descripción");
  assert.ok(css.includes('.kitchen-lane>header h2{overflow:hidden;margin:0;color:var(--surface)'),"Los títulos de carril conservan contraste blanco");
  assert.equal(css.includes("border-left:4px solid transparent"),false,"Los carriles no usan barra lateral de color");
  assert.equal(css.includes(".kitchen-lane::before"),false,"Los carriles no usan franja superior de color");
  assert.ok(kitchen.includes("ticket.tableName?.toUpperCase()"),"Los nombres de mesa se muestran en mayúsculas");
  assert.ok(css.includes("background:var(--cloud-100)"),"El progreso usa tokens del sistema");
  assert.ok(css.includes("--kds-cooking:var(--digital-500)"),"En preparación usa violeta del patrón operativo");
  assert.ok(css.includes("--kds-cooking-dark:var(--digital-700)"),"El texto de preparación usa violeta oscuro");
  assert.ok(css.includes("--kds-cooking-soft:var(--digital-100)"),"El fondo de preparación usa violeta suave");
  assert.ok(css.includes(".kitchen-ticket-time.late{background:var(--warning-50);border-color:var(--warning-600);color:var(--warning-600)}"),"La demora usa warning y no danger");
  const lateRules=css.split("\n").filter(line=>line.includes(".late"));
  assert.equal(lateRules.some(line=>line.includes("danger-600")),false,"Los estados de demora no usan danger; rojo queda disponible para errores reales");
  assert.equal(css.includes("#"),false,"Cocina no introduce colores hexadecimales directos");
  assert.equal(css.includes("rgba("),false,"Cocina no introduce colores rgba directos");
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
    "src/app/(admin)/reservas/page.tsx":"@/modules/operations",
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

test("mvp admin no presenta datos simulados como operacion real",()=>{
  const dashboard=read("src/modules/dashboard/presentation/dashboard-view.tsx");
  const sales=read("src/modules/sales/presentation/sales-page.tsx");
  const receipts=read("src/modules/sales/presentation/receipts-page.tsx");
  const reservations=read("src/modules/operations/reservations/presentation/reservations-page.tsx");
  const configuration=read("src/modules/configuration/presentation/configuration-home-page.tsx");
  assert.ok(dashboard.includes("getDashboard"),"Dashboard consume datos del backend");
  assert.equal(dashboard.includes("12,840.50"),false,"Dashboard no conserva ventas ficticias");
  assert.ok(sales.includes("listSales"),"Ventas consulta pedidos realmente pagados");
  assert.equal(sales.includes("#10482"),false,"Ventas no conserva filas de ejemplo");
  assert.ok(receipts.includes("Fuera del MVP actual"),"Comprobantes no simula facturacion aun no implementada");
  assert.ok(reservations.includes("saveReservation"),"Reservas persiste altas y ediciones");
  assert.ok(configuration.includes('href:"/configuracion/empresa"'),"Configuracion enlaza los datos de empresa");
  assert.equal(configuration.includes("Facturación electrónica"),false,"Facturacion futura no se ofrece en el hub MVP");
});

test("platform admin ve todo el catalogo y modulos no listos no se activan desde UI",()=>{
  const shell=read("src/shell/admin-shell.tsx");
  const view=read("src/modules/modules/presentation/modules-view.tsx");
  const types=read("src/modules/modules/domain/types.ts");

  assert.ok(shell.includes("const isPlatformAdmin=Boolean(user?.platformAdmin)"),"El shell identifica explícitamente al admin de plataforma");
  assert.ok(shell.includes("isPlatformAdmin?true"),"El admin de plataforma no filtra módulos del sidebar");
  assert.ok(shell.includes("const blockReason=isPlatformAdmin?null"),"El admin de plataforma puede entrar a rutas aunque el módulo esté inactivo");
  assert.ok(types.includes("ModuleAvailability = \"ready\" | \"development\" | \"planned\""),"El frontend modela disponibilidad separada de activación");
  for(const label of ["Disponible","En desarrollo","Planificado"])assert.ok(view.includes(label),label);
  assert.ok(view.includes("const activable=m.availability===\"ready\""),"Solo módulos listos son activables");
  assert.ok(view.includes("disabled={!activable||toggle.isPending}"),"El toggle se bloquea para módulos no disponibles");
  assert.ok(view.includes("Solo los módulos disponibles pueden habilitarse para una empresa."),"La UI explica la regla de activación");
});
test("pantallas completas usan loader FUDIA y cargas internas conservan skeleton",()=>{
  const routeLoading=read("src/app/loading.tsx");
  const loader=read("src/design-system/full-screen-loader.tsx");
  const loaderCss=read("src/design-system/styles/full-screen-loader.css");
  const shell=read("src/shell/admin-shell.tsx");
  const platform=read("src/modules/platform/presentation/platform-shell.tsx");

  assert.ok(routeLoading.includes("<FullScreenLoader"),"La transición de ruta usa el loader FUDIA");
  assert.ok(shell.includes('<FullScreenLoader label="Preparando tu espacio"/>'),"El bootstrap de sesión Admin usa el loader de pantalla completa");
  assert.ok(platform.includes('<FullScreenLoader label="Validando acceso de plataforma"/>'),"Plataforma usa el mismo loader de pantalla completa");
  assert.ok(loader.includes("<Logo/>"),"El loader reutiliza el logo oficial del design system");
  assert.ok(loaderCss.includes("full-screen-logo-paint"),"El logo se revela mediante la animación de pintado");
  assert.ok(loaderCss.includes("position:fixed;inset:0"),"El loader cubre la pantalla completa");
  assert.ok(loaderCss.includes("prefers-reduced-motion:reduce"),"El loader respeta reducción de movimiento");
  assert.equal(existsSync(join(root,"src/styles/loading.css")),false,"El skeleton genérico de pantalla completa fue retirado");
  assert.ok(read("src/modules/sales/presentation/sales-page.tsx").includes("<SalesTableSkeleton/>"),"Ventas conserva skeleton interno");
  assert.ok(read("src/modules/dashboard/presentation/dashboard-view.tsx").includes("<DashboardSkeleton/>"),"Reportes conserva skeleton interno");
  assert.ok(read("src/design-system/remote-modal-skeleton.tsx").includes("RemoteModalSkeleton"),"Los modales remotos conservan skeleton interno");
});

test("reportes y ventas usan skeleton con forma final",()=>{
  const dashboard=read("src/modules/dashboard/presentation/dashboard-view.tsx");
  const dashboardCss=read("src/modules/dashboard/presentation/dashboard.css");
  const sales=read("src/modules/sales/presentation/sales-page.tsx");
  const salesCss=read("src/modules/sales/presentation/sales.css");
  assert.ok(dashboard.includes("<DashboardSkeleton/>"),"Reportes usa skeleton dedicado durante la carga");
  assert.ok(dashboard.includes('aria-label="Cargando reportes"'),"El skeleton de Reportes expone estado accesible");
  for(const shape of ["dashboard-skeleton-kpi","dashboard-skeleton-chart","dashboard-skeleton-alert","dashboard-skeleton-product"])assert.ok(dashboard.includes(shape),`Reportes reproduce ${shape}`);
  assert.ok(dashboardCss.includes("@keyframes dashboard-shimmer"),"Reportes anima únicamente la geometría skeleton");
  assert.equal(dashboard.includes("Cargando datos reales…"),false,"Reportes no usa un estado de carga genérico");

  assert.ok(sales.includes("<SalesTableSkeleton/>"),"Ventas usa skeleton dedicado durante la carga");
  assert.ok(sales.includes("<th>PEDIDO</th><th>CLIENTE / MESA</th><th>CANAL</th><th>FECHA</th><th>TOTAL</th><th>ESTADO</th>"),"Ventas conserva la cabecera final durante la carga");
  assert.ok(sales.includes("sales-skeleton-name")&&sales.includes("sales-skeleton-customer")&&sales.includes("sales-skeleton-status"),"Ventas reproduce la estructura de las filas reales");
  assert.ok(salesCss.includes("@keyframes sales-shimmer"),"Ventas define shimmer del skeleton");
  assert.equal(sales.includes("Cargando ventas…"),false,"Ventas no usa un estado de carga genérico");
});

test("reservas respeta el contrato de formularios y jerarquia del modal",()=>{
  const page=read("src/modules/operations/reservations/presentation/reservations-page.tsx");
  const schema=read("src/modules/operations/reservations/domain/reservation-schema.ts");
  const css=read("src/modules/operations/reservations/presentation/reservations.css");
  assert.ok(page.includes("useForm<ReservationDraft>"),"Reservas usa React Hook Form");
  assert.ok(page.includes("resolver:reservationResolver"),"Reservas delega validacion a Zod");
  assert.ok(schema.includes("reservationSchema=z.object"),"Reservas define esquema Zod");
  assert.ok(page.includes('className="form-grid reservation-form-grid"'),"El modal usa una reticula propia sobre el formulario estandar");
  assert.ok(css.includes("grid-template-columns:minmax(0,1.35fr) minmax(170px,.65fr)"),"La columna principal conserva mayor jerarquia");
  const modalStart=page.indexOf("function ReservationDialog");
  const modal=page.slice(modalStart);
  const dateIndex=modal.indexOf("Fecha y hora");
  const guestsIndex=modal.indexOf("Personas");
  const tableIndex=modal.indexOf("\n            Mesa\n");
  const durationIndex=modal.indexOf("Duración de mesa");
  assert.ok(dateIndex>=0&&guestsIndex>dateIndex&&tableIndex>guestsIndex&&durationIndex>tableIndex,"El flujo visual sigue fecha, personas, mesa y duracion");
  assert.ok(page.includes('{busy?"Guardando…":"Guardar"}'),"El CTA usa el texto estandar Guardar");
  assert.equal(page.includes("Guardar reserva"),false,"El modal no agrega sufijos al CTA Guardar");
  assert.ok(page.includes("tablesLoading")&&page.includes("tablesError"),"La dependencia remota de mesas expone carga y error");
  assert.ok(css.includes("@media (width<=600px)"),"El modal define reorganizacion movil");
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


test("identidad alinea permisos administracion plataforma y perfil",()=>{
  const identity=read("src/modules/identity/presentation/users-roles-manager.tsx");
  const shell=read("src/shell/admin-shell.tsx");
  const platformLayout=read("src/app/platform/layout.tsx");
  const platformShell=read("src/modules/platform/presentation/platform-shell.tsx");
  const profile=read("src/modules/identity/presentation/profile-page.tsx");
  const identityApi=read("src/modules/identity/infrastructure/identity-api.ts");

  assert.ok(identity.includes('const canManage=can("users.manage")'),"Usuarios distingue lectura de administración");
  assert.ok(identity.includes("canManage&&!u.platformAdmin"),"La UI protege cuentas de plataforma y acciones de escritura");
  assert.ok(identity.includes("canManage&&role.systemKey!==\"administrator\""),"El Administrador no expone acciones de edición o desactivación");
  assert.ok(identity.includes('invalidateQueries({queryKey:["session-context"]})'),"Cambios de identidad refrescan permisos de sesión");

  assert.ok(platformLayout.includes("SessionProvider"),"El área de plataforma carga contexto de sesión");
  assert.ok(platformShell.includes("!user?.platformAdmin"),"El onboarding de empresas tiene guard funcional de platform admin");

  assert.ok(shell.includes('href="/configuracion/perfil"'),"La cuenta enlaza a un perfil funcional");
  assert.equal(shell.includes("?section=preferences"),false,"No se anuncian preferencias inexistentes");
  assert.equal(shell.includes("?section=security"),false,"No se anuncia una pantalla de seguridad inexistente");
  assert.ok(profile.includes("saveMyProfile"),"Perfil guarda datos y contraseña por su endpoint real");
  assert.ok(identityApi.includes('apiFetch<MyProfile>("me"'),"Perfil consulta el endpoint autenticado propio");
});


test("perfil respeta inputs y formularios del design system",()=>{
  const profile=read("src/modules/identity/presentation/profile-page.tsx");
  const css=read("src/modules/identity/presentation/profile-page.css");
  const schema=read("src/modules/identity/domain/profile-schema.ts");

  assert.ok(profile.includes("useForm<MyProfileDraft>"),"Perfil usa React Hook Form");
  assert.ok(profile.includes("resolver:profileResolver"),"Perfil delega validación a Zod");
  assert.ok(profile.includes("<Input"),"Perfil reutiliza Input del design system");
  assert.ok(profile.includes('icon="check"'),"La acción de guardar usa el icono homologado");
  assert.ok(profile.includes('save.isPending?"Guardando…":"Guardar"'),"Guardar conserva el texto estándar");
  assert.ok(schema.includes('z.object({'),"Perfil define esquema Zod");
  assert.ok(schema.includes('newPassword.length<8'),"Perfil valida la contraseña nueva");
  assert.equal(css.includes(".profile-field input{"),false,"Perfil no redefine localmente la geometría de Input");
  assert.equal(css.includes(".profile-field .ds-input{"),false,"Perfil no sobrescribe la primitiva Input");
  assert.ok(css.includes(".profile-field .ds-input[readonly]"),"El único estado local del input es el modo solo lectura");
  assert.ok(css.includes("height:var(--control-height)"),"El skeleton de controles sigue el token de altura");
});


test("mesas conserva edición inline dentro de la tabla",()=>{
  const page=read("src/modules/operations/tables/presentation/tables-manager.tsx");
  const api=read("src/modules/operations/tables/infrastructure/tables-api.ts");
  assert.ok(page.includes('RowActionButton action="edit"'),"Mesas expone una acción explícita de edición");
  assert.ok(page.includes("tableDraft?.id===t.id"),"La fila detecta cuál mesa se está editando");
  assert.ok(page.includes("editing-row table-editing-row"),"La edición ocurre dentro de la misma fila");
  assert.ok(page.includes('IconButton icon="check" label="Guardar"'),"La fila usa una acción estándar para guardar");
  assert.ok(page.includes('IconButton icon="close" label="Cancelar"'),"La fila usa una acción estándar para cancelar");
  assert.equal(page.includes("TableDialog"),false,"Editar una mesa no abre un modal");
  assert.equal(page.includes("Mesa activa"),false,"El estado no se duplica dentro de la edición");
  assert.equal(page.includes("QR habilitado"),false,"La edición básica no mezcla configuración de QR");
  assert.ok(page.includes('RowActionButton action="activate"'),"Una mesa inactiva se reactiva desde la acción de fila");
  assert.ok(page.includes('RowActionButton action="deactivate"'),"Una mesa activa se desactiva desde la acción de fila");
  assert.ok(page.includes("persistTable"),"La vista conecta la edición con su API");
  assert.ok(api.includes('method:"PATCH"'),"La edición persiste con PATCH");
  assert.ok(api.includes('tables/${draft.id}'),"La edición usa el endpoint específico de la mesa");
  assert.equal(page.includes('style={{flexWrap:"wrap"}}'),false,"Mesas no introduce estilos inline para composición");
});

test("empresa y kardex respetan no duplicación y patrón de gestión",()=>{
  const company=read("src/modules/organizations/presentation/organization-admin.tsx");
  const companyCss=read("src/modules/organizations/presentation/organization-admin.css");
  const kardex=read("src/modules/supply/inventory/presentation/kardex-page.tsx");
  const inventoryCss=read("src/modules/supply/inventory/presentation/inventory.css");
  assert.equal(company.includes("organization-summary"),false,"Empresa no repite los datos del formulario en una tarjeta resumen");
  assert.ok(company.includes("organization-section"),"Empresa separa identidad legal y configuración general sin duplicar datos");
  assert.ok(company.includes("organization-section-header"),"Empresa usa cabeceras compactas de sección");
  assert.ok(companyCss.includes(".organization-savebar"),"Empresa conserva una barra de guardado consistente");
  assert.ok(kardex.includes('className="panel standardized-management inventory-panel"'),"Kárdex usa el panel estándar de gestión");
  assert.ok(kardex.includes("inventory-toolbar kardex-toolbar"),"Kárdex reutiliza la barra de Inventario");
  assert.ok(kardex.includes("management-cards kardex-cards"),"Kárdex ofrece tarjetas equivalentes en móvil");
  assert.ok(kardex.includes("<th>ARTÍCULO</th><th>FECHA</th>"),"Kárdex presenta primero la entidad gestionada");
  assert.ok(kardex.includes('className="inventory-product-cell"'),"La primera columna reutiliza el patrón de entidad de Inventario");
  assert.equal(kardex.includes("kardex-results-header"),false,"Kárdex no agrega una cabecera redundante de resultados");
  assert.equal(kardex.includes("selectedItem"),false,"Kárdex no repite el artículo seleccionado fuera del filtro");
  assert.equal(kardex.includes('movements.data?.total??0} movimientos'),false,"Kárdex no duplica el total ya mostrado por paginación");
  assert.equal(kardex.includes('style={{flexWrap:"wrap"}}'),false,"Kárdex no depende de estilos inline para el layout");
  assert.ok(inventoryCss.includes(".kardex-toolbar"),"Kárdex mantiene solo estilos complementarios al patrón compartido");
});

test("carta y producción usa iconos semánticos por función",()=>{
  const shell=read("src/shell/admin-shell.tsx");
  const icons=read("src/design-system/icons.tsx");
  const recipes=read("src/modules/menu/recipes/presentation/recipes-page.tsx");
  const availability=read("src/modules/menu/availability/presentation/product-availability-manager.tsx");
  assert.ok(shell.includes('name:"Recetas y producción",icon:"cookingPot"'),"Recetas y producción usa cocción/producción, no un rol de chef");
  assert.ok(shell.includes('name:"Disponibilidad de la carta",icon:"availability"'),"Disponibilidad usa un icono específico de plato disponible");
  assert.ok(icons.includes("availability:"),"El design system define el icono semántico de disponibilidad");
  assert.ok(recipes.includes('name="cookingPot"'),"La página de Recetas conserva la misma semántica visual");
  assert.ok(availability.includes('name="availability"'),"El estado vacío de disponibilidad usa su icono de dominio");
});



test("abastecimiento diferencia inventario de kardex por icono",()=>{
  const shell=read("src/shell/admin-shell.tsx");
  const icons=read("src/design-system/icons.tsx");
  const kardex=read("src/modules/supply/inventory/presentation/kardex-page.tsx");
  assert.ok(shell.includes('name:"Inventario",icon:"stock"'),"Inventario conserva el icono de stock físico");
  assert.ok(shell.includes('name:"Kardex",icon:"ledger"'),"Kárdex usa un icono propio de historial/ledger");
  assert.ok(icons.includes("ledger:"),"El design system define el icono de Kárdex");
  assert.ok(kardex.includes('name="ledger"'),"Kárdex reutiliza su semántica visual dentro de la pantalla");
});


test("los nombres del menú evitan redundancias",()=>{
  const shell=read("src/shell/admin-shell.tsx");
  const recipes=read("src/modules/menu/recipes/presentation/recipes-page.tsx");
  const access=read("src/modules/identity/presentation/users-roles-manager.tsx");
  assert.ok(shell.includes('name:"Recetas",icon:"cookingPot"'),"El menú usa Recetas dentro de Carta y producción");
  assert.equal(shell.includes('name:"Recetas y producción"'),false,"El menú no repite el nombre del grupo");
  assert.ok(shell.includes('name:"Usuarios y roles",icon:"users"'),"El menú nombra las dos entidades administradas");
  assert.ok(recipes.includes('title="Recetas"'),"La página de Recetas usa el mismo nombre");
  assert.ok(access.includes('title="Usuarios y roles"'),"La página de accesos usa Usuarios y roles");
});
