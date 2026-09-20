export type SessionContextResponse={
  user:{id:string;name:string;platformAdmin:boolean};
  organization:{id:string;name:string};
  location:{id:string;name:string};
  modules:Record<string,boolean>;
  menuAccess:string[];
  permissions:string[];
};

export async function loadSessionContext():Promise<SessionContextResponse>{
  const response=await fetch("/api/admin/context");
  if(!response.ok)throw new Error(`context_${response.status}`);
  return response.json();
}

export async function deleteSession(){
  const response=await fetch("/api/session",{method:"DELETE"});
  if(!response.ok&&response.status!==204)throw new Error(`session_${response.status}`);
}
