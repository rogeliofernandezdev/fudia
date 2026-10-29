import type {POSCollectionAccess} from "./types";
export function collectionShift(access:POSCollectionAccess|undefined,selectedId:string){
 if(!access?.canChooseShift)return access?.shift??null;
 const options=access.shifts??[];
 return options.length===1?options[0]:options.find(shift=>shift.id===selectedId)??null;
}
