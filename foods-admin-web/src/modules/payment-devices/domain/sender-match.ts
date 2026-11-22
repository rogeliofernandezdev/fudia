/** Nombre comparable: sin tildes (ñ → n), mayúsculas, signos ni espacios repetidos. */
const comparableName=(value?:string)=>(value??"").normalize("NFD").replace(/[̀-ͯ]/g,"").toLowerCase().replace(/[^a-z0-9 ]+/g," ").split(" ").filter(Boolean);

/** El remitente coincide con el cliente cuando cada palabra del nombre más
 * corto empieza la palabra correspondiente del otro, en orden: Yape y Plin
 * abrevian o recortan apellidos («Rogelio Fernandez A.»). Exige al menos dos
 * palabras para no confirmar solo por un nombre de pila. */
export function senderMatchesCustomer(sender?:string,customer?:string){
  const a=comparableName(sender),b=comparableName(customer);
  const [short,long]=a.length<=b.length?[a,b]:[b,a];
  return short.length>=2&&short.every((word,index)=>long[index]?.startsWith(word));
}
