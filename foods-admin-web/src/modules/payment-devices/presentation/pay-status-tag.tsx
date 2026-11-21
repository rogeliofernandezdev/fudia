"use client";

import {payProviderLabel} from "../domain/pay-status";
import type {PayProvider} from "../domain/types";
import "./payment-devices.css";

/**
 * Billetera del pago. `logo`: logos originales, alojados localmente y sin filtros
 * que alteren sus colores. `circle` (2026-11-21): círculo lleno de 40 px con el
 * nombre en blanco, sin logotipo; lo usan la tabla de Pagos detectados y los
 * Avisos. `decorative` lo oculta a lectores de pantalla cuando el texto vecino ya
 * nombra la billetera.
 */
export function PayProviderTag({provider,size="md",appearance="logo",decorative=false}:{provider:PayProvider;size?:"sm"|"md";appearance?:"logo"|"circle";decorative?:boolean}){
  if(appearance==="circle")return <span className="pay-wallet-mark" data-provider={provider} aria-hidden={decorative||undefined}>{payProviderLabel[provider]}</span>;
  const dimension=size==="sm"?32:56;
  return <span className={`pay-provider-tag ${provider}${size==="sm"?" is-small":""}`}>
    {/* eslint-disable-next-line @next/next/no-img-element */}
    <img className="pay-provider-logo" src={`/brands/${provider}.png`} width={dimension} height={dimension} alt={payProviderLabel[provider]}/>
  </span>;
}
