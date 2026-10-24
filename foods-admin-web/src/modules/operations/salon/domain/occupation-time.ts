/** El salón se ocupa al guardar la primera comanda activa de esa mesa. */
export type TableOccupationTime={
  label:string;
  description:string;
  needsReview:boolean;
  /** Sin hora de apertura válida no hay tiempo que mostrar en la tarjeta. */
  known:boolean;
};
export function formatTableOccupation(occupiedSince:string|undefined,nowMs:number):TableOccupationTime{
  const openedAt=occupiedSince?Date.parse(occupiedSince):NaN;
  if(!Number.isFinite(openedAt)||openedAt>nowMs)return{
    label:"—",description:"No hay una hora de apertura válida",needsReview:false,known:false,
  };
  const minutes=Math.floor((nowMs-openedAt)/60000);
  const days=Math.floor(minutes/1440);
  const hours=Math.floor((minutes%1440)/60);
  const remainder=minutes%60;
  const needsReview=days>=1;
  const label=needsReview?`${days} d ${hours} h`:hours>0?`${hours} h ${remainder} min`:`${minutes} min`;
  return{
    label,
    description:needsReview?`Ocupada desde hace ${label}. Revisar una posible comanda antigua abierta.`:`Mesa ocupada desde hace ${label}.`,
    needsReview,
    known:true,
  };
}
