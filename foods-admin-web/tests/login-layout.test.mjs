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

test("acceso: los tres mensajes y sus iconos comparten una sola fila",()=>{
 const row=values(".admin-auth-trust"),item=values(".admin-auth-trust li"),icon=values(".admin-auth-trust svg");
 assert.equal(row.display,"flex");assert.equal(row["align-items"],"center");assert.equal(row["justify-content"],"space-between");
 assert.equal(item.display,"flex");assert.equal(item["align-items"],"center");assert.equal(item["white-space"],"nowrap");
 assert.equal(icon["flex-shrink"],"0");
 assert.equal(source.includes("text-overflow"),false);assert.equal(source.includes("overflow:hidden"),true,"solo la tarjeta conserva su borde");
 const form=readFileSync(new URL("../src/modules/auth/presentation/login-form.tsx",import.meta.url),"utf8");
 for(const label of ["Conexión segura","Acceso por empresa","Usuario validado"])assert.ok(form.includes(label));
});

test("acceso: tipografía auxiliar legible en móvil sin volver a apilar mensajes",()=>{
 assert.equal(values(".admin-auth-trust li")["font-size"],"11px");
 const mobile=ast.nodes.find(node=>node.type==="atrule"&&node.params==="(max-width:480px)");
 assert.equal(mobile.nodes.find(node=>node.selector===".admin-auth-trust li").nodes.find(node=>node.prop==="font-size").value,"10px");
 ast.walkRules(rule=>{if(rule.selector===".admin-auth-trust"||rule.selector===".admin-auth-trust li"){
  assert.equal(rule.nodes.some(node=>node.prop==="flex-direction"&&node.value==="column"),false);
  assert.equal(rule.nodes.some(node=>node.prop==="flex-wrap"&&node.value==="wrap"),false);
 }});
});

test("acceso: tarjeta móvil compacta, campos neutros y acción con geometría compartida",()=>{
 assert.equal(values(".admin-auth-shell")["max-width"],"400px");
 assert.equal(values(".admin-auth-brand img").width,"180px");
 assert.equal(values(".admin-auth-card").background,"var(--surface)");
 assert.equal(values(".admin-auth-field .admin-auth-input").background,"var(--surface)");
 assert.equal(values(".admin-auth-submit").height,"var(--control-height)");
 assert.equal(values(".admin-auth-submit").width,"100%");
 assert.equal(values(".admin-auth-submit")["max-width"],undefined);
 assert.equal(values(".admin-auth-actions").gap,"4px");
 assert.equal(values(".admin-auth-submit")["margin-top"],undefined);
 assert.equal(values(".admin-auth-submit")["border-radius"],"6px");
 assert.equal(values(".admin-auth-field button").height,"var(--control-height)");
 assert.equal(values(".admin-auth-field button").width,"var(--control-height)");
 assert.equal(values(".admin-auth-footer")["font-size"],"11px");
});

test("acceso: vista baja conserva desplazamiento natural y foco visible",()=>{
 const low=ast.nodes.find(node=>node.type==="atrule"&&node.params==="(max-height:540px)");
 const page=Object.fromEntries(low.nodes.find(node=>node.selector===".admin-auth-page").nodes.map(node=>[node.prop,node.value]));
 assert.equal(page["align-items"],"start");
 assert.equal(values(".admin-auth-page")["min-height"],"100dvh");
 assert.equal(values(".admin-auth-page").height,undefined);
 assert.equal(values(".admin-auth-page").overflow,undefined);
 assert.ok(values(".admin-auth-field button:focus-visible").outline.includes("var(--ops-500)"));
 assert.ok(values(".admin-auth-submit:focus-visible").outline.includes("var(--ops-500)"));
});

test("acceso: Ingresar conserva estados y no incorpora selectores ni enlaces sin destino",()=>{
 const form=readFileSync(new URL("../src/modules/auth/presentation/login-form.tsx",import.meta.url),"utf8");
 assert.match(form,/>Ingresar<Icon/);
 assert.match(form,/aria-busy=\{loading\}/);
 assert.match(form,/Verificando acceso…/);
 assert.match(form,/aria-pressed=\{showPassword\}/);
 assert.match(form,/autoComplete="username"/);
 assert.match(form,/autoComplete="current-password"/);
 assert.match(form,/<h1 className="sr-only" id="login-title">Iniciar sesión<\/h1>/);
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
  "@/design-system":{Dialog:"Dialog",FormField:"FormField",Input:"input"},
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
 const title=find(dialog,node=>node.type==="h2");
 assert.equal(title.props.children,"Próximamente");
 assert.equal(find(dialog,node=>node.type==="p"||node.type==="small"),null);
 const textContent=node=>typeof node==="string"?node:Array.isArray(node)?node.map(textContent).join(""):node&&typeof node==="object"?textContent(node.props?.children):"";
 assert.equal(textContent(dialog),"Próximamente");
 assert.equal(calls,0,"La ayuda no consulta identidad ni envía correos");
 const close=find(dialog,node=>node.props?.["aria-label"]==="Cerrar ayuda");
 close.props.onClick();assert.equal(find(render(),node=>node.type==="Dialog"),null);
});

test("acceso: recuperar conserva texto plano al pasar el cursor y foco visible con teclado",()=>{
 assert.equal(ast.nodes.some(node=>node.selector===".admin-auth-recovery:hover"),false);
 assert.equal(values(".admin-auth-recovery").background,"transparent");
 assert.equal(values(".admin-auth-recovery")["box-shadow"],undefined);
 assert.equal(values(".admin-auth-recovery")["text-decoration"],undefined);
 assert.ok(values(".admin-auth-recovery:focus-visible").outline.includes("var(--ops-500)"));
 assert.equal(ast.nodes.some(node=>node.selector===".admin-auth-field button:hover"),false);
 assert.equal(values(".admin-auth-field button").background,"transparent");
 assert.equal(values(".admin-auth-field button")["box-shadow"],undefined);
 assert.equal(values(".admin-auth-field button:focus-visible").background,undefined);
});

test("acceso: conserva una tarjeta y el orden original; destaca marca y campos sin panel lateral",()=>{
 assert.equal(values(".admin-auth-shell")["max-width"],"400px");
 assert.equal(values(".admin-auth-card")["grid-template-columns"],undefined);
 assert.ok(values(".admin-auth-brand").background.includes("linear-gradient"));
 assert.equal(values(".admin-auth-field>svg").background,"var(--primary-100)");
 assert.equal(values(".admin-auth-field>svg").width,"26px");
 assert.equal(values(".admin-auth-accent").height,"4px");
 assert.equal(source.includes("min-width:900px"),false);
 assert.equal(source.includes("admin-auth-brand-copy"),false);
 const form=readFileSync(new URL("../src/modules/auth/presentation/login-form.tsx",import.meta.url),"utf8");
 assert.match(form,/<div className="admin-auth-body">\s*<div className="admin-auth-brand">/);
 assert.doesNotMatch(form,/Gestión de restaurantes|Áreas del sistema|Bienvenido/);
});
