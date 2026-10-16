import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync,writeFileSync,mkdirSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import {cashReportFixture} from "./helpers/cash-report-fixture.mjs";
import {resolveTokens} from "./helpers/design-tokens.mjs";
const require=createRequire(import.meta.url),root=new URL("../src/",import.meta.url);
function compile(file,resolve=require,extras={}){const exports={};vm.runInNewContext(ts.transpileModule(readFileSync(new URL(file,root),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve,Blob,Uint8Array,Date,...extras});return exports;}
const regional=compile("shared/i18n/regional-format.ts");
const format=compile("modules/operations/cash/domain/report-format.ts",name=>name==="@/shared/i18n/regional-format"?regional:require(name));
function nodes(tree){const list=[];function visit(item){if(!item||typeof item!=="object")return;if(Array.isArray(item)){item.forEach(visit);return;}list.push(item);visit(item.props?.children);}visit(tree);return list;}
const palette=Object.fromEntries(Object.entries({primary:"--primary-600",ink:"--ink-950",muted:"--ink-500",surface:"--surface",soft:"--cloud-50",line:"--line",brand:"--brand-600"}).map(([key,value])=>[key,resolveTokens(`var(${value})`)]));

test("el informe usa moneda y fecha del snapshot, no la configuración actual",()=>{
 const f=format.cashReportFormat(cashReportFixture());assert.equal(f.money("60.00"),"S/ 60.00");assert.match(f.date("2026-10-09T01:00:00Z"),/8\/10\/26.*(20:00|8:00)/);assert.equal(f.money(null),"—");
 const other=format.cashReportFormat({...cashReportFixture(),currencySymbol:"€",currencyPosition:"after",currencyDecimals:3});assert.equal(other.money("2"),"2.000 €");
});
test("el resumen conserva cada producto, cantidades, precio, autor y egreso",()=>{
 const component=compile("modules/operations/cash/presentation/cash-report-content.tsx",name=>name.endsWith("report-format")?format:require(name));const fixture=cashReportFixture(4),all=nodes(component.CashReportContent({report:fixture}));
 const tables=all.filter(item=>typeof item.type==="function"&&item.props.headers);assert.equal(tables[0].props.children.length,4);assert.deepEqual(Array.from(tables[0].props.headers),["Fecha y hora","Pedido / mesa","Producto","Cantidad","Precio unitario","Importe"]);assert.equal(tables[2].props.children.length,1);assert.equal(all.filter(item=>item.type==="details").length,5);
});
test("PDF real incluye logo y continúa las filas en múltiples páginas",async()=>{
 const builder=compile("modules/operations/cash/infrastructure/cash-report-pdf.ts",name=>name.endsWith("report-format")?format:require(name));
 const logo=new Uint8Array(readFileSync(new URL("../public/assets/images/logo.png",import.meta.url)));
 const fixture=cashReportFixture(80);fixture.sales[79].name="PRODUCTO FINAL - Piña y limón";
 const blob=builder.buildCashReportPdf(fixture,logo,palette);assert.equal(blob.type,"application/pdf");const buffer=Buffer.from(await blob.arrayBuffer());assert.equal(buffer.subarray(0,5).toString(),"%PDF-");assert.ok((buffer.toString("latin1").match(/\/Type \/Page\b/g)??[]).length>4);assert.match(buffer.toString("latin1"),/\/Subtype \/Image/);
 if(process.env.CASH_REPORT_QA_DIR){mkdirSync(process.env.CASH_REPORT_QA_DIR,{recursive:true});writeFileSync(`${process.env.CASH_REPORT_QA_DIR}/cash-report-stress.pdf`,buffer);writeFileSync(`${process.env.CASH_REPORT_QA_DIR}/cash-report-sample.pdf`,Buffer.from(await builder.buildCashReportPdf(cashReportFixture(),logo,palette).arrayBuffer()));}
});

test("cerrar exige botón explícito, bloquea doble envío y espera el API",async()=>{
 const schema=compile("modules/operations/cash/domain/cash-schema.ts");const rhf=require("react-hook-form");let form,saveCount=0,done;
 const dialog=compile("modules/operations/cash/presentation/cash-dialogs.tsx",name=>{
  if(name==="react")return{useState:value=>[value,()=>{}],useRef:value=>({current:value})};
  if(name==="react-hook-form")return{useForm:options=>{form=rhf.createFormControl(options);return {...form,formState:{errors:{}}}},useWatch:options=>form.control._getWatch(options.name)};
  if(name.endsWith("cash-schema"))return schema;
  if(name==="@/design-system")return new Proxy({},{get:(_,key)=>String(key)});
  if(name==="@/design-system/dialog")return{Dialog:"Dialog"};
  if(name.endsWith("cash-report-content"))return{CashReportContent:"CashReportContent"};
  if(name.endsWith("shift-attribution"))return{};return require(name);
 });
 const report=cashReportFixture();report.shift.status="open";const all=nodes(dialog.CloseCashShiftDialog({shift:report.shift,currency:"PEN",formatMoney:format.cashReportFormat(report).money,busy:false,close:()=>{},save:async draft=>{saveCount++;assert.equal(draft.counts.length,0);await new Promise(resolve=>{done=resolve;});}}));
 const formNode=all.find(item=>item.type==="form");let prevented=false;formNode.props.onSubmit({preventDefault:()=>{prevented=true;}});assert.ok(prevented);assert.equal(saveCount,0);
 for(const tagName of ["INPUT","TEXTAREA","BUTTON"]){let blocked=false;formNode.props.onKeyDown({key:"Enter",target:{tagName},preventDefault:()=>{blocked=true;}});assert.equal(blocked,tagName==="INPUT",`Enter en ${tagName}`);assert.equal(saveCount,0);}
 let tabBlocked=false;formNode.props.onKeyDown({key:"Tab",target:{tagName:"INPUT"},preventDefault:()=>{tabBlocked=true;}});assert.equal(tabBlocked,false);
 const button=all.find(item=>item.type==="Button"&&item.props.icon==="lock");assert.equal(button.props.type,"button");assert.equal(button.props.children,"Cerrar turno");button.props.onClick();button.props.onClick();await new Promise(resolve=>setImmediate(resolve));assert.equal(saveCount,1);button.props.onClick();assert.equal(saveCount,1);done();await new Promise(resolve=>setImmediate(resolve));
});
test("API del informe y consulta quedan aisladas por empresa, local y turno",()=>{
 const calls=[];const api=compile("modules/operations/cash/infrastructure/cash-api.ts",name=>name==="@/shared/api/client"?{apiFetch:path=>calls.push(path)}:require(name));api.getCashShiftReport("shift-2");assert.equal(calls[0],"cash-shifts/shift-2/report");
 let options;const hook=compile("modules/operations/cash/application/use-cash-report.ts",name=>name==="@/providers"?{useSession:()=>({organization:{id:"company"},location:{id:"local"}})}:name==="@tanstack/react-query"?{useQuery:value=>{options=value;return value;}}:name.endsWith("cash-api")?api:require(name));hook.useCashReport("shift-2",false);assert.deepEqual(Array.from(options.queryKey),["cash-report","company","local","shift-2"]);assert.equal(options.enabled,false);assert.equal(options.refetchOnWindowFocus,false);
});
test("éxito abre la vista previa solo después de cerrar el aviso global",()=>{
 const page=readFileSync(new URL("modules/operations/cash/presentation/cash-page.tsx",root),"utf8");assert.match(page,/onClose:\(\)=>setReportId\(shift.id\)/);assert.match(page,/const closeShiftMutation=useMutation\(\{\s*retry:false/);
 const callbacks=[],state=[];const provider=compile("providers/feedback-provider.tsx",name=>{
  if(name==="react")return{createContext:()=>({Provider:"Provider"}),useCallback:fn=>fn,useEffect:()=>{},useMemo:fn=>fn(),useRef:value=>({current:value}),useState:value=>[value,next=>state.push(next)]};
  if(name==="@/design-system/dialog")return{Dialog:"Dialog",closeDialogsForFeedback:()=>state.push("origin-closed")};if(name==="@/design-system/icons")return{Icon:"Icon"};if(name.endsWith(".css"))return{};return require(name);
 },{window:{clearTimeout(){},setTimeout:fn=>{callbacks.push(fn);return 1;}}});
 let preview=false;const tree=provider.FeedbackProvider({children:[]});tree.props.value.notify({tone:"success",title:"Cerrado",message:"Confirmado",onClose:()=>{preview=true;}});assert.equal(preview,false);assert.equal(state[0],"origin-closed");
 // The exit callback is invoked only after the feedback animation clears.
 assert.match(readFileSync(new URL("providers/feedback-provider.tsx",root),"utf8"),/setItem\(null\);setClosing\(false\);callback\?\.\(\)/);
});
test("vista previa libera el Blob y conserva reintento sin repetir cierre",()=>{
 const source=readFileSync(new URL("modules/operations/cash/presentation/cash-report-dialog.tsx",root),"utf8");assert.match(source,/URL.revokeObjectURL\(url\)/);assert.match(source,/cancelled=true/);assert.match(source,/loadFudiaReportLogo/);assert.ok(!source.includes("closeCashShift"));assert.match(source,/iframe/);assert.match(source,/anchor.download=pdf.filename/);
 const css=readFileSync(new URL("modules/operations/cash/presentation/cash-report.css",root),"utf8");assert.match(css,/@media\(max-width:600px\)/);assert.match(css,/overflow:auto/);assert.match(css,/min-height:var\(--control-height\)/);
 const closingCss=readFileSync(new URL("modules/operations/cash/presentation/cash.css",root),"utf8");assert.match(closingCss,/\.cash-close-modal[^\n]+display:flex;flex-direction:column/);assert.match(closingCss,/\.cash-close-modal \.cash-dialog-body\{min-height:0;overflow:auto/);assert.match(closingCss,/\.cash-close-modal form>footer\{flex-shrink:0/);
 assert.match(closingCss,/\.cash-close-modal\{[^\n]+max-height:calc\(100dvh - var\(--size-48\)\)/);
 assert.match(closingCss,/\.cash-close-modal>form\{display:flex;flex-direction:column;min-height:0/);
 assert.match(closingCss,/scrollbar-gutter:stable;scrollbar-width:thin;scrollbar-color:var\(--ink-400\)/);
 assert.match(closingCss,/grid-auto-rows:max-content;align-content:start/);
 assert.match(closingCss,/@media\(max-width:600px\)\{\s*\.cash-close-modal\{max-height:calc\(100dvh - var\(--size-20\)\)/);
 assert.ok(!closingCss.includes(".cash-modal,.cash-close-modal,.cash-detail-modal{width:100vw;height:100dvh"));
});
