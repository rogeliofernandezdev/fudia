import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import postcss from "postcss";
import {resolveDesignScales} from "./helpers/design-tokens.mjs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const source=readFileSync(new URL("../src/modules/auth/presentation/login.css",import.meta.url),"utf8");
const ast=postcss.parse(resolveDesignScales(source));
const values=selector=>Object.fromEntries(ast.nodes.find(node=>node.selector===selector).nodes.filter(node=>node.type==="decl").map(node=>[node.prop,node.value]));

test("acceso: una sola indicación de acceso autorizado y legible",()=>{
 const row=values(".admin-auth-trust"),item=values(".admin-auth-trust li"),icon=values(".admin-auth-trust svg");
 assert.equal(row.display,"flex");assert.equal(row["align-items"],"center");assert.equal(row["justify-content"],"center");
 assert.equal(item.display,"flex");assert.equal(item["align-items"],"center");assert.equal(item["white-space"],undefined);
 assert.equal(icon["flex-shrink"],"0");
 assert.equal(source.includes("text-overflow"),false);assert.equal(source.includes("overflow:hidden"),true,"solo la tarjeta conserva su borde");
 const form=readFileSync(new URL("../src/modules/auth/presentation/login-form.tsx",import.meta.url),"utf8");
 assert.ok(form.includes("Acceso reservado al personal autorizado"));
 assert.doesNotMatch(form,/Conexión segura|Usuario validado|Bienvenido/);
});

test("acceso: nota de seguridad legible en móvil sin apilarse",()=>{
 assert.equal(values(".admin-auth-trust li")["font-size"],"12px");
 const mobile=ast.nodes.find(node=>node.type==="atrule"&&node.params==="(max-width:480px)");
 assert.equal(mobile.nodes.find(node=>node.selector===".admin-auth-trust").nodes.find(node=>node.prop==="padding-inline").value,"16px");
 ast.walkRules(rule=>{if(rule.selector===".admin-auth-trust"||rule.selector===".admin-auth-trust li"){
  assert.equal(rule.nodes.some(node=>node.prop==="flex-direction"&&node.value==="column"),false);
  assert.equal(rule.nodes.some(node=>node.prop==="flex-wrap"&&node.value==="wrap"),false);
 }});
});

test("acceso: tarjeta móvil compacta, campos neutros y acción con geometría compartida",()=>{
 assert.equal(values(".admin-auth-shell")["max-width"],"420px");
 assert.equal(values(".admin-auth-brand img").width,"160px");
 assert.equal(values(".admin-auth-card").background,"var(--surface)");
 assert.equal(values(".admin-auth-field .admin-auth-input").background,"var(--surface)");
 assert.equal(values(".admin-auth-submit")["min-height"],"calc(var(--control-height) + var(--space-1))");
 assert.equal(values(".admin-auth-submit").width,"100%");
 assert.equal(values(".admin-auth-submit")["max-width"],undefined);
 assert.equal(values(".admin-auth-actions").gap,"8px");
 assert.equal(values(".admin-auth-submit")["margin-top"],undefined);
 assert.equal(values(".admin-auth-submit")["border-radius"],"6px");
 assert.equal(values(".admin-auth-field button").height,"var(--control-height)");
 assert.equal(values(".admin-auth-field button").width,"var(--control-height)");
 assert.equal(values(".admin-auth-footer")["font-size"],"12px");
});

test("acceso: vista baja conserva desplazamiento natural y foco visible",()=>{
 const low=ast.nodes.find(node=>node.type==="atrule"&&node.params==="(max-height:540px)");
 const page=Object.fromEntries(low.nodes.find(node=>node.selector===".admin-auth-page").nodes.map(node=>[node.prop,node.value]));
 assert.equal(page["align-items"],"start");
 assert.equal(values(".admin-auth-page")["min-height"],"100dvh");
 assert.equal(values(".admin-auth-page").height,undefined);
 assert.equal(values(".admin-auth-page").overflow,undefined);
 assert.ok(values(".admin-auth-field button:focus-visible").outline.includes("var(--primary-600)"));
 assert.ok(values(".admin-auth-submit:focus-visible").outline.includes("var(--primary-600)"));
});

test("acceso: Ingresar conserva estados y no incorpora selectores ni enlaces sin destino",()=>{
 const form=readFileSync(new URL("../src/modules/auth/presentation/login-form.tsx",import.meta.url),"utf8");
 assert.match(form,/>Ingresar<Icon/);
 assert.match(form,/aria-busy=\{loading\}/);
 assert.match(form,/Verificando acceso…/);
 assert.match(form,/aria-pressed=\{showPassword\}/);
 assert.match(form,/autoComplete="username"/);
 assert.match(form,/autoComplete="current-password"/);
 assert.match(form,/<h1 id="login-title">Iniciar sesión<\/h1>/);
 assert.match(form,/<p>Ingresa con la cuenta asignada a tu local\.<\/p>/);
 assert.doesNotMatch(form,/Bienvenido|<Select|href="#"/);
 assert.match(form,/firstAccessibleRoute\(context\)/);
});

