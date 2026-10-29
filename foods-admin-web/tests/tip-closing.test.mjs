import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
const require=createRequire(import.meta.url),root=new URL("../src/",import.meta.url);
function compile(file,resolve=require){const exports={};vm.runInNewContext(ts.transpileModule(readFileSync(new URL(file,root),"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText,{exports,require:resolve,crypto:{randomUUID:()=>"delivery-at-close"}});return exports;}
function nodes(tree){if(!tree||typeof tree!=="object")return[];if(Array.isArray(tree))return tree.flatMap(nodes);return[tree,...nodes(tree.props?.children)];}
const {TipClosingSection}=compile("modules/operations/tips/presentation/tip-closing-section.tsx",name=>{
 if(name==="@/design-system")return new Proxy({},{get:(_,key)=>String(key)});
 if(name==="react-select")return "LookupSelect";
 if(name.includes("search-select-styles"))return{createLookupSelectStyles:()=>({})};
 if(name.endsWith(".css"))return{};return require(name);
});
const workspace={available:"25.00",recipients:[{id:"waiter",name:"Mozo del local"}]};
const props={workspace,busy:false,formatMoney:value=>`S/ ${value.toFixed(2)}`,onChange:()=>{}};

test("el cierre solo muestra propinas si hay saldo y nunca preconfirma una entrega",()=>{
 assert.equal(TipClosingSection({...props,workspace:{...workspace,available:"0.00"}}),null);
 const all=nodes(TipClosingSection(props));
 assert.equal(all.find(item=>item.type==="input").props.checked,false);
 assert.equal(all.some(item=>item.type==="LookupSelect"),false);
 assert.equal(all.some(item=>item.type==="Button"),false,"la sección usa el mismo botón de cierre");
});

test("aceptar y desmarcar la entrega conserva una única clave por intento",()=>{
 let draft;
 const checkbox=nodes(TipClosingSection({...props,onChange:value=>{draft=value;}})).find(item=>item.type==="input");
 checkbox.props.onChange({target:{checked:true}});
 assert.equal(draft.amount,"25.00");assert.equal(draft.recipientId,"");assert.equal(draft.requestKey,"delivery-at-close");
 const all=nodes(TipClosingSection({...props,value:draft,onChange:value=>{draft=value;}}));
 all.find(item=>item.type==="LookupSelect").props.onChange({value:"waiter"});
 assert.equal(draft.recipientId,"waiter");assert.equal(draft.requestKey,"delivery-at-close");
 checkbox.props.onChange({target:{checked:false}});assert.equal(draft,undefined);
});

test("se valida destinatario e importe solo cuando se registra una entrega",()=>{
 const {closeCashShiftResolver}=compile("modules/operations/cash/domain/cash-schema.ts");
 const base={countedAmount:"150",counts:[],note:""};
 const payout={recipientId:"waiter",amount:"5",method:"cash",requestKey:"closing-delivery"};
 assert.equal(Object.keys(closeCashShiftResolver(base,{},{}).errors).length,0);
 assert.equal(Object.keys(closeCashShiftResolver({...base,tipPayout:payout},{},{}).errors).length,0);
 for(const patch of [{recipientId:""},{amount:"0"},{amount:"-1"},{amount:"NaN"},{method:"card"}])assert.ok(closeCashShiftResolver({...base,tipPayout:{...payout,...patch}},{},{}).errors.tipPayout);
});

test("cierre y entrega se envían en una sola petición; sin casilla no hay entrega",()=>{
 const calls=[];
 const {closeCashShift}=compile("modules/operations/cash/infrastructure/cash-api.ts",name=>name==="@/shared/api/client"?{apiFetch:(path,init)=>calls.push({path,body:JSON.parse(init.body)})}:require(name));
 const base={countedAmount:"150",counts:[],note:""};
 closeCashShift("shift",base);assert.equal("tipPayout" in calls[0].body,false);
 closeCashShift("shift",{...base,tipPayout:{recipientId:"waiter",amount:"5",method:"transfer",requestKey:"closing-delivery"}});
 assert.equal(calls.length,2);assert.equal(calls[1].path,"cash-shifts/shift/close");assert.equal(calls[1].body.tipPayout.amount,5);assert.equal(calls[1].body.countedAmount,150);
});
