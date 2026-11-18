"use client";

import {payProviderLabel} from "../domain/pay-status";
import type {PayProvider} from "../domain/types";

/** Identidad del proveedor: monograma con el color característico de Yape
 *  (morado) y Plin (verde azulado) más su nombre en texto. Sin logotipos de
 *  terceros: la identidad es de color y letra, no de marca registrada. */
export function PayProviderTag({provider}:{provider:PayProvider}){
  return <span className={`pay-provider-tag ${provider}`}>
    <span className="pay-provider-mark" aria-hidden="true">{provider==="yape"?"Y":"P"}</span>
    {payProviderLabel[provider]}
  </span>;
}
