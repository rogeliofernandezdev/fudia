import type {ChatInput,ChatMenu,ChatState,ProductOptions} from "../domain/types";

async function call<T>(qr:string,operation:string,payload:unknown={}):Promise<T> {
  const response=await fetch(`/api/public/chat/${encodeURIComponent(qr)}/${operation}`,{method:"POST",headers:{"Content-Type":"application/json"},body:JSON.stringify(payload),cache:"no-store"});
  const data=await response.json();
  if (!response.ok) throw new Error(data.message??"No pudimos cargar el chat.");
  return data as T;
}
export const openChat=(qr:string,scan="")=>call<{ready:boolean}>(qr,"session",scan?{scan}:{});
export const loadChat=(qr:string)=>call<ChatState>(qr,"state");
export const loadChatMenu=(qr:string)=>call<ChatMenu>(qr,"menu");
export const loadProductOptions=(qr:string,productId:string)=>call<ProductOptions>(qr,"options",{productId});
export const sendChatInput=(qr:string,input:ChatInput)=>call<{accepted:boolean}>(qr,"messages",input);
export function newChatRequestId() {
  return Array.from(crypto.getRandomValues(new Uint8Array(18)),byte=>byte.toString(16).padStart(2,"0")).join("");
}
