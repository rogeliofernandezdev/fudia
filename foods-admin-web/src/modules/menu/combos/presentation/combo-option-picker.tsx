"use client";
import dynamic from "next/dynamic";
import {Skeleton} from "@/design-system/skeleton";

/**
 * Buscador «Agregar opción» con carga diferida: react-select (y emotion) solo se
 * descarga al abrir la composición del combo, no con la lista de combos. Mientras
 * llega, un skeleton con la altura del control evita saltos de layout.
 */
export const ComboOptionPicker=dynamic(()=>import("./combo-option-search").then(module=>module.ComboOptionSearch),{ssr:false,loading:()=><Skeleton height="var(--control-height)"/>});
