"use client";

import {payProviderLabel} from "../domain/pay-status";
import type {PayProvider} from "../domain/types";
import "./payment-devices.css";

/** Logos originales, alojados localmente y sin filtros que alteren sus colores. */
export function PayProviderTag({provider,size="md"}:{provider:PayProvider;size?:"sm"|"md"}){
  const dimension=size==="sm"?32:56;
  return <span className={`pay-provider-tag ${provider}${size==="sm"?" is-small":""}`}>
    {/* eslint-disable-next-line @next/next/no-img-element */}
    <img className="pay-provider-logo" src={`/brands/${provider}.png`} width={dimension} height={dimension} alt={payProviderLabel[provider]}/>
  </span>;
}
