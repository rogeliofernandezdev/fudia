import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
const require=createRequire(import.meta.url);
const source=path=>readFileSync(new URL(`../src/${path}`,import.meta.url),"utf8");
function compile(path,resolve){const exports={};vm.runInNewContext(ts.transpileModule(source(path),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,target:ts.ScriptTarget.ES2022}}).outputText,{exports,require:resolve});return exports;}
const routes=compile("shared/routing/page-routes.ts",require);
const navigation=compile("shell/navigation.ts",name=>name==="@/shared/routing/page-routes"?routes:require(name));
const grouping=compile("modules/identity/presentation/permission-navigation.ts",name=>name==="@/shell/navigation"?navigation:routes);
const {permissionSections,filterPermissionSections}=grouping;
const {RoleActionTree}=compile("modules/identity/presentation/role-action-tree.tsx",name=>name==="@/design-system/icons"?{Icon:"Icon"}:name==="@/design-system/icon-tile"?{IconTile:"IconTile"}:name==="./permission-navigation"?grouping:require(name));
const item=(value,label=value)=>({value,label});
const catalog=[{group:"Administración",items:[item("users.manage","Administrar usuarios y roles"),item("orders.manage","Gestionar pedidos"),item("cash.manage","Operar turnos y cobros"),item("kitchen.manage","Preparar en Cocina"),item("bar.manage","Preparar en Barra"),item("menu.read","Ver carta y productos"),item("menu.manage","Administrar carta y productos")]}];
const plain=value=>JSON.parse(JSON.stringify(value));
// Acciones por pantalla: misma fuente que el API (permission-catalog → screenActions).
const screenActions=Object.fromEntries(readFileSync(new URL("../../foods-backend/internal/modules/identity/domain/screen_actions.go",import.meta.url),"utf8").split(/\r?\n/).flatMap(line=>{const match=line.match(/^\t"(\w+)":\s*\{(.*)\},$/);return match?[[match[1],[...match[2].matchAll(/\{"([^"]+)", "([^"]+)"\}/g)].map(([,value,label])=>({value,label}))]]:[];}));
const sectionsOf=(groups,menus)=>permissionSections(groups,menus,screenActions);
function nodes(tree,result=[]){if(Array.isArray(tree))tree.forEach(node=>nodes(node,result));else if(tree&&typeof tree==="object"){result.push(tree);nodes(tree.props?.children,result);}return result;}

test("Acciones usa exactamente secciones, módulos, iconos y orden del menú",()=>{
 const sections=sectionsOf(catalog);
 assert.ok(Object.keys(screenActions).length>20,"se leyeron las acciones por pantalla del backend");
 assert.deepEqual(plain(sections.map(section=>section.name)),["Control","Operación","Carta y producción","Configuración"]);
 assert.deepEqual(plain(sections[1].modules.map(module=>module.name)),["Punto de venta","Pedidos","Salón","Cocina","Mesas y zonas","Caja y turnos"]);
 for(const section of sections)for(const feature of section.modules){const nav=navigation.navigationGroups.flatMap(group=>group.items).find(item=>item.href===feature.id);assert.equal(feature.name,nav.name);assert.equal(feature.icon,nav.icon);}
 assert.equal(sections.some(section=>section.name==="Administración"),false);
 assert.deepEqual(plain(permissionSections([])),[]);
});

test("el catálogo remoto se conserva completo sin inventar acciones ni perder permisos futuros",()=>{
 const backend=readFileSync(new URL("../../foods-backend/internal/modules/identity/domain/administration.go",import.meta.url),"utf8").split("var PermissionCatalog =")[1].split("// PermissionLabel")[0];
 const values=[...backend.matchAll(/"value": "([^"]+)"/g)].map(match=>match[1]);
 const sections=sectionsOf([{group:"catálogo efectivo",items:values.map(value=>item(value,`Etiqueta genérica: ${value}`))}]);
 const rendered=sections.flatMap(section=>section.modules.flatMap(module=>module.items));
 assert.deepEqual([...new Set(rendered.map(item=>item.value))].sort(),[...values].sort());
 assert.equal(sections.some(section=>section.name==="Otros permisos"),false,"todos los permisos actuales tienen pantalla");
 assert.ok(rendered.every(item=>!item.label.startsWith("Etiqueta genérica:")),"cada pantalla usa su propia etiqueta");
 const receipts=sections.flatMap(section=>section.modules).find(module=>module.name==="Comprobantes");
 assert.deepEqual(plain(receipts.items.map(item=>item.label)),["Ver y descargar comprobantes","Reenviar comprobantes a SUNAT"]);
 assert.ok(receipts.items[0].sharedWith.includes("Pedidos"),"la acción compartida avisa dónde más aplica");
 assert.equal(grouping.sharedNote(["Ventas","Pedidos","Salón","Cocina","Punto de venta"]),"También en Ventas, Pedidos, Salón y 2 más");
 assert.equal(grouping.sharedNote(["Caja y turnos"]),"También en Caja y turnos");
 assert.equal(grouping.sharedNote(["Pedidos","Salón"]),"También en Pedidos y Salón");
 const future=sectionsOf([{group:"API",items:[item("future.manage","Acción nueva del API")]}]);assert.equal(future[0].modules[0].items[0].label,"Acción nueva del API");
});

