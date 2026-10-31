import type {CatalogProduct,ComboDetail} from "../../salon/domain/catalog-types";
import type {LineDraft} from "../../salon/domain/types";

export function productUsage(lines:LineDraft[]){
 const used:Record<string,number>={};
 for(const line of lines){
  if(line.itemType==="product")used[line.productId]=(used[line.productId]??0)+line.qty;
  else for(const option of line.selections)used[option.productId]=(used[option.productId]??0)+line.qty;
 }
 return used;
}
const quotaKey=(combo:string,group:string,product:string)=>JSON.stringify([combo,group.trim().toLocaleLowerCase(),product]);
function quotaUsage(lines:LineDraft[]){
 const used:Record<string,number>={};
 for(const line of lines)if(line.itemType==="combo")for(const option of line.selections){const key=quotaKey(line.productId,option.groupName,option.productId);used[key]=(used[key]??0)+line.qty;}
 return used;
}
export type LineAvailability={max:number|undefined;blocked:boolean;pending:boolean;name:string};

/** Both dishes and combo components consume the same physical stock. Quotas are per menu/part. */
export function lineAvailability(line:LineDraft,lines:LineDraft[],initial:LineDraft[],products:Record<string,CatalogProduct>,combos:Record<string,ComboDetail>):LineAvailability{
 const used=productUsage(lines),reserved=productUsage(initial.filter(item=>item.sourceItemId));
 const quotas=quotaUsage(lines),reservedQuotas=quotaUsage(initial.filter(item=>item.sourceItemId));
 let max:number|undefined,blocked=false,pending=false,name=line.name;
 const constrain=(limit:number,label:string)=>{const value=Math.max(0,Math.floor(limit+0.000001));if(max===undefined||value<max){max=value;name=label;}};
 const components=line.itemType==="product"?[{productId:line.productId,name:line.name,groupName:""}]:line.selections;
 const quantities:Record<string,number>={};for(const item of components)quantities[item.productId]=(quantities[item.productId]??0)+1;
 const detail=combos[line.productId];
 if(line.itemType==="combo"&&!detail){pending=true;constrain(line.qty,line.name);}
 for(const component of components){
  const option=line.itemType==="combo"?detail?.groups.find(group=>group.name.trim().toLocaleLowerCase()===component.groupName?.trim().toLocaleLowerCase())?.options.find(option=>option.productId===component.productId):undefined;
  const product=products[component.productId];
  const remaining=product?product.remaining:option?.remaining;
  const units=quantities[component.productId];
  if(!product&&remaining===undefined){pending=true;constrain(line.qty,component.name);}
  if(product&&["sold_out","unavailable"].includes(product.availabilityStatus??"")){blocked=true;constrain(((reserved[component.productId]??0)-(used[component.productId]??0)+line.qty*units)/units,component.name);}
  if(typeof remaining==="number")constrain((remaining+(reserved[component.productId]??0)-(used[component.productId]??0)+line.qty*units)/units,component.name);
  if(line.itemType==="combo"){
   if(typeof option?.remaining==="number")constrain((option.remaining+(reserved[component.productId]??0)-(used[component.productId]??0)+line.qty*units)/units,component.name);
   if(!option){pending=true;constrain(line.qty,component.name);continue;}
   if(!option.available){blocked=true;if((option.remaining==null||option.remaining>=1)&&(option.quotaRemaining==null||option.quotaRemaining>=1))constrain(((reserved[component.productId]??0)-(used[component.productId]??0)+line.qty*units)/units,component.name);}
   if(typeof option.quotaRemaining==="number"){
    const key=quotaKey(line.productId,component.groupName??"",component.productId);
    constrain(option.quotaRemaining+(reservedQuotas[key]??0)-(quotas[key]??0)+line.qty,component.name);
   }
  }
 }
 return{max,blocked,pending,name};
}
