export type SessionEvent="signed-in"|"signed-out";

const channelName="fudia-session";
let sourceId="";

function currentSource():string{
  if(!sourceId)sourceId=globalThis.crypto?.randomUUID?.()??`${Date.now()}-${Math.random()}`;
  return sourceId;
}

export function broadcastSessionChange(event:SessionEvent):void{
  if(typeof BroadcastChannel==="undefined")return;
  const channel=new BroadcastChannel(channelName);
  channel.postMessage({event,source:currentSource()});
  channel.close();
}

export function listenForSessionChange(handler:(event:SessionEvent)=>void):()=>void{
  if(typeof BroadcastChannel==="undefined")return()=>{};
  const channel=new BroadcastChannel(channelName);
  channel.onmessage=message=>{
    const data=message.data as {event?:SessionEvent;source?:string};
    if(data.source===currentSource())return;
    if(data.event==="signed-in"||data.event==="signed-out")handler(data.event);
  };
  return()=>channel.close();
}
