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
function mountPage(value=shift,tab="registers",permissions=["cash.manage","cash.registers.manage"]){
 let index=0;
 const resolve=name=>{
  if(name==="react")return{...require(name),useState:initial=>[index++===0?tab:initial,()=>{}]};
  if(name==="@tanstack/react-query")return{useQueryClient:()=>({invalidateQueries:()=>{}}),useMutation:()=>({}),useQuery:options=>({data:options.queryKey[0]==="cash-registers"?{items:[{id:"register-1",code:"CAJ",name:"Caja principal",active:true,openShift:value}]}:options.queryKey[0]==="cash-shifts"?{items:[value],total:1}:{},isLoading:false,isError:false})};
  if(name==="@/design-system")return new Proxy({},{get:(_,key)=>String(key)});
  if(name==="@/design-system/dialog")return{Dialog:"Dialog"};
  if(name==="@/providers")return{useSession:()=>({can:permission=>permissions.includes(permission),user:{id:"cash-user"},location:{country,timezone}}),useFeedback:()=>({notify:()=>{}}),useMoney:()=>value=>"S/ "+value};
  if(name==="@/providers/settings-context")return{useSettings:()=>({currencySymbol:"S/",currencyPosition:"before",currencyDecimals:2})};
  if(name==="@/shared/i18n/regional-format")return regional;
  if(name.endsWith("shift-attribution"))return attribution;
  if(name.endsWith("shift-ownership"))return compile("modules/operations/cash/domain/shift-ownership.ts");
  if(name.startsWith("@/")||name.startsWith(".")||name==="react-hook-form")return{};
  return require(name);
 };
 const page=compile("modules/operations/cash/presentation/cash-page.tsx",resolve,"\nexport {CashRegisters};\n");
 const tree=page.CashPage();
 const registers=nodes(tree).find(node=>node.type===page.CashRegisters);
 return{all:nodes(registers?page.CashRegisters(registers.props):tree),date:registers?.props.dateTime,resolve,renderRegisters:changes=>page.CashRegisters({...registers.props,...changes})};
}

test("un cajero abre y cierra su turno sin acciones para crear, editar o desactivar cajas",()=>{
 const cashier=mountPage({...shift,openedById:"cash-user"},"registers",["cash.manage"]);
 assert.equal(cashier.all.some(node=>node.type==="RowActionButton"&&["edit","activate","deactivate"].includes(node.props.action)),false);
 assert.ok(cashier.all.some(node=>node.type==="Button"&&node.props.children==="Cerrar turno"));
 assert.equal(cashier.renderRegisters({items:[]}).props.action,undefined);
 const idle=nodes(cashier.renderRegisters({items:[{id:"register",name:"Caja",active:true,openShift:null}]}));
 assert.ok(idle.some(node=>node.type==="Button"&&node.props.children==="Iniciar turno"));
 const administrator=mountPage();
 assert.ok(administrator.all.some(node=>node.type==="RowActionButton"&&node.props.action==="edit"));
 assert.ok(administrator.renderRegisters({items:[]}).props.action);
});

test("una caja ocupada por otro cajero no muestra importes, movimientos ni acciones",()=>{
 const cashier=mountPage();
 const foreign={...shift,id:"foreign",detailsVisible:false,expectedVisible:false,openedById:"other",openingAmount:"",incomeAmount:"",expenseAmount:"",expectedAmount:"",movementCount:0};
 const tree=cashier.renderRegisters({items:[{id:"register",name:"Caja ocupada",active:true,openShift:foreign}],canManage:true,canManageRegisters:false});
 const all=nodes(tree);
 assert.equal(all.some(node=>node.type==="RowActionButton"&&node.props.action==="view"),false);
 assert.equal(all.some(node=>node.type==="IconButton"),false);
 assert.equal(all.some(node=>node.type==="Button"),false);
 assert.equal(all.some(node=>node.props?.className==="cash-register-balance"),false);
 assert.equal(all.some(node=>node.props?.className==="cash-register-flow"),false);
 assert.ok(all.some(node=>node.type==="span"&&node.props.children==="Este turno está a cargo de otro cajero."));
 assert.ok(all.some(node=>node.type==="small"&&node.props.children==="Efectivo de tu turno"));
 assert.equal(all.some(node=>node.type==="strong"&&node.props.children==="S/ 0"),false);
});

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