test("acceso: Recuperar contraseña muestra solo Próximamente sin solicitudes ni envío simulado",()=>{
 const require=createRequire(import.meta.url),exports={},state=[false,false,"","",false,false];
 let cursor=0,calls=0;
 const source=readFileSync(new URL("../src/modules/auth/presentation/login-form.tsx",import.meta.url),"utf8");
 const imports={
  "react":{useEffect(){},useState:()=>{const index=cursor++;return[state[index],value=>{state[index]=value;}];}},
  "react-hook-form":{useForm:()=>({register:name=>({name}),handleSubmit:fn=>fn,formState:{errors:{}}})},
  "next/navigation":{useRouter:()=>({})},"next/image":{default:"img"},
  "@tanstack/react-query":{useQueryClient:()=>({})},
  "@/design-system/icons":{Icon:"svg"},
  "@/design-system":{Button:"Button",Dialog:"Dialog",ModalHeader:"ModalHeader",FormField:"FormField",Input:"input"},
  "../infrastructure/auth-api":{login:()=>calls++,logout:()=>calls++},
  "../domain/login-schema":{},
  "@/shared/session/session-api":{loadSessionContext:()=>calls++},
  "@/shell/navigation":{},"@/shared/session/session-events":{},
 };
 vm.runInNewContext(ts.transpileModule(source,{compilerOptions:{target:ts.ScriptTarget.ES2020,module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:name=>name.endsWith(".css")?{}:imports[name]??require(name)});
 const render=()=>{cursor=0;return exports.LoginForm();};
 function find(node,predicate){if(!node||typeof node!=="object")return null;if(Array.isArray(node))return node.map(child=>find(child,predicate)).find(Boolean)??null;if(predicate(node))return node;return find(node.props?.children,predicate);}
 const opener=find(render(),node=>node.props?.className==="admin-auth-recovery");
 assert.equal(opener.props.type,"button");assert.equal(opener.props.children,"Recuperar contraseña");
 opener.props.onClick();
 const dialog=find(render(),node=>node.type==="Dialog");
 assert.ok(dialog);assert.equal(dialog.props["aria-labelledby"],"login-recovery-title");
 assert.ok(dialog.props.onResponseClose);
 assert.equal(find(dialog,node=>node.type==="form"),null);
 const title=find(dialog,node=>node.type==="ModalHeader");
 assert.equal(title.props.title,"Próximamente");
 assert.equal(find(dialog,node=>node.type==="p"||node.type==="small"),null);
 const textContent=node=>typeof node==="string"?node:Array.isArray(node)?node.map(textContent).join(""):node&&typeof node==="object"?textContent(node.props?.children):"";
 assert.equal(textContent(dialog),"");
 assert.equal(calls,0,"La ayuda no consulta identidad ni envía correos");
 assert.equal(title.props.closeLabel,"Cerrar ayuda");
 title.props.onClose();assert.equal(find(render(),node=>node.type==="Dialog"),null);
});

test("acceso: recuperar conserva texto plano al pasar el cursor y foco visible con teclado",()=>{
 assert.equal(ast.nodes.some(node=>node.selector===".admin-auth-recovery:hover"),false);
 assert.equal(values(".admin-auth-recovery").background,"transparent");
 assert.equal(values(".admin-auth-recovery")["box-shadow"],undefined);
 assert.equal(values(".admin-auth-recovery")["text-decoration"],undefined);
 assert.ok(values(".admin-auth-recovery:focus-visible").outline.includes("var(--primary-600)"));
 assert.equal(ast.nodes.some(node=>node.selector===".admin-auth-field button:hover"),false);
 assert.equal(values(".admin-auth-field button").background,"transparent");
 assert.equal(values(".admin-auth-field button")["box-shadow"],undefined);
 assert.equal(values(".admin-auth-field button:focus-visible").background,undefined);
});

test("acceso: conserva una tarjeta y el orden original; destaca marca y campos sin panel lateral",()=>{
 assert.equal(values(".admin-auth-shell")["max-width"],"420px");
 assert.equal(values(".admin-auth-card")["grid-template-columns"],undefined);
 assert.ok(values(".admin-auth-brand").background.includes("linear-gradient"));
 assert.equal(values(".admin-auth-heading h1")["font-size"],"22px");
 assert.equal(values(".admin-auth-heading p").color,"var(--ink-600)");
 assert.equal(values(".admin-auth-field:focus-within .admin-auth-input:not([aria-invalid=\"true\"])")["box-shadow"],"var(--ring-focus)");
 assert.equal(values(".admin-auth-field>svg").background,"var(--primary-100)");
 assert.equal(values(".admin-auth-field>svg").width,"26px");
 assert.equal(values(".admin-auth-accent").height,"4px");
 assert.equal(source.includes("min-width:900px"),false);
 assert.equal(source.includes("admin-auth-brand-copy"),false);
 const form=readFileSync(new URL("../src/modules/auth/presentation/login-form.tsx",import.meta.url),"utf8");
 assert.match(form,/<div className="admin-auth-body">\s*<div className="admin-auth-brand">/);
 assert.match(form,/<form className="admin-auth-form"[\s\S]*?<header className="admin-auth-heading">/);
 assert.doesNotMatch(form,/Gestión de restaurantes|Áreas del sistema|Bienvenido/);
});

test("acceso: al ingresar pasa al loader FUDIA y no vuelve a habilitar el botón antes de navegar",()=>{
 const form=readFileSync(new URL("../src/modules/auth/presentation/login-form.tsx",import.meta.url),"utf8");
 assert.match(form,/if\(redirecting\)return <FullScreenLoader label="Ingresando a tu espacio"\/>/);
 assert.match(form,/await login\(values\);[\s\S]*await enterWorkspace\(\);/);
 assert.match(form,/router\.prefetch\(route\);\s*router\.replace\(route\);/);
 assert.doesNotMatch(form,/finally\s*\{\s*setLoading\(false\)/,"El botón solo se libera si el ingreso falla");
 assert.match(form,/catch \(reason\) \{\s*setRedirecting\(false\);\s*setLoading\(false\);/);
});
