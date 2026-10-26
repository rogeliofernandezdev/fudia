import type {ReactNode} from "react";

/** Documentos obtenidos con la autenticación existente; nunca URLs con tokens. */
export type PdfViewerProps={
 document:Blob|null;
 filename:string;
 title:string;
 subtitle?:string;
 /** Mantiene las medidas de las vistas previas existentes: 480/960 px. */
 size?:"sm"|"lg";
 onClose:()=>void;
 loading?:boolean;
 error?:string|null;
 onRetry?:()=>void;
 notice?:ReactNode;
 details?:ReactNode;
};
