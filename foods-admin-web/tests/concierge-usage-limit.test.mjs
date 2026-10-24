import assert from "node:assert/strict";
import {readFileSync,readdirSync,statSync} from "node:fs";
import {createRequire} from "node:module";
import test from "node:test";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url);
const root=new URL("../",import.meta.url);
const read=file=>readFileSync(new URL(file,root),"utf8");
const platform="src/modules/platform/";
const usageUi=read(`${platform}presentation/platform-concierge-usage.tsx`);

function compile(path,resolve,globals={}){
 const exports={};
 const source=ts.transpileModule(read(path),{compilerOptions:{module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2022,jsx:ts.JsxEmit.ReactJSX}}).outputText;
 vm.runInNewContext(source,{exports,require:resolve,...globals});
 return exports;
}
const regional=compile("src/shared/i18n/regional-format.ts",require);
const domain=compile(`${platform}domain/concierge-usage.ts`,name=>name==="@/shared/i18n/regional-format"?regional:require(name));

function sourceFiles(dir,out=[]){
 for(const name of readdirSync(new URL(dir,root))){
  const path=`${dir}${name}`;
  if(statSync(new URL(path,root)).isDirectory())sourceFiles(`${path}/`,out);
  else if(/\.(ts|tsx)$/.test(name))out.push(path);
 }
 return out;
}

function platformApi(){
 const calls=[];
 const api=compile(`${platform}infrastructure/platform-api.ts`,name=>{
  if(name==="@/shared/api/client")return {ApiClientError:class extends Error{},apiFetch:async()=>({})};
  if(name==="@/shared/session/expire-session")return {expireBrowserSession(){}};
  if(name==="../domain/concierge-usage")return domain;
  return require(name);
 },{
  URLSearchParams,
  fetch:async(url,init={})=>{calls.push({url,method:init.method??"GET",body:init.body?JSON.parse(init.body):undefined});return {status:200,ok:true,headers:{get:()=>null},text:async()=>"{}"}},
 });
 return {api,calls};
}

