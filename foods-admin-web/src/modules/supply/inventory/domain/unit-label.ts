/**
 * Cantidad con su unidad en español: «1 kilogramo», «5 kilogramos», «2 unidades».
 * El nombre llega del catálogo del API; solo se ajusta el plural para leerlo.
 */
export function unitQuantityLabel(quantity:string,unitName:string){
  const unit=unitName.trim().toLocaleLowerCase("es");
  const amount=Number(quantity);
  if(!unit||amount===1)return `${quantity} ${unit}`.trim();
  return `${quantity} ${pluralUnit(unit)}`;
}

/** Nombre de la unidad en plural y minúsculas: «botellas», «unidades». */
export function pluralUnit(unitName:string){
  const unit=unitName.trim().toLocaleLowerCase("es");
  if(/[aeiouáéó]$/.test(unit))return unit+"s";
  if(unit.endsWith("z"))return unit.slice(0,-1)+"ces";
  return unit+"es";
}
