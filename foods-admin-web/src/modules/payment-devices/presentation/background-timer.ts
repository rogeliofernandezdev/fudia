"use client";

/**
 * Temporizador que sigue a tiempo con la pestaña oculta. Chrome limita los
 * temporizadores de una pestaña oculta a uno por minuto, pero no los de un Web
 * Worker dedicado: la espera corre en el worker y solo el aviso vuelve a la
 * página. Sin soporte de Worker usa window.setTimeout.
 */
const WORKER_SOURCE="let timer;onmessage=event=>{clearTimeout(timer);if(event.data>=0)timer=setTimeout(()=>postMessage(0),event.data)};";

export type BackgroundTimer={after:(delay:number,run:()=>void)=>void;dispose:()=>void};

export function createBackgroundTimer():BackgroundTimer{
  let pending:(()=>void)|null=null;
  let worker:Worker|null=null;
  let url="";
  let fallback:number|undefined;
  try{
    url=URL.createObjectURL(new Blob([WORKER_SOURCE],{type:"text/javascript"}));
    worker=new Worker(url);
    worker.onmessage=()=>{const run=pending;pending=null;run?.()};
  }catch{
    worker=null;
  }
  return {
    // Una sola espera a la vez: programar otra reemplaza la anterior.
    after(delay,run){
      pending=run;
      if(worker){worker.postMessage(delay);return}
      window.clearTimeout(fallback);
      fallback=window.setTimeout(()=>{pending=null;run()},delay);
    },
    dispose(){
      pending=null;
      worker?.terminate();
      if(url)URL.revokeObjectURL(url);
      window.clearTimeout(fallback);
    },
  };
}
