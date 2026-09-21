"use client";
import "./profile-page.css";

import {useState} from "react";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Icon,Input,PageHeader} from "@/design-system";
import {useFeedback} from "@/providers";
import type {MyProfile} from "../domain/types";
import {getMyProfile,saveMyProfile} from "../infrastructure/identity-api";

export function ProfilePage(){
  const profile=useQuery({queryKey:["my-profile"],queryFn:getMyProfile});
  if(profile.isLoading)return <><ProfileHeader/><section className="profile-state"><p>Cargando perfil…</p></section></>;
  if(profile.isError)return <><ProfileHeader/><section className="profile-state"><p>{profile.error.message}</p><Button kind="secondary" onClick={()=>profile.refetch()}>Reintentar</Button></section></>;
  if(!profile.data)return <><ProfileHeader/><section className="profile-state"><p>No pudimos cargar tu perfil.</p></section></>;
  return <ProfileForm profile={profile.data}/>;
}

function ProfileHeader(){
  return <PageHeader eyebrow="CUENTA" title="Mi perfil y seguridad" description="Actualiza tus datos personales y administra tu contraseña desde un solo lugar."/>;
}

function ProfileForm({profile}:{profile:MyProfile}){
  const qc=useQueryClient();
  const{notify}=useFeedback();
  const[fullName,setFullName]=useState(profile.fullName);
  const[currentPassword,setCurrentPassword]=useState("");
  const[newPassword,setNewPassword]=useState("");

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

  const passwordChanging=Boolean(newPassword);

  return <><ProfileHeader/>
    <div className="profile-shell">
      <section className="profile-card">
        <header>
          <span className="profile-card-icon"><Icon name="users" size={19}/></span>
          <div>
            <small>CUENTA PERSONAL</small>
            <h2>Información de tu cuenta</h2>
            <p>Estos datos corresponden a tu usuario. Tus roles y accesos se administran por separado.</p>
          </div>
        </header>

        <form className="profile-form" onSubmit={event=>{event.preventDefault();save.mutate()}}>
          <section className="profile-section">
            <div className="profile-section-heading">
              <span><Icon name="users" size={16}/></span>
              <div><b>Datos personales</b><small>Información utilizada para identificarte dentro del sistema.</small></div>
            </div>
            <div className="profile-grid">
              <label className="profile-field">
                <span>Nombre completo</span>
                <Input required maxLength={180} value={fullName} onChange={event=>setFullName(event.target.value)}/>
              </label>
              <label className="profile-field">
                <span>Correo electrónico <small>No editable</small></span>
                <Input readOnly value={profile.email}/>
              </label>
            </div>
          </section>

          <section className="profile-section">
            <div className="profile-section-heading">
              <span><Icon name="lock" size={16}/></span>
              <div><b>Cambiar contraseña</b><small>Solo completa estos campos cuando quieras definir una nueva contraseña.</small></div>
            </div>
            <div className="profile-grid">
              <label className="profile-field">
                <span>Contraseña actual</span>
                <Input type="password" autoComplete="current-password" value={currentPassword} onChange={event=>setCurrentPassword(event.target.value)} required={passwordChanging} placeholder="Ingresa tu contraseña actual"/>
              </label>
              <label className="profile-field">
                <span>Nueva contraseña <small>Mínimo 8 caracteres</small></span>
                <Input type="password" autoComplete="new-password" minLength={8} value={newPassword} onChange={event=>setNewPassword(event.target.value)} placeholder="Déjala vacía para conservarla"/>
              </label>
            </div>
            <div className="profile-password-note"><Icon name="lock" size={14}/><span>Si no deseas cambiar tu contraseña, deja ambos campos vacíos. Tus roles y permisos no se modifican desde esta pantalla.</span></div>
          </section>

          <footer className="profile-actions">
            <span><b>Cambios de cuenta</b><small>Se aplicarán únicamente a tu perfil personal.</small></span>
            <Button type="submit" disabled={save.isPending||!fullName.trim()||Boolean(newPassword&&!currentPassword)}>{save.isPending?"Guardando…":"Guardar cambios"}</Button>
          </footer>
        </form>
      </section>
    </div>
  </>;
}
