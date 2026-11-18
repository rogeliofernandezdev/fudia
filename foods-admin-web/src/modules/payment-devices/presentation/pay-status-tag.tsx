"use client";

import {payProviderLabel} from "../domain/pay-status";
import type {PayProvider} from "../domain/types";
import "./payment-devices.css";

/** Logos originales, alojados localmente y sin filtros que alteren sus colores. */
export function PayProviderTag({provider}:{provider:PayProvider}){
  return <span className={`pay-provider-tag ${provider}`}>
    {/* eslint-disable-next-line @next/next/no-img-element */}
    <img className="pay-provider-logo" src={`/brands/${provider}.png`} width={56} height={56} alt={payProviderLabel[provider]}/>
  </span>;
}
