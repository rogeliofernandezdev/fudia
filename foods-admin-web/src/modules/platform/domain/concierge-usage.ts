import {formatRegionalCalendarDate,formatRegionalNumber} from "@/shared/i18n/regional-format";
import type {StatusTone} from "./subscription-status";
import type {ConciergeAlert,ConciergeLimitDraft,ConciergeLimitOverride,ConciergeLimitSource,ConciergeTier,ConciergeUsageMode,ConciergeUsageSettingsDraft,ConciergeUsageSettingsInput,ConciergeUsageSummary} from "./types";

/**
 * Única fuente de etiquetas del límite mensual de Fudia Concierge en Plataforma.
 * Son enums del contrato, no catálogos de negocio: los topes siempre llegan del API.
 */
export const conciergeModeOptions:Array<{value:ConciergeUsageMode;label:string;description:string}>=[
 {value:"measure",label:"Solo medir",description:"Cuenta las conversaciones de cada empresa sin avisar ni bloquear."},
 {value:"warn",label:"Avisar",description:"Marca en Empresas y planes a las empresas cerca del tope o que ya lo alcanzaron."},
 {value:"enforce",label:"Bloquear conversaciones nuevas",description:"Avisa igual y, al llegar al 100 %, deja de abrir conversaciones nuevas."},
];

export function conciergeModeLabel(mode:ConciergeUsageMode){return conciergeModeOptions.find(option=>option.value===mode)?.label??mode;}

const alertLabels:Record<ConciergeAlert,string>={none:"Dentro del tope",warning:"Cerca del tope",reached:"Tope alcanzado"};
/** Ámbar = resolución pendiente y rojo = límite cumplido, según la matriz de `Status`. */
const alertTones:Record<ConciergeAlert,StatusTone>={none:"gray",warning:"orange",reached:"red"};
export function conciergeAlertLabel(alert:ConciergeAlert){return alertLabels[alert]??alert;}
export function conciergeAlertTone(alert:ConciergeAlert):StatusTone{return alertTones[alert]??"gray";}

/**
 * Estado legible del mes. En «Solo medir» el API siempre envía `none`, así que no
 * se afirma «Dentro del tope»: se explica que solo se está midiendo.
 */
export function conciergeAlertStatus(usage:Pick<ConciergeUsageSummary,"alert"|"maxConversations">,mode:ConciergeUsageMode):{label:string;tone:StatusTone}{
 if(usage.alert!=="none")return {label:conciergeAlertLabel(usage.alert),tone:conciergeAlertTone(usage.alert)};
 if(usage.maxConversations===null)return {label:"Sin tope",tone:"gray"};
 if(mode==="measure")return {label:"Solo medición",tone:"gray"};
 return {label:conciergeAlertLabel("none"),tone:conciergeAlertTone("none")};
}

export const conciergeLimitOptions:Array<{value:ConciergeLimitSource;label:string}>=[
 {value:"plan",label:"Usar el del plan"},
 {value:"custom",label:"Tope propio"},
 {value:"unlimited",label:"Ilimitado"},
];

const tierLabels:Record<ConciergeTier,string>={lite:"Lite",flash:"Flash"};
export function conciergeTierLabel(tier:ConciergeTier){return tierLabels[tier]??tier;}

/** Enteros con el formato regional del país de la sesión (`formatRegionalNumber`). */
export function formatCount(value:number,country?:string|null){return formatRegionalNumber(value,country,{maximumFractionDigits:0});}

/** «123 / N» o «123 / Ilimitado»; cero se muestra como «0 / N», nunca vacío. */
export function conciergeUsageLabel(usage:Pick<ConciergeUsageSummary,"conversations"|"maxConversations">,country?:string|null){
 return `${formatCount(usage.conversations??0,country)} / ${usage.maxConversations===null?"Ilimitado":formatCount(usage.maxConversations,country)}`;
}

/** Origen del tope efectivo: «Plan Impulso: N», «Tope propio» o «Ilimitado». */
export function conciergeLimitSourceLabel(usage:Pick<ConciergeUsageSummary,"limitSource"|"planName">&{planMaxConversations?:number|null},country?:string|null){
 if(usage.limitSource==="custom")return "Tope propio";
 if(usage.limitSource==="unlimited")return "Ilimitado";
 const plan=usage.planName?`Plan ${usage.planName}`:"Plan";
 return usage.planMaxConversations===undefined?plan:`${plan}: ${usage.planMaxConversations===null?"ilimitado":formatCount(usage.planMaxConversations,country)}`;
}

/** «2026-10» → «octubre de 2026», en calendario UTC para no depender de la zona del navegador. */
export function conciergeMonthLabel(month:string,country?:string|null){
 if(!/^\d{4}-\d{2}$/.test(month))return month;
 const label=formatRegionalCalendarDate(`${month}-01`,country,{month:"long",year:"numeric"});
 return label==="—"?month:label;
}

export function conciergeSettingsError(draft:ConciergeUsageSettingsDraft){
 const value=draft.warnPercent.trim();
 if(!/^\d+$/.test(value)||Number(value)<1||Number(value)>100)return "Usa un porcentaje entero entre 1 y 100.";
 return "";
}

export function conciergeSettingsPayload(draft:ConciergeUsageSettingsDraft):ConciergeUsageSettingsInput{
 return {mode:draft.mode,warnPercent:Number(draft.warnPercent.trim())};
}

export function conciergeLimitError(draft:ConciergeLimitDraft){
 if(draft.limitSource!=="custom")return "";
 const value=draft.maxConversations.trim();
 if(!/^\d+$/.test(value)||Number(value)<1)return "Ingresa un tope entero mayor que cero.";
 return "";
}

/** `maxConversations` solo viaja con un tope propio; plan e ilimitado envían `null`. */
export function conciergeLimitPayload(draft:ConciergeLimitDraft):ConciergeLimitOverride{
 return {limitSource:draft.limitSource,maxConversations:draft.limitSource==="custom"?Number(draft.maxConversations.trim()):null};
}

/** Vacío = ilimitado (`null`); cualquier otro valor viaja como número. */
export function optionalLimit(value:string):number|null{
 return value.trim()?Number(value):null;
}
