const entities:Record<string,string>={amp:"&",lt:"<",gt:">",quot:"\"",apos:"'",nbsp:" "};

/**
 * Texto legible de una descripción guardada con el editor enriquecido («<p>Lomo…</p>»):
 * quita etiquetas, decodifica entidades y une espacios. Para mostrarla en una línea o tooltip.
 */
export function plainText(html:string|null|undefined){
 if(!html)return "";
 return html
  .replace(/<\s*(br|\/p|\/li|\/div|\/h\d)\s*\/?>/gi," ")
  .replace(/<[^>]*>/g,"")
  .replace(/&(#x[0-9a-f]+|#\d+|[a-z]+);/gi,(match,code:string)=>{
   if(code[0]==="#"){const value=code[1]?.toLowerCase()==="x"?parseInt(code.slice(2),16):parseInt(code.slice(1),10);return Number.isFinite(value)?String.fromCodePoint(value):match}
   return entities[code.toLowerCase()]??match;
  })
  .replace(/\s+/g," ")
  .trim();
}
