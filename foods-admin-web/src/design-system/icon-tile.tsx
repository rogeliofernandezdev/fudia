import {Icon,type IconName} from "./icons";

export type IconTileTone="primary"|"brand"|"digital"|"teal"|"sky"|"peach"|"rose"|"warning"|"danger"|"neutral";
type IconTileProps={name:IconName;size?:"sm"|"md"|"lg";className?:string}&(
 {tone?:IconTileTone;appearance?:"soft"}|
 {tone:"primary"|"brand"|"danger";appearance:"vivid"}
);

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
