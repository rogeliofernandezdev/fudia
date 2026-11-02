"use client";
import {InlineNotice} from "@/design-system";
import {formatRegionalCalendarDate,formatRegionalNumber} from "@/shared/i18n/regional-format";
import {fullDate,lastDay,ratePercentLabel,specialTaxStatusOn,type SpecialTaxRegime,type SpecialTaxStatus} from "@/shared/fiscal/special-tax";

// The company only declares it is enrolled (SUNAT padrón); platform
// administrators keep the dated rates, so the form shows today's rate and warns
// when the registered rates are about to run out.
export const percentOf=(rate:string,country:string)=>ratePercentLabel(rate,v=>formatRegionalNumber(v,country,{maximumFractionDigits:4}));
export function specialTaxNotice(status:SpecialTaxStatus,general:string,country:string){
 const day=(iso:string)=>formatRegionalCalendarDate(iso,country,fullDate);
 if(status.state==="expired")return {tone:"warning" as const,title:"El IGV reducido no tiene tasa para hoy",text:`Se factura con ${general} hasta que la plataforma registre el periodo vigente.`};
 if(status.state==="expiring"&&status.coverageEnd)return {tone:"warning" as const,title:`El IGV reducido vence el ${day(lastDay(status.coverageEnd))}`,text:`Desde el ${day(status.coverageEnd)} se facturará con ${general} si no se registra el periodo siguiente.`};
 return null;
}
function specialTaxToday(status:SpecialTaxStatus,country:string){
 if(!status.rate)return "Sin tasa para hoy";
 return status.rateValidTo?`Hoy factura con ${percentOf(status.rate,country)} hasta el ${formatRegionalCalendarDate(lastDay(status.rateValidTo),country,fullDate)}`:`Hoy factura con ${percentOf(status.rate,country)}`;
}
export function SpecialRegimeSwitch({regime,country,general,checked,onChange}:{regime:SpecialTaxRegime;country:string;general:string;checked:boolean;onChange:(on:boolean)=>void}){const status=specialTaxStatusOn(regime,new Date().toLocaleDateString("en-CA"));const notice=checked?specialTaxNotice(status,general,country):null;return <><label className="switch-row compact span-2"><input type="checkbox" checked={checked} onChange={e=>onChange(e.target.checked)}/><span/><b>{regime.name}<small>{checked?`En el padrón de SUNAT. ${specialTaxToday(status,country)}.`:"Márcalo solo si la empresa está en el padrón MYPE de SUNAT."}</small></b></label>{notice&&<InlineNotice className="span-2" tone={notice.tone} title={notice.title}>{notice.text}</InlineNotice>}</>}
