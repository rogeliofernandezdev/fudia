import type {PublicTableInfo} from "../domain/types";

export async function getPublicTable(token:string):Promise<PublicTableInfo>{
 const response=await fetch(`/api/public/tables/${encodeURIComponent(token)}`);
 const body=await response.json();
 if(!response.ok)throw new Error(body.message??"No pudimos cargar la mesa.");
 return body as PublicTableInfo;
}
