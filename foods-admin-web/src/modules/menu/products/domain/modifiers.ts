/** Opciones que se eligen al tomar el pedido (término, acompañamiento, adicionales). */
export type ModifierOptionDraft={key:string;name:string;surcharge:string};
export type ModifierGroupDraft={key:string;name:string;required:boolean;maxSelections:number;options:ModifierOptionDraft[]};
export type ModifierConfig={productId:string;groups:{id:string;name:string;required:boolean;minSelections:number;maxSelections:number;options:{id:string;name:string;surcharge:string}[]}[]};
/** Errores por grupo y por opción; la clave «form» resume el primero para el aviso general. */
export type ModifierErrors=Record<string,string>;

let sequence=0;
export const draftKey=()=>`draft-${Date.now().toString(36)}-${(sequence++).toString(36)}`;
export const emptyOption=():ModifierOptionDraft=>({key:draftKey(),name:"",surcharge:""});
export const emptyGroup=():ModifierGroupDraft=>({key:draftKey(),name:"",required:true,maxSelections:1,options:[emptyOption(),emptyOption()]});

export function groupsFromConfig(config:ModifierConfig):ModifierGroupDraft[]{
 return config.groups.map(group=>({
  key:group.id,name:group.name,required:group.required,maxSelections:group.maxSelections,
  options:group.options.map(option=>({key:option.id,name:option.name,surcharge:Number(option.surcharge)>0?option.surcharge:""})),
 }));
}

/** Cuerpo del PUT: un grupo obligatorio exige al menos una opción y el máximo nunca supera las opciones. */
export type ModifiersPayload={groups:{name:string;required:boolean;minSelections:number;maxSelections:number;options:{name:string;surcharge:string}[]}[]};
export function modifiersPayload(groups:ModifierGroupDraft[]):ModifiersPayload{
 return {groups:groups.map(group=>{
  const options=group.options.filter(option=>option.name.trim());
  const maxSelections=Math.max(1,Math.min(group.maxSelections,options.length));
  return {name:group.name.trim(),required:group.required,minSelections:group.required?1:0,maxSelections,
   options:options.map(option=>({name:option.name.trim(),surcharge:option.surcharge.trim()||"0"}))};
 })};
}

/** Las mismas reglas que valida el API (ValidateModifierConfig), con el mensaje junto al campo. */
export function validateModifierGroups(groups:ModifierGroupDraft[]):ModifierErrors{
 const errors:ModifierErrors={};
 const groupNames=new Set<string>();
 for(const group of groups){
  const name=group.name.trim();
  if(!name)errors[`${group.key}:name`]="Escribe el nombre del grupo";
  else if(name.length>80)errors[`${group.key}:name`]="Usa hasta 80 caracteres";
  else if(groupNames.has(name.toLocaleLowerCase("es")))errors[`${group.key}:name`]="Ya hay un grupo con este nombre";
  groupNames.add(name.toLocaleLowerCase("es"));
  const filled=group.options.filter(option=>option.name.trim());
  if(!filled.length)errors[`${group.key}:options`]="Agrega al menos una opción";
  const optionNames=new Set<string>();
  for(const option of group.options){
   const optionName=option.name.trim();
   if(!optionName){if(option.surcharge.trim())errors[`${option.key}:name`]="Escribe la opción";continue}
   if(optionName.length>80)errors[`${option.key}:name`]="Usa hasta 80 caracteres";
   else if(optionNames.has(optionName.toLocaleLowerCase("es")))errors[`${option.key}:name`]="Opción repetida";
   optionNames.add(optionName.toLocaleLowerCase("es"));
   const surcharge=option.surcharge.trim();
   if(surcharge&&!(/^\d+(\.\d{1,2})?$/.test(surcharge)))errors[`${option.key}:surcharge`]="Importe no válido";
  }
 }
 const first=Object.values(errors)[0];
 if(first)errors.form="Revisa las opciones del pedido marcadas en rojo.";
 return errors;
}
