export function AccountRole({roleNames,platformAdmin,loading,error}:{roleNames?:string[];platformAdmin:boolean;loading:boolean;error?:string}){
  if(platformAdmin)return <small>Administrador</small>;
  if(loading)return <small className="account-role-loading" aria-label="Cargando rol" aria-busy="true"><span/></small>;
  if(error)return <small title={error}>Rol no disponible</small>;
  const names=roleNames?.map(name=>name.replace(/^Administrador de (empresa|plataforma)$/iu,"Administrador"))??[];
  const label=names.length?[...new Set(names)].join(" · "):"Sin rol asignado";
  return <small title={label}>{label}</small>;
}
