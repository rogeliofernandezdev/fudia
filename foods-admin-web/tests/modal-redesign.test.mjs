import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import test from "node:test";
import postcss from "postcss";
import {renderDetail,exampleOrder,detailComponents} from "./helpers/order-detail-fixture.mjs";
const read=file=>readFileSync(new URL(`../src/${file}`,import.meta.url),"utf8");

test("los paneles del pedido quedan remetidos bajo las pestañas sin cambiar su diseño",()=>{
 const css=postcss.parse(read("modules/operations/orders/presentation/order-detail.css"));
 const panel=css.nodes.find(node=>node.selector===".order-detail-panel");
 assert.ok(panel.nodes.some(node=>node.prop==="padding-inline"&&node.value==="var(--space-4)"),"16 px de sangría en escritorio");
 assert.equal(panel.nodes.some(node=>["background","border","border-radius"].includes(node.prop)),false,"sin otra tarjeta ni cambios de radio");
 const mobile=css.nodes.find(node=>node.type==="atrule"&&node.params==="(width<=600px)");
 assert.ok(mobile.nodes.some(node=>node.selector===".order-detail-panel"&&node.nodes.some(decl=>decl.prop==="padding-inline"&&decl.value==="var(--space-3)")),"12 px de sangría en móvil");
 assert.equal((renderDetail().match(/class="order-detail-panel"/g)??[]).length,2,"Productos y Seguimiento comparten la sangría; no hay pestaña Detalles");
 assert.equal(css.nodes.some(node=>node.selector===".order-detail-tabs"),false,"los tabs heredan radio y quedan a 12 px del contenido, sin margen extra");
 assert.doesNotMatch(read("modules/operations/orders/presentation/order-detail-dialog.tsx"),/<header className="order-detail-section-head"><h3 id="order-detail-products-title"/,"el título accesible no añade otra fila al panel");
});

