import {z} from "zod";
import type {FieldError,FieldErrors,Resolver} from "react-hook-form";
import type {PurchaseOrderDraft,SupplierDraft} from "./types";

const lineSchema=z.object({
  inventoryItemId:z.string().trim().min(1,"Selecciona un artículo."),
  presentationId:z.string().trim().min(1,"Selecciona una presentación."),
  quantity:z.string().trim().refine(value=>{
    const parsed=Number(value);
    return value!==""&&Number.isFinite(parsed)&&parsed>0;
  },"Ingresa una cantidad mayor que cero."),
  unitCost:z.string().trim().regex(/^[0-9]+([.][0-9]{1,4})?$/,"Ingresa un costo válido."),
});

export const purchaseOrderSchema=z.object({
  id:z.string(),
  supplierId:z.string().trim().min(1,"Selecciona un proveedor."),
  expectedAt:z.string(),
  notes:z.string().max(500,"Las notas no pueden superar 500 caracteres."),
  items:z.array(lineSchema).min(1,"Agrega al menos un artículo.").max(100),
}).superRefine((value,ctx)=>{
  const seen=new Set<string>();
  value.items.forEach((item,index)=>{
    const key=`${item.inventoryItemId}:${item.presentationId}`;
    if(seen.has(key)){
      ctx.addIssue({code:"custom",path:["items",index,"inventoryItemId"],message:"Esta presentación ya está en la orden."});
    }
    seen.add(key);
  });
});

export const supplierSchema=z.object({
  id:z.string(),
  taxId:z.string().trim().max(11,"El RUC no puede superar 11 caracteres."),
  name:z.string().trim().min(1,"Ingresa el nombre del proveedor.").max(180),
  email:z.string().trim().max(180).refine(value=>value===""||/^[^@\s]+@[^@\s]+\.[^@\s]+$/.test(value),"Ingresa un correo válido."),
  phone:z.string().trim().max(40),
});

function makeResolver<T extends Record<string,unknown>>(schema:z.ZodType<T>):Resolver<T>{
  return values=>{
    const result=schema.safeParse(values);
    if(result.success)return {values:result.data,errors:{}};
    const errors:Record<string,FieldError|Record<string,unknown>>={};
    for(const issue of result.error.issues){
      const [first,second,third]=issue.path;
      if(first==="items"&&typeof second==="number"&&typeof third==="string"){
        const rows=(errors.items??{}) as Record<string,Record<string,FieldError>>;
        rows[String(second)]={...(rows[String(second)]??{}),[third]:{type:issue.code,message:issue.message}};
        errors.items=rows;
        continue;
      }
      if(typeof first==="string"&&!errors[first])errors[first]={type:issue.code,message:issue.message};
    }
    return {values:{},errors:errors as FieldErrors<T>};
  };
}

export const purchaseOrderResolver=makeResolver<PurchaseOrderDraft>(purchaseOrderSchema);
export const supplierResolver=makeResolver<SupplierDraft>(supplierSchema);
