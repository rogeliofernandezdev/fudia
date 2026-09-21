"use client";
import "./profile-page.css";

import {useForm} from "react-hook-form";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Icon,Input,PageHeader} from "@/design-system";
import {useFeedback} from "@/providers";
import type {MyProfile,MyProfileDraft} from "../domain/types";
import {profileResolver} from "../domain/profile-schema";
import {getMyProfile,saveMyProfile} from "../infrastructure/identity-api";

export function ProfilePage(){
  const profile=useQuery({queryKey:["my-profile"],queryFn:getMyProfile});
  if(profile.isLoading)return <><ProfileHeader/><ProfileSkeleton/></>;
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
  const{
    register,handleSubmit,reset,
    formState:{errors,isDirty},
  }=useForm<MyProfileDraft>({
    defaultValues:{fullName:profile.fullName,currentPassword:"",newPassword:""},
    resolver:profileResolver,
    mode:"onSubmit",
    reValidateMode:"onChange",
  });

  const save=useMutation({
    mutationFn:saveMyProfile,
    onSuccess:data=>{
      reset({fullName:data.fullName,currentPassword:"",newPassword:""});
      void qc.invalidateQueries({queryKey:["session-context"]});
      void qc.invalidateQueries({queryKey:["my-profile"]});
      notify({tone:"success",title:"Perfil actualizado",message:"Los cambios de tu cuenta quedaron guardados."});
    },
    onError:error=>notify({tone:"danger",title:"No se pudo actualizar",message:error.message}),
  });

  function submit(draft:MyProfileDraft){
    save.mutate(draft);
  }

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

        <form className="profile-form" onSubmit={handleSubmit(submit)} noValidate>
          <section className="profile-section">
            <div className="profile-section-heading">
              <span><Icon name="users" size={16}/></span>
              <div><b>Datos personales</b><small>Información utilizada para identificarte dentro del sistema.</small></div>
            </div>
            <div className="profile-grid">
              <label className="profile-field">
                <span>Nombre completo</span>
                <Input autoFocus maxLength={180} {...register("fullName")} aria-invalid={Boolean(errors.fullName)}/>
                {errors.fullName?.message&&<small className="wizard-field-error">{errors.fullName.message}</small>}
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
                <Input type="password" autoComplete="current-password" {...register("currentPassword")} aria-invalid={Boolean(errors.currentPassword)} placeholder="Ingresa tu contraseña actual"/>
                {errors.currentPassword?.message&&<small className="wizard-field-error">{errors.currentPassword.message}</small>}
              </label>
              <label className="profile-field">
                <span>Nueva contraseña <small>Mínimo 8 caracteres</small></span>
                <Input type="password" autoComplete="new-password" {...register("newPassword")} aria-invalid={Boolean(errors.newPassword)} placeholder="Déjala vacía para conservarla"/>
                {errors.newPassword?.message&&<small className="wizard-field-error">{errors.newPassword.message}</small>}
              </label>
            </div>
            <div className="profile-password-note"><Icon name="lock" size={14}/><span>Si no deseas cambiar tu contraseña, deja ambos campos vacíos. Tus roles y permisos no se modifican desde esta pantalla.</span></div>
          </section>

          <footer className="profile-actions">
            <span><b>Cambios de cuenta</b><small>Se aplicarán únicamente a tu perfil personal.</small></span>
            <Button type="submit" icon="check" disabled={save.isPending||!isDirty}>{save.isPending?"Guardando…":"Guardar"}</Button>
          </footer>
        </form>
      </section>
    </div>
  </>;
}

function ProfileSkeleton(){
  return <div className="profile-shell" aria-label="Cargando perfil" aria-busy="true">
    <section className="profile-card profile-skeleton">
      <header><i/><div><b/><span/><span/></div></header>
      <section><i/><div><span/><span/></div></section>
      <section><i/><div><span/><span/></div></section>
      <footer><span/><i/></footer>
    </section>
  </div>;
}