test("Modal y detalle remoto comparten cabecera y workspace, sin implementación alternativa",()=>{
 const modal=read("design-system/modal.tsx"),detail=read("modules/operations/orders/presentation/order-detail-dialog.tsx");
 assert.match(modal,/<ModalHeader title=\{title\}/);
 assert.equal((detail.match(/<ModalHeader /g)??[]).length,2);
 assert.equal((detail.match(/<ModalWorkspace /g)??[]).length,2);
 assert.doesNotMatch(detail,/<header>\s*<span className="modal-title-icon"/);
 assert.doesNotMatch(read("modules/operations/orders/presentation/order-detail.css"),/\.order-detail-(workspace|heading|grand|subtotal)\{/);
});
test("mesa identificada, estados independientes y resumen fuera del cuerpo desplazable",()=>{
 const html=renderDetail();
 assert.match(html,/>Mesa 04<\/h2>/);
 assert.match(html,/Confirmado/);assert.match(html,/Cuenta abierta/);
 assert.doesNotMatch(html,/DETALLE DEL PEDIDO/);
 assert.match(html,/modal-workspace/);assert.match(html,/modal-summary/);
 assert.ok(html.indexOf('class="modal-summary')>html.indexOf("Notas generales"));
 assert.match(html,/Saldo pendiente/);assert.match(html,/S\/ 30.00/);
 assert.match(html,/Cielo Fernandez/);assert.match(html,/Cocina/);assert.match(html,/Barra/);
});
test("cabecera breve, responsables en el cuerpo y estados junto a su información",()=>{
 const html=renderDetail(),modal=read("design-system/modal.tsx");
 assert.doesNotMatch(modal,/modal-heading-main/);
 assert.match(modal,/details=\{details\}/);
 const headerEnd=html.indexOf("</header>");
 assert.doesNotMatch(html.slice(0,headerEnd),/Confirmado|Cuenta abierta|Cielo Fernandez|<time/);
 assert.ok(html.indexOf('class="order-attribution"')>headerEnd);
 assert.ok(html.indexOf('class="order-detail-hero"')>headerEnd,"estado y hora encabezan el cuerpo");
 assert.ok(html.indexOf('aria-label="Estado del pedido"')<html.indexOf('role="tablist"'),"el estado va antes de las pestañas");
 assert.match(html,/role="tablist" aria-label="Información del pedido"/);
 assert.ok(html.indexOf('class="order-attribution"')<html.indexOf('id="order-detail-products-title"'));
 assert.match(html,/status sky emphasized[^>]*><svg[^>]*class="[^"]*status-icon"/);
 assert.match(html,/<\/svg>Cuenta abierta/);
 assert.match(renderDetail({skeleton:true}),/order-detail-information/);
 const delivery=renderDetail({order:{...exampleOrder,channel:"delivery",customerPhone:"999 123 456",address:"Av. Los Olivos 124"}});
 assert.match(delivery,/aria-label="Entrega"/);
 assert.match(delivery,/class="order-detail-contact"/);
 assert.match(delivery,/href="tel:999123456"/);
});
test("cada estado del pedido tiene un tono propio y las preparaciones lo reutilizan",()=>{
 const {orderStatus,serviceStatusTone,accountTone}=detailComponents();
 const states={nuevo:"gray",confirmado:"blue",preparando:"violet",listo:"green",en_camino:"orange",entregado:"teal",cancelado:"red"};
 for(const [state,tone] of Object.entries(states))assert.equal(orderStatus(state).tone,tone);
 assert.equal(new Set(Object.values(states)).size,Object.keys(states).length);
 for(const state of ["preparando","listo","entregado","cancelado"])assert.equal(serviceStatusTone(state),states[state]);
 for(const state of ["pendiente","nuevo","confirmado"])assert.equal(serviceStatusTone(state),"orange");
 assert.equal(serviceStatusTone("desconocido"),"gray");
 assert.equal(accountTone({...exampleOrder,completedAt:"2026-10-10"}),"gray");
});
test("productos, notas y cuenta tienen límites propios y cantidades circulares con la paleta existente",()=>{
 const html=renderDetail(),css=read("design-system/styles/modal.css");
 assert.match(html,/class="modal-item-list order-detail-items" role="list"/);
 assert.doesNotMatch(html,/modal-accent/);
 assert.doesNotMatch(html,/modal-item-columns/);
 assert.match(html,/role="listitem"/);
 const consumptionEnd=html.indexOf('</section>',html.indexOf('class="order-detail-consumption"'));
 assert.ok(html.indexOf('class="modal-detail-notes')>consumptionEnd);
 assert.match(html,/class="status sky emphasized"/);
 assert.doesNotMatch(html,/class="status gray outline"/);
 assert.match(html,/class="status orange"[^>]*><i[^>]*><\/i>Pendiente/);
 const quantityRule=css.match(/\.modal-item-quantity\{([^}]*)\}/)?.[1]??"";
 assert.match(quantityRule,/border-radius:var\(--radius-pill\)/);
 assert.match(quantityRule,/background:var\(--primary-100\)/);
 assert.match(quantityRule,/color:var\(--primary-700\)/);
 assert.doesNotMatch(quantityRule,/cursor|box-shadow|#[\da-f]+/i);
 assert.match(renderDetail({skeleton:true}),/--radius-pill/);
 assert.match(css,/\.modal-summary\{[^}]*border-top:var\(--stroke-1\) solid var\(--line\)[^}]*background:var\(--cloud-50\)/);
 assert.doesNotMatch(css.match(/\.modal-item-list\{([^}]*)\}/)?.[1]??"",/border|background/);
 assert.match(css,/\.modal-detail-notes\{[^}]*background:var\(--cloud-50\)/);
});
test("el rediseño conserva autorización del mozo y el bloqueo de cierre sin entrega",()=>{
 const own=renderDetail();assert.match(own,/Agregar productos/);assert.doesNotMatch(own,/Cerrar cuenta/);assert.doesNotMatch(own,/Cobrar saldo/);
 const other=renderDetail({userId:"another-waiter"});assert.doesNotMatch(other,/Agregar productos|Cerrar cuenta|Cobrar saldo|Cancelar pedido/);
 const delivered={...exampleOrder,status:"entregado",serviceItems:exampleOrder.serviceItems.map(item=>({...item,status:"entregado",statusLabel:"Entregado"}))};
 assert.match(renderDetail({order:delivered}),/Cerrar cuenta/);
 assert.match(renderDetail({order:{...delivered,billClosedAt:"2026-10-09T20:00:00-05:00",accountState:"awaiting_payment"},permissions:["cash.manage"]}),/Cobrar saldo/);
});
test("se conservan detalle vacío, error recuperable, carga y cuenta pagada",()=>{
 assert.match(renderDetail({order:{...exampleOrder,items:[],serviceItems:[]}}),/Sin ítems cargados aún/);
 assert.match(renderDetail({error:"Error de prueba"}),/No pudimos cargar el pedido/);
 assert.match(renderDetail({skeleton:true}),/ds-skeleton/);
 assert.match(renderDetail({order:{...exampleOrder,paidAmount:"30",remainingAmount:"0"}}),/Total pagado/);
});
test("el resumen global es inferior en todos los tamaños y no usa vh estático",()=>{
 const css=read("design-system/styles/modal.css");
 assert.match(css,/\.crud-modal>\.modal-workspace\{display:flex;flex-direction:column/);
 assert.doesNotMatch(css,/grid-template-columns:minmax\(0,1fr\) var\(--size-240\)/);
 assert.match(css,/\.modal-summary-line\{[^}]*flex-direction:column/);
 assert.match(css,/\.modal-workspace>:first-child\{[^}]*overscroll-behavior:contain/);
 assert.doesNotMatch(read("modules/operations/orders/presentation/order-detail.css"),/\b\d+vh\b/);
});
test("la cuenta distingue abierta, por cobrar y pagada sin cambiar el estado del pedido",()=>{
 assert.match(renderDetail(),/status sky emphasized[^>]*><svg[\s\S]*?<\/svg>Cuenta abierta/);
 assert.match(renderDetail({order:{...exampleOrder,billClosedAt:"2026-10-09T20:00:00-05:00"}}),/status orange emphasized[^>]*><svg[\s\S]*?<\/svg>Por cobrar/);
 assert.match(renderDetail({order:{...exampleOrder,accountState:"paid",paidAmount:"30",remainingAmount:"0"}}),/status green emphasized[^>]*><svg[\s\S]*?<\/svg>Pagada/);
 assert.match(renderDetail({order:{...exampleOrder,accountState:"paid",paidAmount:"30",remainingAmount:"0",completedAt:"2026-10-10T13:00:00-05:00"}}),/status gray emphasized[^>]*><svg[\s\S]*?<\/svg>Finalizada/);
});
test("resumen agrupado y estado de cuenta destacado sin recolorear los demás estados",()=>{
 const css=read("design-system/styles/modal.css"),global=read("styles/globals.css"),html=renderDetail();
 const heading=html.slice(html.indexOf('class="modal-summary-heading"'),html.indexOf('class="modal-summary-lines"'));
 assert.match(heading,/Resumen de cuenta[\s\S]*Cuenta abierta/);
 assert.match(css,/\.modal-summary-heading\{[^}]*flex-wrap:wrap[^}]*border-bottom:var\(--stroke-1\) solid var\(--line\)/,'el resumen abre con la misma línea fina que las secciones');
 assert.match(css,/\.modal-summary-lines\{[^}]*grid-template-columns:repeat\(2,minmax\(0,1fr\)\)/);
 assert.match(global,/\.status.blue\{color:var\(--primary-700\);background:var\(--primary-100\)/);
 assert.match(global,/\.status.sky\{color:var\(--sky-800\);background:var\(--sky-100\)/);
 assert.doesNotMatch(global,/\.status\.(?:blue|sky|violet)\.emphasized\{/);
 assert.match(css,/\.modal-summary\{grid-template-columns:minmax\(0,2fr\) minmax\(0,1fr\)/);
 assert.match(html,/class="status blue"/);assert.match(html,/class="status orange"/);
 assert.doesNotMatch(heading,/<button|href=/);
});
test("el encabezado de productos no repite unidades y conserva cantidades por producto",()=>{
 const html=renderDetail();
 assert.doesNotMatch(html,/\b\d+ unidades?\b/);
 assert.equal((html.match(/class="modal-item-quantity order-detail-qty"/g)??[]).length,exampleOrder.items.length);
 assert.equal(detailComponents().accountTone(exampleOrder),"sky");
});
test("los cuatro estados de cuenta tienen tonos distintos sin recolorear la preparación",()=>{
 const {accountTone,serviceStatusTone,orderStatus}=detailComponents();
 const states=[exampleOrder,{...exampleOrder,billClosedAt:"2026-10-10"},{...exampleOrder,accountState:"paid"},{...exampleOrder,accountState:"paid",completedAt:"2026-10-10"}];
 assert.deepEqual(states.map(accountTone),["sky","orange","green","gray"]);
 assert.notEqual(accountTone(exampleOrder),orderStatus("confirmado").tone);
 assert.equal(new Set(states.map(accountTone)).size,4);
 assert.equal(serviceStatusTone("preparando"),"violet");
 assert.equal(serviceStatusTone("entregado"),"teal");
});
