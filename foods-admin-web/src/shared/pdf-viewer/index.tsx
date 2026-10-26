"use client";
import dynamic from "next/dynamic";

/** El visor, el motor y su worker se cargan exclusivamente al abrir en cliente. */
export const PdfViewer=dynamic(()=>import("./pdf-viewer").then(module=>module.PdfViewerModal),{ssr:false});
export type {PdfViewerProps} from "./types";
