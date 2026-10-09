import {z} from "zod";
import type {FieldError,FieldErrors,Resolver} from "react-hook-form";
import type {PurchaseInventoryItemDraft,PurchaseOrderDraft,PurchaseReceiptDraft,SupplierDraft} from "./types";

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


const purchaseInventoryCommon={
  categoryId:z.string(),
  name:z.string().trim().min(1,"Ingresa el nombre del artículo.").max(160),
  description:z.string().max(1000),
  price:z.string(),
  unit:z.string().trim().min(1,"Selecciona una unidad de inventario."),
  presentationType:z.string().trim().regex(/^[a-z][a-z0-9_-]{0,31}$/,"Selecciona una presentación."),
  unitsPerPresentation:z.string(),
  presentations:z.array(z.object({presentationType:z.string().regex(/^[a-z][a-z0-9_-]{0,31}$/),unitsPerPresentation:z.string(),isDefault:z.boolean()})).min(1,"Agrega una presentación.").max(50).optional(),
  minimumStock:z.string().trim().refine(value=>{
    const parsed=Number(value);
    return value!==""&&Number.isFinite(parsed)&&parsed>=0;
  },"El stock mínimo no puede ser negativo."),
};

export const purchaseInventoryItemSchema=z.discriminatedUnion("mode",[
  z.object({
    mode:z.literal("new_product"),
    ...purchaseInventoryCommon,
    categoryId:z.string().trim().min(1,"Selecciona una categoría."),
    price:z.string().trim().regex(/^[0-9]+([.][0-9]{1,2})?$/,"Ingresa un precio válido."),
  }),
  z.object({
    mode:z.literal("new_ingredient"),
    ...purchaseInventoryCommon,
  }),
]).superRefine((value,ctx)=>{
  const rows=value.presentations??[{presentationType:value.presentationType,unitsPerPresentation:value.unitsPerPresentation,isDefault:true}];
  const seen=new Set<string>();
  for(const row of rows){
    const factor=Number(row.unitsPerPresentation),key=row.presentationType+":"+factor;
    if(row.unitsPerPresentation.trim()===""||!Number.isFinite(factor)||factor<=0||factor>=1e11||Math.abs(factor*1000-Math.round(factor*1000))>0.000001||(row.presentationType==="unit"&&factor!==1)){
      ctx.addIssue({code:"custom",path:[value.presentations?"presentations":"unitsPerPresentation"],message:"Ingresa un contenido mayor que cero, con hasta tres decimales. La compra en la misma unidad de inventario equivale a 1."});
    }else if(seen.has(key)){ctx.addIssue({code:"custom",path:["presentations"],message:"No repitas la misma presentación y conversión."})}
    seen.add(key);
  }
  if(rows.filter(row=>row.isDefault).length!==1)ctx.addIssue({code:"custom",path:["presentations"],message:"Elige una presentación predeterminada."});
});

export const purchaseReceiptSchema=z.object({
  purchaseOrderId:z.string().trim().min(1),
  idempotencyKey:z.string().trim().min(1),
  notes:z.string().max(500,"Las notas no pueden superar 500 caracteres."),
  items:z.array(z.object({
    purchaseOrderItemId:z.string().trim().min(1),
    quantity:z.string().trim().refine(value=>{
      const parsed=Number(value);
      return value!==""&&Number.isFinite(parsed)&&parsed>=0;
    },"Ingresa una cantidad válida."),
  })).min(1),
}).superRefine((value,ctx)=>{
  if(!value.items.some(item=>Number(item.quantity)>0)){
    ctx.addIssue({code:"custom",path:["items"],message:"Registra al menos una cantidad recibida."});
  }
});

export const purchaseInventoryItemResolver=makeResolver<PurchaseInventoryItemDraft>(purchaseInventoryItemSchema);
export const purchaseReceiptResolver=makeResolver<PurchaseReceiptDraft>(purchaseReceiptSchema);
