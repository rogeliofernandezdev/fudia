"use client";
import "./profile-page.css";

import {useForm} from "react-hook-form";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Input,PageHeader} from "@/design-system";
import {useFeedback,useSession} from "@/providers";
import type {OrganizationSubscription} from "@/modules/platform";
import type {MyProfile,MyProfileDraft} from "../domain/types";
import {profileResolver} from "../domain/profile-schema";
import {getMyProfile,getOrganizationSubscription,saveMyProfile} from "../infrastructure/identity-api";

export function ProfilePage(){
  const{can}=useSession();
  const profile=useQuery({queryKey:["my-profile"],queryFn:getMyProfile});
  const canViewSubscription=can("organizations.read");
  const subscription=useQuery({queryKey:["organization-subscription"],queryFn:getOrganizationSubscription,enabled:canViewSubscription});
  if(profile.isLoading)return <><ProfileHeader/><ProfileSkeleton/></>;
  if(profile.isError)return <><ProfileHeader/><section className="profile-state"><p>{profile.error.message}</p><Button kind="secondary" onClick={()=>profile.refetch()}>Reintentar</Button></section></>;
  if(!profile.data)return <><ProfileHeader/><section className="profile-state"><p>No pudimos cargar tu perfil.</p></section></>;
  return <><ProfileHeader/><ProfileForm profile={profile.data}/>{canViewSubscription&&<SubscriptionPanel subscription={subscription.data??null} loading={subscription.isLoading} error={subscription.isError?subscription.error.message:""} retry={()=>subscription.refetch()}/>}</>;
}

function ProfileHeader(){
  return <PageHeader eyebrow="CUENTA" title="Mi perfil y seguridad" description="Actualiza tus datos personales y consulta la suscripción vigente de tu empresa."/>;
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

  return <div className="profile-shell">
    <section className="profile-card">
      <form className="profile-form" onSubmit={handleSubmit(draft=>save.mutate(draft))} noValidate>
        <section className="profile-section">
          <div className="profile-section-title"><h2>Datos personales</h2></div>
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
          <div className="profile-section-title"><h2>Contraseña</h2></div>
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
        </section>

        <footer className="profile-actions">
          <Button type="submit" icon="check" disabled={save.isPending||!isDirty}>{save.isPending?"Guardando…":"Guardar"}</Button>
        </footer>
      </form>
    </section>
  </div>;
}

const statusLabel:Record<OrganizationSubscription["status"],string>={trial:"Prueba",active:"Activa",past_due:"Pago pendiente",cancelled:"Cancelada"};
function dateLabel(value:string|null){
  if(!value)return "—";
  const date=new Date(value);
  return Number.isNaN(date.getTime())?"—":new Intl.DateTimeFormat("es-PE",{dateStyle:"medium"}).format(date);
}

function SubscriptionPanel({subscription,loading,error,retry}:{subscription:OrganizationSubscription|null;loading:boolean;error:string;retry:()=>void}){
  if(loading)return <section className="profile-subscription profile-card"><div className="profile-subscription-loading">Cargando suscripción…</div></section>;
  if(error||!subscription)return <section className="profile-subscription profile-card"><div className="profile-state"><p>{error||"No hay una suscripción registrada para esta empresa."}</p><Button kind="secondary" onClick={retry}>Reintentar</Button></div></section>;
  const payment=subscription.payments[0];
  const legacy=subscription.plan.code==="legacy";
  return <section className="profile-subscription profile-card">
    <header><div><small>SUSCRIPCIÓN DE LA EMPRESA</small><h2>{subscription.plan.name}</h2><p>{subscription.plan.description}</p></div><span className={"subscription-status "+subscription.status}>{statusLabel[subscription.status]}</span></header>
    <div className="subscription-overview">
      <article><small>PRECIO</small><b>{legacy?"No registrado":subscription.currency+" "+Number(subscription.priceAmount).toFixed(2)}</b><span>{subscription.billingCycle==="annual"?"Facturación anual":"Facturación mensual"}</span></article>
      <article><small>RENOVACIÓN</small><b>{dateLabel(subscription.renewsAt)}</b><span>{subscription.autoRenew?"Renovación automática":"Renovación manual"}</span></article>
      <article><small>PRUEBA GRATUITA</small><b>{subscription.plan.trialDays?subscription.plan.trialDays+" días":"No incluida"}</b><span>{subscription.trialEndsAt?"Hasta "+dateLabel(subscription.trialEndsAt):"Sin periodo de prueba activo"}</span></article>
      <article><small>CONDICIONES</small><b>{subscription.termsVersion??"Pendientes"}</b><span>{subscription.termsAcceptedAt?"Aceptadas "+dateLabel(subscription.termsAcceptedAt):"Sin aceptación registrada"}</span></article>
    </div>
    <div className="subscription-limits">
      <div><span>Locales</span><b>{subscription.usage.locations+" / "+(subscription.plan.maxLocations??"∞")}</b></div>
      <div><span>Usuarios</span><b>{subscription.usage.users+" / "+(subscription.plan.maxUsers??"∞")}</b></div>
      <div><span>Módulos incluidos</span><b>{subscription.plan.moduleKeys.length}</b></div>
    </div>
    <section className="subscription-payment">
      <div><small>ÚLTIMO PAGO</small><h3>{payment?payment.currency+" "+Number(payment.amount).toFixed(2):"Sin pagos registrados"}</h3></div>
      {payment&&<div className="subscription-payment-meta"><span>{payment.status==="paid"?"Pagado":payment.status==="pending"?"Pendiente":payment.status==="failed"?"Fallido":"Reembolsado"}</span><b>{payment.provider}</b><small>{dateLabel(payment.paidAt??payment.createdAt)}{payment.externalReference?" · "+payment.externalReference:""}</small></div>}
    </section>
    <footer><span>Los cambios de plan, estado y pagos los administra FUDIA desde Plataforma.</span></footer>
  </section>;
}

function ProfileSkeleton(){
  return <div className="profile-shell" aria-label="Cargando perfil" aria-busy="true">
    <section className="profile-card profile-skeleton">
      <section><i/><div><span/><span/></div></section>
      <section><i/><div><span/><span/></div></section>
      <footer><i/></footer>
    </section>
  </div>;
}