function historyCells(changes){
 return mountPage({...shift,...changes},"history").all.filter(node=>node.type==="td");
}
function cellText(cell){
 const text=node=>node==null||typeof node==="boolean"?"":Array.isArray(node)?node.map(text).join(""):typeof node==="object"?text(node.props?.children):String(node);
 return text(cell);
}

test("el historial explica el faltante con el esperado y contado guardados al cerrar",()=>{
 const cells=historyCells({status:"closed",expectedAmount:"999",closingExpectedAmount:"65.00",closingCountedAmount:"59.97",varianceAmount:"-5.03"});
 assert.equal(cells.length,10);
 assert.equal(cellText(cells[5]),"S/ 65");
 assert.equal(cellText(cells[6]),"S/ 59.97");
 assert.equal(cellText(cells[7]),"FaltanteS/ 5.03");
 assert.equal(nodes(cells[7]).find(node=>node.type==="Status").props.tone,"red");
});

test("el resultado distingue sobrante y cuadrado, incluso con contado cero",()=>{
 for(const [expected,counted,variance,label,tone] of [["60","65","5","SobranteS/ 5","blue"],["0","0","0","Cuadrado","green"]]){
  const cells=historyCells({status:"closed",closingExpectedAmount:expected,closingCountedAmount:counted,varianceAmount:variance});
  assert.equal(cellText(cells[7]),label);
  assert.equal(nodes(cells[7]).find(node=>node.type==="Status").props.tone,tone);
  assert.equal(cellText(cells[6]),"S/ "+Number(counted));
 }
});

test("un turno abierto muestra esperado actual y resultado pendiente, respetando cierre ciego",()=>{
 const open={expectedAmount:"150",closingCountedAmount:"100",varianceAmount:"-50"};
 const cells=historyCells(open);
 assert.equal(cellText(cells[5]),"S/ 150");
 assert.equal(cellText(cells[6]),"—");
 assert.equal(cellText(cells[7]),"Pendiente");
 const blind=historyCells({...open,expectedVisible:false});
 assert.equal(cellText(blind[5]),"Cierre ciego");
 assert.equal(cellText(blind[6]),"—");
 assert.equal(cellText(blind[7]),"Pendiente");
});

test("un cierre sin datos no inventa un importe cero ni un resultado cuadrado",()=>{
 for(const missing of [null,undefined,"","invalid"]){
  const cells=historyCells({status:"closed",expectedAmount:"65",closingExpectedAmount:missing,closingCountedAmount:missing,varianceAmount:missing});
  for(const index of [5,6,7])assert.equal(cellText(cells[index]),"Sin registro");
 }
});

test("el detalle identifica claramente Apertura y Cierre junto a sus autores",()=>{
 const {resolve,date}=mountPage();
 const {CashShiftDetailDialog}=compile("modules/operations/cash/presentation/cash-dialogs.tsx",resolve);
 for(const closedAt of [null,"2026-10-03T04:45:15.654321Z"]){
  const all=nodes(CashShiftDetailDialog({shift:{...shift,status:closedAt?"closed":"open",closedAt,closedByName:closedAt?"Edith":""},formatMoney:String,formatDateTime:date,formatBusinessDate:value=>value,close:()=>{}}));
  const facts=all.find(node=>node.type==="DetailFacts").props.items;
  const opening=facts.find(item=>item.label==="Apertura");
  assert.equal(opening.value.props.children,expected(shift.openedAt));
  assert.equal(facts.some(item=>item.label==="Cerrado por"),Boolean(closedAt));
  assert.equal(facts.some(item=>item.value==="En curso"),false,"un turno abierto no inventa responsable de cierre");
  assert.equal(all.some(node=>node.type==="ModalHeader"&&node.props.title===shift.code),false,"el detalle no titula con el código técnico");
  if(closedAt){
   const closing=facts.find(item=>item.label==="Cerrado por").value.props.children[1];
   assert.equal(closing.props.children[0],"Cierre: ");
   assert.equal(closing.props.children[1].props.children,expected(closedAt));
  }
 }
});
