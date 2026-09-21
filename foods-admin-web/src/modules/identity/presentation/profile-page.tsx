"use client";

import {useEffect,useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Input,PageHeader} from "@/design-system";
import {useFeedback} from "@/providers";
import {getMyProfile,saveMyProfile} from "../infrastructure/identity-api";

export function ProfilePage(){
  const qc=useQueryClient();
  const{notify}=useFeedback();
  const profile=useQuery({queryKey:["my-profile"],queryFn:getMyProfile});
  const[fullName,setFullName]=useState("");
  const[currentPassword,setCurrentPassword]=useState("");
  const[newPassword,setNewPassword]=useState("");

  useEffect(()=>{if(profile.data)setFullName(profile.data.fullName)},[profile.data]);

  const save=useMutation({
    mutationFn:()=>saveMyProfile({fullName,currentPassword,newPassword}),
    onSuccess:data=>{
      setFullName(data.fullName);
      setCurrentPassword("");
      setNewPassword("");
      void qc.invalidateQueries({queryKey:["session-context"]});
      void qc.invalidateQueries({queryKey:["my-profile"]});
      notify({tone:"success",title:"Perfil actualizado",message:newPassword?"Tu perfil y contraseña quedaron actualizados.":"Tus datos personales quedaron actualizados."});
    },
    onError:error=>notify({tone:"danger",title:"No se pudo actualizar",message:error.message}),
  });

  if(profile.isLoading)return <><PageHeader eyebrow="CUENTA" title="Mi perfil y seguridad" description="Administra tus datos y tu contraseña."/><section className="panel management"><p>Cargando perfil…</p></section></>;
  if(profile.isError)return <><PageHeader eyebrow="CUENTA" title="Mi perfil y seguridad" description="Administra tus datos y tu contraseña."/><section className="panel management"><p>{profile.error.message}</p><Button kind="secondary" onClick={()=>profile.refetch()}>Reintentar</Button></section></>;

  return <><PageHeader eyebrow="CUENTA" title="Mi perfil y seguridad" description="Actualiza tu nombre y cambia tu contraseña sin afectar tus roles ni locales."/>
    <section className="panel management standardized-management">
      <form onSubmit={event=>{event.preventDefault();save.mutate()}}>
        <div className="form-grid">
          <label>Nombre completo<Input required maxLength={180} value={fullName} onChange={event=>setFullName(event.target.value)}/></label>
          <label>Correo electrónico<Input readOnly value={profile.data?.email??""}/></label>
          <label>Contraseña actual<Input type="password" autoComplete="current-password" value={currentPassword} onChange={event=>setCurrentPassword(event.target.value)} required={Boolean(newPassword)}/></label>
          <label>Nueva contraseña<Input type="password" autoComplete="new-password" minLength={8} value={newPassword} onChange={event=>setNewPassword(event.target.value)} placeholder="Déjala vacía para conservarla"/></label>
        </div>
        <footer><Button type="submit" disabled={save.isPending||!fullName.trim()||Boolean(newPassword&&!currentPassword)}>{save.isPending?"Guardando…":"Guardar cambios"}</Button></footer>
      </form>
    </section>
  </>;
}
