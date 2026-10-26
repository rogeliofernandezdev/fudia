import type {PurchaseCombination} from "./types";

/** Regla de la unidad base según el API: las contables (botella, lata...) se cuentan por piezas. */
export type PresentationUnitRule={countable:boolean;unitName:string};

export function presentationUnitRule(catalog:PurchaseCombination[],fallbackName:string):PresentationUnitRule{
  return {
    countable:catalog.some(option=>option.unitCountable),
    unitName:catalog.find(option=>option.code==="unit")?.unitName??catalog[0]?.unitName??fallbackName,
  };
}

/** Un paquete de unidades contables contiene al menos una pieza; kilos o litros admiten fracciones. */
export function presentationContentError(rule:PresentationUnitRule|undefined,presentationType:string,unitsPerPresentation:string){
  const factor=Number(unitsPerPresentation);
  if(!rule?.countable||presentationType==="unit"||unitsPerPresentation.trim()===""||!Number.isFinite(factor)||factor<=0||factor>=1)return "";
  return `La presentación debe contener al menos 1 ${rule.unitName.trim().toLowerCase()||"unidad"}. Las fracciones solo se usan con unidades de medida como kilos o litros.`;
}
