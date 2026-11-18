"use client";

import {payProviderLabel} from "../domain/pay-status";
import type {PayProvider} from "../domain/types";

/** Identidad del proveedor sobre la etiqueta del sistema: tinte de marca de
 *  Yape (morado) y Plin (verde azulado), con su nombre en texto. Sin
 *  logotipos de terceros: la identidad es de color, no de marca registrada. */
export function PayProviderTag({provider}:{provider:PayProvider}){
  return <span className={`pay-provider-tag ${provider}`}>{payProviderLabel[provider]}</span>;
}