test("búsqueda por sección, módulo o acción conserva contexto y los filtros no alteran totales",()=>{
 const sections=sectionsOf(catalog);
 assert.deepEqual(plain(filterPermissionSections(sections,"salon","all",[]).flatMap(section=>section.modules.map(module=>module.name))),["Salón"]);
 assert.deepEqual(plain(filterPermissionSections(sections,"operacion","all",[]).map(section=>section.name)),["Operación"]);
 const filtered=filterPermissionSections(sections,"barra","assigned",["bar.manage"]);
 assert.equal(filtered[0].modules[0].name,"Cocina");assert.equal(filtered[0].modules[0].items.length,2);assert.equal(filtered[0].modules[0].visibleItems.length,1);
 assert.equal(filterPermissionSections(sections,"no existe","all",[]).length,0);
});

test("permisos compartidos reflejan una selección y la selección masiva respeta requeridos y filtro",()=>{
 const sections=sectionsOf(catalog),calls=[];
 const view=nodes(RoleActionTree({sections,selected:["orders.manage","menu.read"],required:new Map([["menu.read",["Carta y productos"]]]),search:"carta y productos",onToggleModule:values=>calls.push(values)}));
 const checks=view.filter(node=>node.type==="input");assert.ok(checks.some(node=>node.props.checked&&node.props.disabled));
 for(const button of view.filter(node=>node.props.className?.startsWith("re-acc-toggle")&&!node.props.disabled))button.props.onClick();
 assert.ok(calls.length);assert.ok(calls.every(values=>values.length===1&&values[0]==="menu.manage"));
 const shared=nodes(RoleActionTree({sections,selected:["orders.manage"]})).filter(node=>node.type==="input"&&node.props.checked);assert.equal(shared.length,3,"Comprobantes, Pedidos y Salón muestran el mismo permiso marcado");
 const ids=view.filter(node=>node.type==="small"&&node.props.id).map(node=>node.props.id);assert.ok(ids.length);assert.equal(new Set(ids).size,ids.length,"motivos compartidos con IDs accesibles únicos");
 const readonly=nodes(RoleActionTree({sections,selected:["orders.manage"],show:"assigned",readOnly:true}));assert.equal(readonly.some(node=>node.type==="input"||node.type==="button"),false);
});


test("las acciones no muestran módulos fuera del catálogo efectivo del plan",()=>{
 const sections=sectionsOf(catalog,[{group:"API",items:[item("pedidos","Pedidos"),item("salon","Salón")]}]);
 assert.deepEqual(plain(sections.find(section=>section.name==="Operación").modules.map(module=>module.name)),["Pedidos","Salón"]);
 // Cada pantalla tiene su propia clave: con solo «pedidos», Salón no aparece.
 assert.deepEqual(plain(sectionsOf(catalog,[{group:"API",items:[item("pedidos","Pedidos")]}]).find(section=>section.name==="Operación").modules.map(module=>module.name)),["Pedidos"]);
 assert.ok(!sections.some(section=>section.name==="Carta y producción"));
});


test("cada módulo incluye su permiso de lectura oficial aunque sea compartido",()=>{
 const actions=[...new Set(navigation.navigationGroups.flatMap(group=>group.items.flatMap(item=>item.permission?[item.permission]:[])))];
 const sections=sectionsOf([{group:"API",items:actions.map(value=>item(value))}]);
 for(const entry of navigation.navigationGroups.flatMap(group=>group.items).filter(item=>item.permission)){
  const feature=sections.flatMap(section=>section.modules).find(feature=>feature.id===entry.href);
  assert.ok(feature?.items.some(item=>item.value===entry.permission),entry.name);
 }
 const scoped=sectionsOf([{group:"API",items:[item("orders.read"),item("orders.manage")]}],[{group:"Operación",items:[item("pos")]}]);
 assert.equal(scoped[0].modules[0].name,"Punto de venta");assert.deepEqual(plain(scoped[0].modules[0].items.map(item=>item.value)),["orders.read"]);
 // Sin una pantalla del plan que la muestre, la acción sigue editable en «Otros permisos».
 assert.deepEqual(plain(scoped.map(section=>section.name)),["Operación","Otros permisos"]);
});


test("buscar abre los resultados, pero el usuario puede volver a plegar un módulo",()=>{
 const calls=[],sections=sectionsOf(catalog);
 const view=nodes(RoleActionTree({sections,selected:[],search:"pedidos",openGroups:{"actions-/orders":false},onOpen:key=>calls.push(key)}));
 const heading=view.find(node=>node.type==="button"&&node.props["aria-controls"]==="actions--orders");
 assert.equal(heading.props["aria-expanded"],false);heading.props.onClick();assert.equal(calls[0],"actions-/orders");
 const manager=source("modules/identity/presentation/users-roles-manager.tsx");assert.ok(manager.includes('setSearch(e.target.value);setOpenGroups({})'));
});

test("cada opción del menú lateral tiene su propia clave de acceso (rol › Menú refleja el menú)",()=>{
 const items=navigation.navigationGroups.flatMap(group=>group.items).filter(item=>!item.platformAdminOnly);
 const keys=items.map(item=>navigation.accessKey(item));
 const repeated=keys.filter((key,index)=>keys.indexOf(key)!==index);
 assert.equal(repeated.join(", "),"","dos pantallas no comparten casilla en el editor de roles");
});
