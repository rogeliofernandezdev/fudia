// Production presentation with illustrative data; no API calls or persisted writes.
import {readFileSync,existsSync} from "node:fs";
import {createRequire} from "node:module";
import {dirname,resolve} from "node:path";
import {fileURLToPath} from "node:url";
import vm from "node:vm";
import ts from "typescript";
import React from "react";
import {renderToStaticMarkup} from "react-dom/server";

const require=createRequire(import.meta.url);
const {QueryClient,QueryClientProvider}=require("@tanstack/react-query");
const root=resolve(dirname(fileURLToPath(import.meta.url)),"../../src");
const h=React.createElement;
export const exampleOrder={id:"preview-order",code:"PED-004",channel:"salon",status:"confirmado",tableId:"preview-table",tableName:"04",waiterId:"preview-waiter",waiterName:"Cielo Fernandez",customerId:"",customerName:"",customerPhone:"",address:"",reference:"",notes:"Atender primero las bebidas.",subtotal:"30",deliveryFee:"0",total:"30",paidAmount:"0",remainingAmount:"30",accountState:"open",createdAt:"2026-10-09T18:53:00-05:00",updatedAt:"2026-10-09T18:53:00-05:00",items:[
 {id:"one",productId:"p-one",name:"Caldo de Gallina Tradicional",qty:"1",unitPrice:"18",note:"Sin cebolla"},
 {id:"two",productId:"p-two",name:"Jarra de Chicha Morada (1.5 L)",qty:"1",unitPrice:"12",note:"Con poco dulce"}
],serviceItems:[
 {id:"s-one",orderItemId:"one",name:"Caldo de Gallina Tradicional",destination:"cocina",destinationLabel:"Cocina",status:"pendiente",statusLabel:"Pendiente"},
 {id:"s-two",orderItemId:"two",name:"Jarra de Chicha Morada (1.5 L)",destination:"barra",destinationLabel:"Barra",status:"pendiente",statusLabel:"Pendiente"}
]};
export function detailComponents({userId="preview-waiter",permissions=["orders.manage"],includePOS=false,includeComanda=false,category="",categoriesOpen=false,comandaOpen=false,catalogState="ready",extraCategories=false}={}){
 const cache=new Map();
 function load(path){
  const file=resolve(root,path);
  if(cache.has(file))return cache.get(file);
  const exports={};cache.set(file,exports);
  const source=ts.transpileModule(readFileSync(file,"utf8"),{compilerOptions:{module:ts.ModuleKind.CommonJS,jsx:ts.JsxEmit.ReactJSX,esModuleInterop:true}}).outputText;
  vm.runInNewContext(source,{exports,require:name=>{
   if(name.endsWith(".css"))return {};
   if(includeComanda&&name==="react"&&file.endsWith("comanda-catalog.tsx")){
    let slot=0;return {...React,useState:initial=>{const index=slot++;return React.useState(index===1?category:index===2?categoriesOpen:initial);}};
   }
   if(includeComanda&&name==="react"&&file.endsWith("comanda-view.tsx")){
    let slot=0;return {...React,useState:initial=>{const index=slot++;return React.useState(index===1?comandaOpen:initial);}};
   }
   if(includeComanda&&name==="@tanstack/react-query"){
    const names=["Entradas","Segundos","Bebidas","Postres",...(extraCategories?Array.from({length:24},(_,index)=>`Categoría de prueba ${index+1} con nombre extenso`):[])];
    const products=["Caldo de Gallina Tradicional","Lomo Saltado","Jarra de Chicha Morada (1.5 L)","Tequeños de Queso","Causa Rellena","Arroz con leche"].map((name,index)=>({id:`p${index}`,name,price:index%2?"25":"18",imageUrl:null,availabilityStatus:index===4?"low":index===5?"sold_out":"available",remaining:index===4?2:null}));
    return {useQuery:()=>({data:{items:names.map((name,index)=>({id:`cat${index}`,name}))},isLoading:catalogState==="loading",isError:catalogState==="category-error",isFetching:false,refetch:()=>{}}),useInfiniteQuery:options=>({data:{pages:[{items:catalogState==="empty"?[]:options.queryKey[1]==="combos"?[{id:"combo1",name:"Menú criollo",price:"25",groupCount:3,imageUrl:null},{id:"combo2",name:"Combo para compartir",price:"48",groupCount:2,imageUrl:null}]:products,total:6}]},isLoading:catalogState==="loading",isError:catalogState==="error",error:new Error("Sin conexión de prueba"),isFetching:false,hasNextPage:false,isFetchingNextPage:false,refetch:()=>{},fetchNextPage:()=>{}})};
   }
   if(name==="next/link")return {__esModule:true,default:({children,...props})=>h("a",props,children)};
   if(name==="@/providers")return {useSession:()=>({user:{id:userId},location:{country:"PE",timezone:"America/Lima"},can:value=>permissions.includes(value)}),useMoney:()=>value=>`S/ ${Number(value).toFixed(2)}`,useFeedback:()=>({notify:()=>{}})};
   if(name==="@/providers/feedback-provider")return {useFeedback:()=>({notify:()=>{}})};
   if(name==="@/design-system")return {...load("design-system/page-header.tsx"),...load("design-system/icons.tsx"),...load("design-system/skeleton.tsx"),...load("design-system/states.tsx"),...load("design-system/modal.tsx"),...(includeComanda?{...load("design-system/dialog.tsx"),...load("design-system/confirm-dialog.tsx"),...load("design-system/form-field.tsx"),...load("design-system/form-controls.tsx"),...load("design-system/search-field.tsx")}:{})};
   if(name==="../infrastructure/service-api")return {closeOrderAccount:()=>{throw new Error("API writes are disabled in visual fixtures");}};
   if(name.startsWith(".")||name.startsWith("@/")){
    const base=name.startsWith("@/")?resolve(root,name.slice(2)):resolve(dirname(file),name);
    const target=[base,base+".tsx",base+".ts"].find(path=>existsSync(path));
    if(!target)throw new Error(`Missing fixture dependency: ${name}`);
    return load(target);
   }
   return require(name);
  }});
  return exports;
 }
 return {...load("modules/operations/orders/domain/order-actions.ts"),...load("modules/operations/orders/domain/service-flow.ts"),...load("modules/operations/orders/presentation/order-detail-dialog.tsx"),...load("design-system/modal.tsx"),...load("modules/operations/orders/presentation/order-meta.ts"),...load("modules/operations/orders/presentation/order-service-controls.tsx"),...load("modules/sales/presentation/sale-detail-dialog.tsx"),...(includePOS?load("modules/operations/pos/presentation/pos-page.tsx"):{}),...(includeComanda?load("modules/operations/orders/presentation/comanda-view.tsx"):{})};
}
export function renderDetail({order=exampleOrder,skeleton=false,error,variant="order",permissions,userId,busy=false,editable=false}={}){
 const {OrderDetailContent,OrderDetailSkeleton}=detailComponents({permissions,userId});
 const noop=()=>{};
 return renderToStaticMarkup(h(QueryClientProvider,{client:new QueryClient()},
  h("div",{className:"modal-backdrop"},h("section",{className:"crud-modal order-detail",role:"dialog","aria-modal":true,"aria-labelledby":"order-detail-title"},
   skeleton?h(OrderDetailSkeleton,{variant,close:noop}):h(OrderDetailContent,{variant,order,error,canManage:permissions?.includes("orders.manage")??true,busy,close:noop,retry:noop,advance:noop,add:noop,cancel:noop,edit:editable?noop:undefined})))));
}
