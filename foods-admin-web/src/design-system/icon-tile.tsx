import {Icon,type IconName} from "./icons";

export type IconTileTone="primary"|"brand"|"digital"|"teal"|"sky"|"peach"|"rose"|"warning"|"danger"|"navy"|"info"|"purple"|"whatsapp"|"neutral";
/** Identidad de cada función. No representa el estado de un registro. */
const identityTones:Partial<Record<IconName,IconTileTone>>={
 user:"sky",users:"sky",badge:"sky",calendar:"sky",info:"sky",
 phone:"teal",mapPin:"teal",truck:"teal",cart:"teal",
 dish:"digital",utensils:"digital",lock:"digital",key:"digital",contract:"digital",
 chefHat:"brand",cookingPot:"brand",kitchen:"brand",register:"brand",cash:"brand",cashShift:"brand",wallet:"brand",
 receipt:"peach",layers:"peach",tag:"peach",tax:"peach",note:"peach",
};
export function getIdentityTone(icon:IconName):IconTileTone{return identityTones[icon]??"primary"}
type IconTileProps={name:IconName;size?:"sm"|"md"|"lg";className?:string;tone?:IconTileTone;appearance?:"soft"|"vivid"};

/**
 * Icono sobre un cuadrito tintado. Es la única forma de dar color de identidad
 * a un icono: el trazo es monocromo, con tinta según el contraste de su superficie.
 * `warning` y `danger` requieren una intención real; los estados usan `Status`.
 * `vivid` acentúa iconos de navegación, ingresos o egresos, nunca sus etiquetas.
 */
export function IconTile({name,tone="primary",appearance="soft",size="md",className=""}:IconTileProps){
  const glyph=size==="sm"?16:size==="lg"?20:18;
  return <span className={["icon-tile",`tone-${tone}`,appearance==="vivid"?"vivid":"",size!=="md"?`size-${size}`:"",className].filter(Boolean).join(" ")} aria-hidden="true"><Icon name={name} size={glyph}/></span>;
}
