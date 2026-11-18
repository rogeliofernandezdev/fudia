/** Metadata belongs to the remote inventory unit catalog. */
export type MeasurementUnit={code:string;name:string;family:string;familyLabel:string;referenceFactor:string;inventoryDefault:boolean;recipeDefault:boolean};

export function compatibleMeasurementUnits(units:MeasurementUnit[],baseCode:string){
 const base=units.find(unit=>unit.code===baseCode);
 return units.filter(unit=>unit.code===baseCode||Boolean(base&&["mass","volume"].includes(base.family)&&unit.family===base.family));
}

export function suggestedRecipeUnit(units:MeasurementUnit[],baseCode:string){
 const compatible=compatibleMeasurementUnits(units,baseCode);
 return compatible.find(unit=>unit.recipeDefault)?.code??baseCode;
}

const decimalScale=BigInt("1000000000"),zero=BigInt(0);
function decimal(value:string){
 if(!/^\d+(\.\d{1,9})?$/.test(value.trim()))return null;
 const[whole,fraction=""]=value.trim().split(".");
 return BigInt(whole)*decimalScale+BigInt(fraction.padEnd(9,"0"));
}

/** Exact decimal conversion for display and unit changes; the API validates again. */
export function convertMeasurementQuantity(value:string,from:MeasurementUnit|undefined,to:MeasurementUnit|undefined){
 if(!from||!to)return "";
 if(from.code===to.code)return value;
 if(from.family!==to.family||!["mass","volume"].includes(from.family))return "";
 const amount=decimal(value),source=decimal(from.referenceFactor),target=decimal(to.referenceFactor);
 if(amount===null||source===null||target===null||target===zero)return "";
 const numerator=amount*source;
 if(numerator%target!==zero)return "";
 const converted=numerator/target,whole=converted/decimalScale,fraction=(converted%decimalScale).toString().padStart(9,"0").replace(/0+$/,"");
 return whole.toString()+(fraction?"."+fraction:"");
}
