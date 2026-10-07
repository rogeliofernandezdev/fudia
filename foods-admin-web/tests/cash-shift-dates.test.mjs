import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";

const require=createRequire(import.meta.url),root=new URL("../src/",import.meta.url);
function compile(file,resolve=require,extra=""){
 const exports={};
 vm.runInNewContext(ts.transpileModule(readFileSync(new URL(file,root),"utf8")+extra,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
const regional=compile("shared/i18n/regional-format.ts"),attribution=compile("modules/operations/cash/domain/shift-attribution.ts");
const shift={id:"shift-1",code:"CAJ-001",cashRegisterId:"register-1",cashRegisterName:"Caja principal",status:"open",openedByName:"Ana",closedByName:"",activeUserNames:["Ana"],businessDate:"2026-10-02",openedAt:"2026-10-03T02:15:30.123456Z",closedAt:null,expectedVisible:true,openingAmount:"0",incomeAmount:"0",expenseAmount:"0",expectedAmount:"0",movementCount:0};
const country="PE",timezone="America/Lima";
const expected=value=>new Intl.DateTimeFormat("es-PE",{timeZone:timezone,dateStyle:"medium",timeStyle:"short"}).format(new Date(value));
function nodes(tree){const result=[];function visit(node){if(!node||typeof node!=="object")return;if(Array.isArray(node)){node.forEach(visit);return}result.push(node);visit(node.props?.children)}visit(tree);return result}
function mountPage(value=shift,tab="registers"){
 let index=0;
 const resolve=name=>{
  if(name==="react")return{...require(name),useState:initial=>[index++===0?tab:initial,()=>{}]};
  if(name==="@tanstack/react-query")return{useQueryClient:()=>({invalidateQueries:()=>{}}),useMutation:()=>({}),useQuery:options=>({data:options.queryKey[0]==="cash-registers"?{items:[{id:"register-1",code:"CAJ",name:"Caja principal",active:true,openShift:value}]}:options.queryKey[0]==="cash-shifts"?{items:[value],total:1}:{},isLoading:false,isError:false})};
  if(name==="@/design-system")return new Proxy({},{get:(_,key)=>String(key)});
  if(name==="@/design-system/dialog")return{Dialog:"Dialog"};
  if(name==="@/providers")return{useSession:()=>({can:()=>true,location:{country,timezone}}),useFeedback:()=>({notify:()=>{}})};
  if(name==="@/providers/settings-context")return{useSettings:()=>({currencySymbol:"S/",currencyPosition:"before",currencyDecimals:2})};
  if(name==="@/shared/i18n/regional-format")return regional;
  if(name.endsWith("shift-attribution"))return attribution;
  if(name.startsWith("@/")||name.startsWith(".")||name==="react-hook-form")return{};
  return require(name);
 };
 const page=compile("modules/operations/cash/presentation/cash-page.tsx",resolve,"\nexport {CashRegisters};\n");
 const tree=page.CashPage();
 const registers=nodes(tree).find(node=>node.type===page.CashRegisters);
 return{all:nodes(registers?page.CashRegisters(registers.props):tree),date:registers?.props.dateTime,resolve};
}

test("la tarjeta de caja muestra fecha y hora de apertura del local, no el día operativo",()=>{
 const {all}=mountPage();
 const time=all.find(node=>node.type==="time");
 assert.equal(time.props.dateTime,shift.openedAt);
 assert.equal(time.props.children.join(""),"Apertura: "+expected(shift.openedAt));
 assert.notEqual(time.props.children.at(-1),"—");
 assert.match(time.props.children.at(-1),/21:15|9:15/);
 assert.ok(all.some(node=>node.type==="em"&&node.props.children.join("").includes("2 oct. 2026")));
});

test("Caja admite timestamps RFC3339 y offsets históricos conservando el mismo instante",()=>{
 const {date}=mountPage();
 for(const value of ["2026-10-03T02:15:30.123456Z","2026-10-03T02:15:30.123456+00","2026-10-02T21:15:30.123456-05","2026-10-02T21:15:30.123456-05:00","2026-10-03T08:00:30.123456+05:45"]){
  assert.equal(date(value),expected(shift.openedAt));
 }
 assert.equal(date("invalid"),"—");
});

test("el historial conserva apertura y cierre y no inventa cierre en un turno abierto",()=>{
 const open=mountPage(shift,"history").all.filter(node=>node.type==="time");
 assert.equal(open.length,1);
 const closed={...shift,status:"closed",closedByName:"Edith",closedAt:"2026-10-03T04:45:15.654321Z"};
 const times=mountPage(closed,"history").all.filter(node=>node.type==="time");
 assert.equal(times.length,2);
 assert.equal(times[0].props.children,expected(shift.openedAt));
 assert.equal(times[1].props.children,expected(closed.closedAt));
});

test("el detalle identifica claramente Apertura y Cierre junto a sus autores",()=>{
 const {resolve,date}=mountPage();
 const {CashShiftDetailDialog}=compile("modules/operations/cash/presentation/cash-dialogs.tsx",resolve);
 for(const closedAt of [null,"2026-10-03T04:45:15.654321Z"]){
  const all=nodes(CashShiftDetailDialog({shift:{...shift,status:closedAt?"closed":"open",closedAt,closedByName:closedAt?"Edith":""},formatMoney:String,formatDateTime:date,formatBusinessDate:value=>value,close:()=>{}}));
  const spans=all.filter(node=>node.type==="span");
  assert.ok(spans.some(node=>node.props.children?.[0]==="Apertura: "));
  assert.equal(spans.some(node=>node.props.children?.[0]==="Cierre: "),Boolean(closedAt));
  const times=all.filter(node=>node.type==="time");
  assert.equal(times[0].props.children,expected(shift.openedAt));
  if(closedAt)assert.equal(times[1].props.children,expected(closedAt));
 }
});
