import type {Draft,Product} from "./types";

export type WizardIssue={step:number;field:string;message:string};
const amount=(value:string)=>/^\d+(\.\d{1,2})?$/.test(value);

export function validateComboStep(draft:Draft,products:Product[],step:number):WizardIssue|null{
  const issue=(field:string,message:string):WizardIssue=>({step,field,message});
  if(step===1){
    if(!draft.name.trim())return issue("combo-name","Ingresa el nombre del menú o combo.");
    if(!amount(draft.price))return issue("combo-price","Ingresa un precio válido, con hasta dos decimales.");
  }
  if(step===2){
    if(!draft.groups.length)return issue("combo-add-group","Agrega al menos una parte al menú.");
    for(const [index,group] of draft.groups.entries()){
      const prefix=`combo-group-${index}`;
      if(!group.name.trim())return issue(`${prefix}-name`,"Ingresa el nombre de esta parte del menú.");
      if(group.required&&group.options.length<Math.max(1,group.minSelections))return issue(`${prefix}-product`,`Agrega las opciones necesarias para ${group.name}.`);
      if(!Number.isInteger(group.minSelections)||!Number.isInteger(group.maxSelections)||group.minSelections<0||group.maxSelections<Math.max(1,group.minSelections))return issue(`${prefix}-name`,`Revisa los límites de elección de ${group.name}.`);
      const selected=new Set<string>();
      for(const [position,option] of group.options.entries()){
        const product=products.find(item=>item.id===option.productId&&item.active);
        if(!product||selected.has(option.productId))return issue(`${prefix}-product`,`Revisa los productos de ${group.name}: deben estar activos y no repetirse.`);
        selected.add(option.productId);
        if(option.quota&&(!/^\d+$/.test(option.quota)||(product.defaultDailyQuota!=null&&Number(option.quota)>product.defaultDailyQuota)))return issue(`${prefix}-quota-${position}`,`Revisa la cantidad reservada de ${product.name}. No puede superar su cupo disponible.`);
        if(option.surcharge.trim()&&!amount(option.surcharge))return issue(`${prefix}-surcharge-${position}`,`Ingresa un recargo válido para ${product.name}.`);
      }
    }
  }
  if(step===3){
    if(draft.availableFrom&&!Number.isFinite(Date.parse(draft.availableFrom)))return issue("combo-from","Selecciona una fecha de inicio válida.");
    if(draft.availableUntil&&!Number.isFinite(Date.parse(draft.availableUntil)))return issue("combo-until","Selecciona una fecha de fin válida.");
    if(draft.availableFrom&&draft.availableUntil&&Date.parse(draft.availableUntil)<=Date.parse(draft.availableFrom))return issue("combo-until","La fecha de fin debe ser posterior a la fecha de inicio.");
  }
  return null;
}

export function validateComboDraft(draft:Draft,products:Product[]):WizardIssue|null{
  for(const step of [1,2,3]){const issue=validateComboStep(draft,products,step);if(issue)return issue;}
  return null;
}
