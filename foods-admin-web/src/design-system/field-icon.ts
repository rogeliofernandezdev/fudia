import type {ReactNode} from "react";
import {isValidElement} from "react";
import type {IconName} from "./icons";

/**
 * Icono al inicio de un campo según su etiqueta. Un solo mapeo para toda la
 * aplicación: el mismo concepto lleva siempre el mismo icono.
 */
const rules:[RegExp,IconName][]=[
  [/correo|e-?mail/i,"mail"],
  [/tel[eé]fono|celular|whatsapp/i,"phone"],
  [/contrase[ñn]a/i,"lock"],
  [/direcci[oó]n|ubicaci[oó]n|distrito|ciudad|pa[ií]s/i,"mapPin"],
  [/\bruc\b|\bdni\b|documento|identificaci[oó]n|raz[oó]n social/i,"badge"],
  [/fecha|desde|hasta|vigencia|vence/i,"calendar"],
  [/hora|horario|duraci[oó]n|tiempo/i,"clock"],
  [/monto|fondo|importe|efectivo|contado|precio|costo|tarifa|recargo|propina|total/i,"cash"],
  [/c[oó]digo|clave/i,"key"],
  [/versi[oó]n|condiciones|contrato/i,"contract"],
  [/d[ií]as|periodo|ciclo/i,"calendar"],
  [/conversaci/i,"chat"],
  [/(^|\s)m[aá]x\.?(\s|$)|(^|\s)m[ií]n\.?(\s|$)/i,"gauge"],
  [/proveedor/i,"truck"],
  [/cliente|mozo|cajero|responsable|usuario|contacto/i,"user"],
  [/personas|comensales|segmento/i,"users"],
  [/categor[ií]a/i,"tag"],
  [/local|sede|sucursal/i,"store"],
  [/mesa/i,"tables"],
  [/zona/i,"layout"],
  [/\brol\b|roles|permiso/i,"key"],
  [/medio de pago|m[eé]todo de pago|forma de pago/i,"payment"],
  [/moneda|tipo de cambio/i,"cash"],
  [/impuesto|igv|tasa|porcentaje|descuento/i,"percent"],
  [/referencia|comprobante|n\.?º|n[uú]mero de operaci[oó]n/i,"invoice"],
  [/unidad|presentaci[oó]n|art[ií]culo|insumo|existencia|stock/i,"stock"],
  [/producto|plato|receta|men[uú]|combo/i,"dish"],
  [/caja|turno/i,"register"],
  [/canal|origen/i,"route"],
  [/motivo|raz[oó]n|nota|observaci/i,"note"],
  [/estado/i,"circleCheck"],
  [/tipo|clase|uso/i,"layers"],
  [/cantidad|m[ií]nimo|m[aá]ximo|l[ií]mite|capacidad|asientos/i,"gauge"],
  [/nombre|descripci[oó]n|t[ií]tulo|etiqueta|alias|concepto/i,"edit"],
];

function text(node:ReactNode):string{
  if(typeof node==="string"||typeof node==="number")return String(node);
  if(Array.isArray(node))return node.map(text).join(" ");
  if(isValidElement(node))return text((node.props as {children?:ReactNode}).children);
  return "";
}

export function fieldIcon(label:ReactNode):IconName|undefined{
  const value=text(label);
  return rules.find(([pattern])=>pattern.test(value))?.[1];
}
