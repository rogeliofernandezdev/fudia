import {Icon,type IconName} from "./icons";

export type IconTileTone="primary"|"brand"|"digital"|"teal"|"sky"|"peach"|"rose"|"warning"|"neutral";

/**
 * Icono sobre un cuadrito tintado. Es la única forma de dar color de identidad
 * a un icono: el trazo siempre es monocromo y toma el tono oscuro de la familia.
 * `warning` se reserva para lo que pide atención; los estados usan `Status`.
 */
export function IconTile({name,tone="primary",size="md",className=""}:{name:IconName;tone?:IconTileTone;size?:"sm"|"md"|"lg";className?:string}){
  const glyph=size==="sm"?16:size==="lg"?20:18;
  return <span className={["icon-tile",`tone-${tone}`,size!=="md"?`size-${size}`:"",className].filter(Boolean).join(" ")} aria-hidden="true"><Icon name={name} size={glyph}/></span>;
}
