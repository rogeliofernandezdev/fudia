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
function nodes(tree,result=[]){if(Array.isArray(tree))tree.forEach(node=>nodes(node,result));else if(tree&&typeof tree==="object"){result.push(tree);nodes(tree.props?.children,result);}return result;}

test("Acciones usa exactamente secciones, módulos, iconos y orden del menú",()=>{
 const sections=permissionSections(catalog);
 assert.deepEqual(plain(sections.map(section=>section.name)),["Operación","Carta y producción","Configuración"]);
 assert.deepEqual(plain(sections[0].modules.map(module=>module.name)),["Punto de venta","Pedidos","Salón","Cocina","Caja y turnos"]);
 for(const section of sections)for(const feature of section.modules){const nav=navigation.navigationGroups.flatMap(group=>group.items).find(item=>item.href===feature.id);assert.equal(feature.name,nav.name);assert.equal(feature.icon,nav.icon);}
 assert.equal(sections.some(section=>section.name==="Administración"),false);
 assert.deepEqual(plain(permissionSections([])),[]);
});

test("el catálogo remoto se conserva completo sin inventar acciones ni perder permisos futuros",()=>{
 const backend=readFileSync(new URL("../../foods-backend/internal/modules/identity/domain/administration.go",import.meta.url),"utf8").split("var PermissionCatalog =")[1].split("// PermissionLabel")[0];
 const values=[...backend.matchAll(/"value": "([^"]+)"/g)].map(match=>match[1]);
 const sections=permissionSections([{group:"catálogo efectivo",items:values.map(value=>item(value,`Etiqueta del API: ${value}`))}]);
 const rendered=sections.flatMap(section=>section.modules.flatMap(module=>module.items));
 assert.deepEqual([...new Set(rendered.map(item=>item.value))].sort(),[...values].sort());
 assert.equal(sections.some(section=>section.name==="Otros permisos"),false,"todos los permisos actuales tienen módulo");
 assert.ok(rendered.every(item=>item.label.startsWith("Etiqueta del API:")));
 const future=permissionSections([{group:"API",items:[item("future.manage","Acción nueva del API")]}]);assert.equal(future[0].modules[0].items[0].label,"Acción nueva del API");
});

test("búsqueda por sección, módulo o acción conserva contexto y los filtros no alteran totales",()=>{
 const sections=permissionSections(catalog);
 assert.deepEqual(plain(filterPermissionSections(sections,"salon","all",[]).flatMap(section=>section.modules.map(module=>module.name))),["Salón"]);
 assert.deepEqual(plain(filterPermissionSections(sections,"operacion","all",[]).map(section=>section.name)),["Operación"]);
 const filtered=filterPermissionSections(sections,"barra","assigned",["bar.manage"]);
 assert.equal(filtered[0].modules[0].name,"Cocina");assert.equal(filtered[0].modules[0].items.length,2);assert.equal(filtered[0].modules[0].visibleItems.length,1);
 assert.equal(filterPermissionSections(sections,"no existe","all",[]).length,0);
});

test("permisos compartidos reflejan una selección y la selección masiva respeta requeridos y filtro",()=>{
 const sections=permissionSections(catalog),calls=[];
 const view=nodes(RoleActionTree({sections,selected:["orders.manage","menu.read"],required:new Map([["menu.read",["Carta y productos"]]]),search:"carta y productos",onToggleModule:values=>calls.push(values)}));
 const checks=view.filter(node=>node.type==="input");assert.ok(checks.some(node=>node.props.checked&&node.props.disabled));
 for(const button of view.filter(node=>node.props.className?.startsWith("re-acc-toggle")&&!node.props.disabled))button.props.onClick();
 assert.ok(calls.length);assert.ok(calls.every(values=>values.length===1&&values[0]==="menu.manage"));
 const shared=nodes(RoleActionTree({sections,selected:["orders.manage"]})).filter(node=>node.type==="input"&&node.props.checked);assert.equal(shared.length,3,"Punto de venta, Pedidos y Salón muestran el mismo permiso marcado");
 const ids=view.filter(node=>node.type==="small").map(node=>node.props.id);assert.equal(new Set(ids).size,ids.length,"motivos compartidos con IDs accesibles únicos");
 const readonly=nodes(RoleActionTree({sections,selected:["orders.manage"],show:"assigned",readOnly:true}));assert.equal(readonly.some(node=>node.type==="input"||node.type==="button"),false);
});


test("las acciones no muestran módulos fuera del catálogo efectivo del plan",()=>{
 const sections=permissionSections(catalog,[{group:"API",items:[item("pedidos","Pedidos")]}]);
 assert.deepEqual(plain(sections.find(section=>section.name==="Operación").modules.map(module=>module.name)),["Pedidos","Salón"]);
 assert.ok(!sections.some(section=>section.name==="Carta y producción"));
});


test("cada módulo incluye su permiso de lectura oficial aunque sea compartido",()=>{
 const actions=[...new Set(navigation.navigationGroups.flatMap(group=>group.items.flatMap(item=>item.permission?[item.permission]:[])))];
 const sections=permissionSections([{group:"API",items:actions.map(value=>item(value))}]);
 for(const entry of navigation.navigationGroups.flatMap(group=>group.items).filter(item=>item.permission)){
  const feature=sections.flatMap(section=>section.modules).find(feature=>feature.id===entry.href);
  assert.ok(feature?.items.some(item=>item.value===entry.permission),entry.name);
 }
 const scoped=permissionSections([{group:"API",items:[item("orders.read"),item("orders.manage")]}],[{group:"Operación",items:[item("pos")]}]);
 assert.equal(scoped[0].modules[0].name,"Punto de venta");assert.equal(scoped.length,1);
});


test("buscar abre los resultados, pero el usuario puede volver a plegar un módulo",()=>{
 const calls=[],sections=permissionSections(catalog);
 const view=nodes(RoleActionTree({sections,selected:[],search:"pedidos",openGroups:{"actions-/orders":false},onOpen:key=>calls.push(key)}));
 const heading=view.find(node=>node.type==="button"&&node.props["aria-controls"]==="actions--orders");
 assert.equal(heading.props["aria-expanded"],false);heading.props.onClick();assert.equal(calls[0],"actions-/orders");
 const manager=source("modules/identity/presentation/users-roles-manager.tsx");assert.ok(manager.includes('setSearch(e.target.value);setOpenGroups({})'));
});
