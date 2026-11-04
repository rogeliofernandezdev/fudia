/**
 * Requisitos entre Menú y Acciones publicados por el API (`permission-catalog`):
 * una pantalla o acción marcada arrastra las acciones que el backend exige para
 * usarla. El backend las agrega igual al guardar; aquí se muestran marcadas, sin
 * poder quitarse, con el motivo («Necesario para Recetas»).
 */
export type RoleRequirements={menus:Record<string,string[]>;permissions:Record<string,string[]>};

/** Acción requerida → nombres de lo que la exige (pantallas y acciones marcadas), de forma transitiva. */
export function requiredPermissions(menus:readonly string[],permissions:readonly string[],requirements:RoleRequirements|undefined,labels:Record<string,string>={}):Map<string,string[]>{
  const required=new Map<string,string[]>();
  if(!requirements)return required;
  const reasonFor=(key:string)=>labels[key]??key;
  const visit=(permission:string,reason:string)=>{
    const reasons=required.get(permission);
    if(reasons){if(!reasons.includes(reason))reasons.push(reason);return;}
    required.set(permission,[reason]);
    for(const next of requirements.permissions[permission]??[])visit(next,reason);
  };
  for(const menu of menus)for(const permission of requirements.menus[menu]??[])visit(permission,reasonFor(`menu:${menu}`));
  for(const permission of permissions)for(const next of requirements.permissions[permission]??[])visit(next,reasonFor(permission));
  return required;
}

/** Acciones que se guardan: las marcadas más las requeridas, sin duplicados. */
export function effectivePermissions(permissions:readonly string[],required:Map<string,string[]>):string[]{
  return [...new Set([...permissions,...required.keys()])];
}
