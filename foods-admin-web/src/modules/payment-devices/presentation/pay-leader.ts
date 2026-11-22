/**
 * Una sola pestaña de Fudia web por cajero y local avisa los pagos (sonido,
 * parpadeo del título y push a los teléfonos); las demás solo muestran las
 * tarjetas. Se coordina con Web Locks: la pestaña que tiene el candado es la
 * líder, y al cerrarse otra lo toma sola. La pestaña que el cajero enfoca lo
 * reclama (`steal`), porque ahí ocurrió su último gesto y el navegador la deja
 * sonar. Sin Web Locks, cada pestaña se considera líder (comportamiento previo).
 */
type Locks={request:(name:string,options:{steal?:boolean;signal?:AbortSignal},callback:()=>Promise<void>)=>Promise<unknown>};

export type PayLeadership={claim:()=>void;dispose:()=>void};

export function holdPayLeadership(locks:Locks|undefined,name:string,onChange:(leader:boolean)=>void):PayLeadership{
  if(!locks){onChange(true);return {claim:()=>{},dispose:()=>{}}}
  let disposed=false;
  let leader=false;
  let release:(()=>void)|null=null;
  let waiting:AbortController|null=null;
  let claiming=false;

  const become=(value:boolean)=>{if(leader!==value){leader=value;onChange(value)}};

  const request=(steal:boolean)=>{
    const controller=steal?null:new AbortController();
    if(controller)waiting=controller;
    locks.request(name,steal?{steal:true}:{signal:controller!.signal},()=>{
      if(controller&&waiting===controller)waiting=null;
      if(steal)claiming=false;
      if(disposed)return Promise.resolve();
      become(true);
      return new Promise<void>(resolve=>{release=resolve});
    }).then(()=>undefined,()=>{
      // Rechazo: otra pestaña lo reclamó (pierde el liderazgo y vuelve a la
      // fila) o esta misma canceló su espera para reclamarlo (no hace nada).
      if(steal)claiming=false;
      if(disposed||controller?.signal.aborted)return;
      release=null;
      become(false);
      request(false);
    });
  };

  request(false);
  return {
    claim(){
      if(disposed||leader||claiming)return;
      claiming=true;
      waiting?.abort();
      waiting=null;
      request(true);
    },
    dispose(){
      disposed=true;
      waiting?.abort();
      release?.();
      release=null;
    },
  };
}

