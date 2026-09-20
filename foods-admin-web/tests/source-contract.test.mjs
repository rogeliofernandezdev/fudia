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
  assert.ok(inventory.includes("Nueva entrada"));
  assert.ok(inventory.includes("listInventory"));
  assert.ok(inventory.includes("createInventoryEntry"));
  const dialog=read("src/modules/supply/inventory/presentation/inventory-entry-dialog.tsx");
  assert.ok(dialog.includes("Artículo existente"));
  assert.ok(dialog.includes("Nuevo producto"));
  assert.ok(dialog.includes("mercadería vendible con Inventario físico, no como plato preparado"),"Inventario explica que el alta rápida crea mercadería y no platos");
  assert.ok(dialog.includes("Categoría comercial"),"La categoría se presenta como clasificación comercial, no como tipo de producto");
  assert.ok(dialog.includes("Nuevo insumo"),"Inventario permite crear insumos sin producto vendible");
  assert.ok(dialog.includes("no tendrá precio de venta"),"El flujo de insumos separa inventario de precio comercial");
  assert.ok(dialog.includes('register("categoryId")'),"El alta de producto vendible solicita categoría");
  assert.ok(dialog.includes("Selecciona una categoría"),"El selector de categoría guía una selección explícita");
  assert.ok(dialog.includes("Revisa los campos marcados antes de guardar."),"Guardar inválido informa el problema al usuario");
  const inventoryEntrySchema=read("src/modules/supply/inventory/domain/inventory-entry-schema.ts");
  assert.ok(dialog.includes("useForm<InventoryEntryDraft>"),"Nueva entrada usa React Hook Form");
  assert.ok(dialog.includes("resolver:inventoryEntryResolver"),"Nueva entrada delega la validación al resolver");
  assert.ok(inventoryEntrySchema.includes('z.discriminatedUnion("mode"'),"Zod discrimina reglas según el tipo de registro");
  assert.ok(inventoryEntrySchema.includes('mode:z.literal("existing")'),"Existe esquema para artículo existente");
  assert.ok(inventoryEntrySchema.includes('mode:z.literal("new_product")'),"Existe esquema para producto vendible");
  assert.ok(inventoryEntrySchema.includes('categoryId:z.string().trim().min(1,"Selecciona una categoría.")'),"La categoría es obligatoria al crear un producto vendible");
  assert.ok(inventoryEntrySchema.includes('mode:z.literal("new_ingredient")'),"Existe esquema para insumo");
  assert.equal(dialog.includes("const valid=value.mode"),false,"La vista no mantiene una segunda validación manual de submit");
  assert.ok(dialog.includes("Una sola operación"));
  assert.ok(dialog.includes('type="submit"'),"Guardar inventario debe enviar el formulario");
  assert.ok(dialog.includes('busy?"Guardando…":"Guardar"'),"La acción de guardado usa el texto estándar");
  assert.equal(dialog.includes('Registrar entrada'),false,"El footer no usa etiquetas de guardado específicas");
  assert.equal(dialog.includes('<Button icon="plus" disabled={busy}>'),false,"Guardar no duplica el icono estándar del modal");
  assert.equal(dialog.includes("SKU opcional"),false,"Inventario no expone el SKU interno al usuario");
  assert.equal(dialog.includes("product.name} · {product.sku"),false,"El selector de Inventario no muestra códigos internos");
  assert.ok(dialog.includes("Presentación de ingreso"),"La entrada permite elegir presentación física");
  assert.ok(dialog.includes("+ Nuevo paquete"),"La entrada permite registrar paquetes reutilizables");
  assert.ok(dialog.includes("+ Nueva caja"),"La entrada permite registrar cajas reutilizables");
  assert.ok(dialog.includes("unitsPerPresentation"),"La presentación define su factor hacia la unidad base");
  assert.ok(dialog.includes("Se sumarán"),"La UI anticipa la conversión que afectará el stock");
  assert.equal(inventory.includes("Buscar producto o SKU"),false,"El buscador visible de Inventario no expone SKU");
  assert.equal(inventory.includes("item.categoryName"),false,"Inventario no repite la categoría debajo del producto");
  assert.equal(inventory.includes("Actualizado"),false,"Inventario no muestra metadatos de fecha en la tabla principal");
  assert.equal(inventory.includes("formatInventoryDate"),false,"La tabla principal no necesita formatear timestamps");

  const kardex=read("src/modules/supply/inventory/presentation/kardex-page.tsx");
  const regionalFormat=read("src/shared/i18n/regional-format.ts");
  const sessionApi=read("src/shared/session/session-api.ts");
  assert.ok(kardex.includes("formatRegionalDateTime"),"Kárdex usa el formateador regional compartido");
  assert.ok(kardex.includes("location?.country"),"Kárdex toma el país del local activo");
  assert.ok(kardex.includes("location?.timezone"),"Kárdex toma la zona horaria del local activo");
  assert.equal(kardex.includes(forbiddenRegionalLocale),false,"Kárdex no fija Perú como región");
  assert.ok(kardex.includes("REFERENCIA"),"Kárdex muestra referencia documental en lugar de origen técnico");
  assert.ok(kardex.includes("item.sourceReference"),"Kárdex usa la referencia legible del documento origen");
  assert.ok(kardex.includes("Pedido histórico"),"Kárdex identifica pedidos históricos sin referencia legible");
  assert.ok(kardex.includes("Entrada histórica"),"Kárdex identifica entradas históricas sin referencia legible");
  assert.equal(kardex.includes("item.sourceId.slice"),false,"Kárdex no expone fragmentos de UUID");
  assert.equal(kardex.includes("item.productId.slice"),false,"Kárdex no expone el UUID del producto");
  assert.ok(kardex.includes("item.itemName"),"Kárdex funciona también para insumos sin ProductId");
  assert.equal(kardex.includes("product.name} · {product.sku"),false,"El filtro de Kárdex no expone SKU internos");
  assert.equal(inventory.includes(forbiddenRegionalLocale),false,"Inventario no fija Perú como región");
  assert.ok(regionalFormat.includes("timeZone:context.timeZone||options.timeZone"),"El formateador aplica la zona horaria operativa y conserva un fallback explícito");
  assert.ok(regionalFormat.includes("country?.trim().toUpperCase()"),"El locale regional se deriva del país en contexto");
  assert.ok(sessionApi.includes("country:string;timezone:string"),"La sesión expone país y zona horaria del local");

  const inventoryApi=read("src/modules/supply/inventory/infrastructure/inventory-api.ts");
  assert.ok(inventoryApi.includes('"inventory/entries"'));
  assert.ok(inventoryApi.includes("newProduct"));
  assert.ok(inventoryApi.includes("categoryId:draft.categoryId.trim()"),"La entrada envía la categoría del nuevo producto");
  assert.ok(inventoryApi.includes("listInventoryCategories"),"Inventario obtiene las categorías activas desde el API");
  assert.ok(inventoryApi.includes("newIngredient"),"La API diferencia insumos de productos vendibles");
  assert.ok(inventoryApi.includes("inventoryItemId"),"Las reposiciones operan sobre el artículo de inventario");
  assert.ok(inventoryApi.includes("presentationType"),"La API envía el tipo de presentación");
  assert.ok(inventoryApi.includes("unitsPerPresentation"),"La API envía el factor de conversión");
  assert.ok(inventoryApi.includes("stockQuantity"),"La respuesta distingue cantidad recibida de cantidad de stock");
  const availability=read("src/modules/menu/availability/presentation/product-availability-manager.tsx");
  assert.ok(availability.includes('quantityControl==="portions"'));
  assert.ok(availability.includes('quantityControl==="inventory"'));
  assert.ok(availability.includes("La existencia se actualiza únicamente desde Inventario"));
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
    "src/app/(admin)/mesas/page.tsx":"@/modules/operations",
    "src/app/(admin)/productos/page.tsx":"@/modules/menu",
    "src/app/(admin)/combos/page.tsx":"@/modules/menu",
    "src/app/(admin)/clientes/page.tsx":"@/modules/customers",
    "src/app/(admin)/locales/page.tsx":"@/modules/organizations",
    "src/app/(admin)/configuracion/usuarios/page.tsx":"@/modules/identity",
    "src/app/(admin)/inventario/page.tsx":"@/modules/supply",
    "src/app/(admin)/kardex/page.tsx":"@/modules/supply"
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