test("concierge: el plan edita el tope mensual y lo envía como número o null",async()=>{
 const schema=read(`${platform}domain/plan-schema.ts`);
 assert.match(schema,/maxConciergeConversations:optionalInt\("Conversaciones de Concierge"\)/,"Valida entero ≥ 1 o vacío como los demás límites");
 const page=read(`${platform}presentation/platform-plans-page.tsx`);
 assert.match(page,/label="Conversaciones de Concierge al mes" help="Vacío = ilimitado\./);
 assert.match(page,/register\("maxConciergeConversations"\)/);
 assert.match(page,/maxConciergeConversations:plan\.maxConciergeConversations\?\.toString\(\)\?\?""/,"Editar carga el tope del plan en el borrador");
 assert.match(page,/plan\.maxConciergeConversations===null\?"Ilimitado"/,"La tarjeta muestra el tope junto a los demás límites");
 assert.match(read(`${platform}presentation/platform-skeletons.tsx`),/plan-capacity-wide/,"El skeleton reserva la fila del tope");

 const {api,calls}=platformApi();
 const draft={id:"plan-1",code:"impulso",name:"Impulso",description:"",currency:"PEN",monthlyPrice:"99",annualPrice:"990",trialDays:"14",maxLocations:"",maxUsers:"8",maxConciergeConversations:"",moduleKeys:["whatsapp_bot"],termsVersion:"2026-10",active:true};
 await api.saveSubscriptionPlan(draft);
 await api.saveSubscriptionPlan({...draft,id:undefined,maxConciergeConversations:" 2400 "});
 assert.deepEqual(calls.map(call=>[call.method,call.url]),[["PATCH","/api/platform/plans/plan-1"],["POST","/api/platform/plans"]]);
 assert.equal(calls[0].body.maxConciergeConversations,null,"Vacío = ilimitado");
 assert.equal(calls[1].body.maxConciergeConversations,2400);
 assert.equal(calls[0].body.maxUsers,8);
 assert.equal(calls[0].body.maxLocations,null);
});

test("concierge: el tope no se agrega al plan que ven las empresas",()=>{
 const types=read(`${platform}domain/types.ts`);
 const companyPlan=types.slice(types.indexOf("export type SubscriptionPlan="),types.indexOf("};",types.indexOf("export type SubscriptionPlan=")));
 assert.doesNotMatch(companyPlan,/maxConciergeConversations/);
 assert.match(types,/export type PlatformSubscriptionPlan=SubscriptionPlan&\{maxConciergeConversations:number\|null\}/);
 assert.match(types,/plan:SubscriptionPlan;/,"La suscripción de la empresa conserva el plan sin tope de Concierge");
 const api=read(`${platform}infrastructure/platform-api.ts`);
 assert.match(api,/getCurrentOrganizationSubscription\(\)\{\s*return apiFetch<OrganizationSubscription>\("subscription"\)/);
 assert.doesNotMatch(read(`${platform}index.ts`),/Concierge|PlatformSubscriptionPlan/,"La API pública del módulo no expone el uso a otros módulos");
});

test("concierge: uso y tope solo existen en Plataforma",()=>{
 const pattern=/maxConciergeConversations|concierge-usage|ConciergeUsage|ConciergeLimit|listConciergeUsage/;
 const leaks=sourceFiles("src/").filter(file=>!file.startsWith(platform)&&pattern.test(read(file)));
 assert.deepEqual(leaks,[],"Pantallas de empresa, navegación y Admin › Fudia Concierge no consultan tope ni consumo");
 const users=sourceFiles(platform).filter(file=>/ConciergeUsage(Panel|Cell|Notice|DialogSection|SettingsPanel)\b/.test(read(file))&&!file.endsWith("platform-concierge-usage.tsx"));
 assert.deepEqual(users.sort(),[
  `${platform}presentation/platform-companies-page.tsx`,
  `${platform}presentation/platform-company-detail-dialog.tsx`,
  `${platform}presentation/platform-company-detail-page.tsx`,
  `${platform}presentation/platform-global-settings-page.tsx`,
 ]);
 assert.match(read(`${platform}presentation/platform-companies-page.tsx`),/if\(!user\?\.platformAdmin\)return <RestrictedCompanies\/>/);
 assert.match(read(`${platform}presentation/platform-company-detail-page.tsx`),/if\(!user\?\.platformAdmin\)return <RestrictedCompanies\/>/);
 assert.match(read("src/app/platform/configuracion-global/page.tsx"),/PlatformGlobalSettingsPage/,"Configuración global vive bajo el shell de Plataforma");
 assert.match(read("src/modules/platform/presentation/platform-shell.tsx"),/!user\?\.platformAdmin/);
});

test("concierge: la API usa las rutas de plataforma del contrato",async()=>{
 const {api,calls}=platformApi();
 await api.getConciergeUsageSettings();
 await api.updateConciergeUsageSettings({mode:"enforce",warnPercent:80});
 await api.listConciergeUsage(["org-a","org-b"]);
 await api.listConciergeUsage(Array.from({length:120},(_,index)=>`org-${index}`));
 await api.getOrganizationConciergeUsage("org-a");
 await api.updateOrganizationConciergeLimit("org-a",{limitSource:"custom",maxConversations:900});
 assert.deepEqual(calls.slice(0,3).map(call=>[call.method,call.url]),[
  ["GET","/api/platform/concierge-usage/settings"],
  ["PATCH","/api/platform/concierge-usage/settings"],
  ["GET","/api/platform/concierge-usage?organizationId=org-a&organizationId=org-b"],
 ]);
 assert.equal(new URL(calls[3].url,"http://x").searchParams.getAll("organizationId").length,100,"Máximo 100 empresas por consulta");
 assert.deepEqual(calls.slice(4).map(call=>[call.method,call.url]),[["GET","/api/platform/organizations/org-a/concierge-usage"],["PATCH","/api/platform/organizations/org-a/concierge-usage"]]);
 assert.deepEqual(calls[1].body,{mode:"enforce",warnPercent:80});
 assert.deepEqual(calls[5].body,{limitSource:"custom",maxConversations:900});
 assert.match(read("src/app/api/platform/[...path]/route.ts"),/export const PATCH=proxy/,"El proxy de plataforma reenvía PATCH");
});

test("concierge: formatos, validación y estados usan el mapeo central",()=>{
 assert.equal(domain.conciergeUsageLabel({conversations:0,maxConversations:500}),"0 / 500","Sin consumo muestra 0, no vacío");
 assert.equal(domain.conciergeUsageLabel({conversations:123,maxConversations:null}),"123 / Ilimitado");
 assert.equal(domain.conciergeLimitSourceLabel({limitSource:"plan",planName:"Impulso",planMaxConversations:1500},"PE"),"Plan Impulso: 1,500","Usa el formato regional del país de la sesión");
 assert.equal(domain.conciergeLimitSourceLabel({limitSource:"custom",planName:"Impulso"}),"Tope propio");
 assert.equal(domain.conciergeLimitSourceLabel({limitSource:"unlimited",planName:null}),"Ilimitado");
 assert.match(domain.conciergeMonthLabel("2026-10","PE"),/octubre/i);

 assert.deepEqual({...domain.conciergeLimitPayload({limitSource:"custom",maxConversations:" 900 "})},{limitSource:"custom",maxConversations:900});
 for(const limitSource of ["plan","unlimited"])assert.deepEqual({...domain.conciergeLimitPayload({limitSource,maxConversations:"900"})},{limitSource,maxConversations:null});
 for(const value of ["","0","-3","2.5","abc"])assert.ok(domain.conciergeLimitError({limitSource:"custom",maxConversations:value}),value);
 assert.equal(domain.conciergeLimitError({limitSource:"plan",maxConversations:""}),"");
 for(const value of ["0","101","","8.5"])assert.ok(domain.conciergeSettingsError({mode:"warn",warnPercent:value}),value);
 assert.deepEqual({...domain.conciergeSettingsPayload({mode:"warn",warnPercent:" 75 "})},{mode:"warn",warnPercent:75});

 assert.deepEqual({...domain.conciergeAlertStatus({alert:"warning",maxConversations:500},"warn")},{label:"Cerca del tope",tone:"orange"});
 assert.deepEqual({...domain.conciergeAlertStatus({alert:"reached",maxConversations:500},"enforce")},{label:"Tope alcanzado",tone:"red"});
 assert.equal(domain.conciergeAlertStatus({alert:"none",maxConversations:500},"measure").label,"Solo medición","En Solo medir no se afirma que esté dentro del tope");
 assert.equal(domain.conciergeAlertStatus({alert:"none",maxConversations:null},"warn").label,"Sin tope");
 assert.deepEqual([...domain.conciergeModeOptions.map(option=>option.label)],["Solo medir","Avisar","Bloquear conversaciones nuevas"]);
 assert.ok(domain.conciergeModeOptions.every(option=>option.description.length>20),"Cada modo tiene su explicación de una línea");

 assert.doesNotMatch(usageUi,/<Status tone="(orange|red|green)"/,"No se recolorean estados por pantalla");
 assert.match(usageUi,/conciergeAlertTone\(item\.alert\)/);
});

test("concierge: datos remotos con skeleton, error explícito y vacío",()=>{
 assert.match(usageUi,/ConciergeSettingsSkeleton/);
 assert.match(usageUi,/<ErrorState title="No pudimos cargar la configuración de Concierge"[^>]*onRetry=/);
 assert.match(usageUi,/Las conversaciones en curso nunca se cortan/);
 assert.match(usageUi,/al llegar al 100 % del tope ya no se abren conversaciones nuevas/);
 assert.match(usageUi,/aria-label="Cargando uso de Concierge"><Skeleton/,"La celda del listado tiene su propio skeleton");
 assert.match(usageUi,/No disponible/,"La celda muestra un error explícito");
 assert.match(usageUi,/<InlineNotice tone="danger"[^>]*title="No pudimos cargar el uso de Fudia Concierge"[^>]*action=\{<Button[^>]*onClick=\{\(\)=>void usage\.refetch\(\)\}/);
 assert.match(usageUi,/ConciergeUsagePanelSkeleton/);
 assert.match(usageUi,/<ErrorState title="No pudimos cargar el uso de Concierge"[^>]*onRetry=/);
 assert.match(usageUi,/title="Aún no hay historial"/);
 assert.match(usageUi,/Sin consumo de tokens este mes\./);
 assert.match(usageUi,/client\.setQueryData\(detailKey\(organizationId\),data\)/);
 assert.match(usageUi,/invalidateQueries\(\{queryKey:\["platform-concierge-usage"\]\}\)/,"Guardar el tope refresca el listado");
});

test("concierge: el listado de empresas muestra uso del mes y aviso de cartera",()=>{
 const page=read(`${platform}presentation/platform-companies-page.tsx`);
 assert.match(page,/"Concierge del mes","Acciones"\]/);
 assert.match(page,/queryKey:\["platform-concierge-usage",organizationIds\]/);
 assert.match(page,/enabled:organizationIds\.length>0/,"Consulta después de cargar la página de empresas");
 assert.equal((page.match(/<ConciergeUsageCell organizationId=\{company\.id\} usage=\{conciergeUsage\}\/>/g)||[]).length,2,"Tabla y tarjetas móviles");
 assert.match(page,/<ConciergeUsageNotice usage=\{conciergeUsage\}\/>/);
 assert.match(usageUi,/summary\.warning\+summary\.reached===0\)return null/);
 const dialog=read(`${platform}presentation/platform-company-detail-dialog.tsx`);
 assert.match(dialog,/<ConciergeUsageDialogSection organizationId=\{company\.id\}\/>/);
 assert.doesNotMatch(usageUi.slice(usageUi.indexOf("export function ConciergeUsageDialogSection"),usageUi.indexOf("function AlertStatus")),/<form|<Button/,"La ficha del modal sigue siendo de solo lectura");
 assert.match(read(`${platform}presentation/platform-company-detail-page.tsx`),/<ConciergeUsagePanel organizationId=\{organizationId\}\/>/);
 assert.match(read(`${platform}presentation/platform-global-settings-page.tsx`),/<ConciergeUsageSettingsPanel\/>/);
 assert.doesNotMatch(usageUi,/\{(item|data|detail)\.organizationId\}/,"Nunca se muestra el UUID de la empresa");
});

test("concierge: sin topes comerciales hardcodeados en el frontend",()=>{
 for(const file of sourceFiles(platform)){
  const source=read(file);
  if(!/Concierge|concierge/.test(source))continue;
  assert.doesNotMatch(source,/(?<!max\(|maxLength=\{)\b(500|1500|4000)\b(?!\s*caracteres)/,`${file} no fija topes de plan`);
 }
 const css=read(`${platform}presentation/platform-concierge-usage.css`);
 assert.match(css,/\.concierge-choice\{[^}]*min-height:var\(--size-44\)/,"Opciones con área táctil de 44 px");
 assert.match(css,/@media\(max-width:600px\)\{[\s\S]*?\.concierge-usage-metrics\{grid-template-columns:1fr\}/);
});
