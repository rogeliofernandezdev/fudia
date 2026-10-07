import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {createRequire} from "node:module";
import vm from "node:vm";
import ts from "typescript";
import postcss from "postcss";

const require=createRequire(import.meta.url),root=new URL("../src/",import.meta.url);
function compile(file,resolve=require,extra=""){
 const exports={};
 vm.runInNewContext(ts.transpileModule(readFileSync(new URL(file,root),"utf8")+extra,{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX}}).outputText,{exports,require:resolve});
 return exports;
}
const regional=compile("shared/i18n/regional-format.ts");
const actions=compile("modules/operations/orders/domain/order-actions.ts");
const order={id:"order-1",code:"PED-1",tableName:"Mesa 04",customerName:"",channel:"salon",status:"nuevo",createdAt:"2026-10-07T14:30:00Z",total:"25",items:[]};
const country="PE",timeZone="America/Lima";
const expected=value=>new Intl.DateTimeFormat("es-PE",{timeZone,dateStyle:"medium",timeStyle:"short"}).format(new Date(value));
function nodes(tree){const all=[];function visit(node){if(!node||typeof node!=="object")return;if(Array.isArray(node)){node.forEach(visit);return}all.push(node);visit(node.props?.children)}visit(tree);return all}
const page=compile("modules/operations/orders/presentation/orders-manager.tsx",name=>{
 if(name==="next/dynamic")return{default:()=>"ManualOrderDialog"};
 if(name==="next/link")return{default:"Link"};
 if(name==="react")return{...require(name),useState:value=>[value,()=>{}]};
 if(name==="@tanstack/react-query")return{useQueryClient:()=>({invalidateQueries:()=>{}}),useMutation:()=>({}),useQuery:()=>({data:{items:[order],channelCounts:{salon:1},channelOptions:[{value:"salon",label:"Salón"}],statusOptions:[],total:1},isPending:false,isError:false})};
 if(name==="@/design-system")return{Button:"Button",Icon:"Icon",PageHeader:"PageHeader",Pagination:"Pagination",RowActionButton:"RowActionButton",Status:"Status",ConfirmDialog:"ConfirmDialog"};
 if(name==="@/providers/session-context")return{useSession:()=>({can:()=>false,location:{country,timezone:timeZone}})};
 if(name==="@/providers/feedback-provider")return{useFeedback:()=>({notify:()=>{}})};
 if(name==="@/providers/settings-context")return{useSettings:()=>({currencySymbol:"S/"})};
 if(name==="@/shared/hooks/use-debounced-value")return{useDebouncedValue:value=>value};
 if(name==="@/shared/i18n/regional-format")return regional;
 if(name.endsWith("order-actions"))return actions;
 if(name.startsWith("@/")||name.startsWith("."))return{};
 return require(name);
},"\nexport {registeredAt,OrderDetail};\n");

test("Registrado siempre muestra fecha y hora, no depende del tiempo transcurrido",()=>{
 for(const value of ["2026-10-07T14:30:00Z","2020-01-01T05:00:00Z","2099-01-01T04:30:00Z"]){
  const text=page.registeredAt(value,country,timeZone);
  assert.equal(text,expected(value));
  assert.doesNotMatch(text,/Hace|Justo ahora/);
 }
 assert.equal(page.registeredAt("2026-10-07T14:30:00+00",country,timeZone),expected(order.createdAt));
 assert.equal(page.registeredAt("invalid",country,timeZone),"—");
});

test("tabla, variante móvil y detalle muestran la misma fecha del local",()=>{
 const all=nodes(page.OrdersManager()),table=all.find(node=>node.type==="time"&&node.props.className==="order-registered");
 assert.equal(table.props.children,expected(order.createdAt));
 assert.equal(table.props.dateTime,order.createdAt);
 const mobile=all.find(node=>node.props.className==="order-mobile-registered").props.children;
 assert.equal(mobile.props.children,table.props.children);
 const detail=nodes(page.OrderDetail({loading:false,order,currencySymbol:"S/",canManage:false,busy:false,close:()=>{},advance:()=>{},cancel:()=>{}})).find(node=>node.type==="time");
 assert.equal(detail.props.children,table.props.children);
});

test("en móvil Registrado no desaparece ni se recorta cuando se oculta su columna",()=>{
 const css=postcss.parse(readFileSync(new URL("modules/operations/styles/orders.css",root),"utf8"));
 const rules=[];
 css.walkRules(rule=>{if(rule.selector===".orders-table td:first-child .order-mobile-registered")rules.push(rule)});
 assert.equal(rules.length,2);
 const declarations=rule=>Object.fromEntries(rule.nodes.filter(node=>node.type==="decl").map(node=>[node.prop,node.value]));
 assert.equal(declarations(rules[0]).display,"none");
 assert.equal(rules[1].parent.name,"media");
 assert.equal(declarations(rules[1]).display,"block");
 assert.equal(declarations(rules[1])["white-space"],"normal");
 assert.equal(declarations(rules[1])["max-width"],"none");
});
