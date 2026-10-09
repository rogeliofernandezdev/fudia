"use client";
import "./profile-page.css";

import {useForm} from "react-hook-form";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,FormField,Icon,Input,PageHeader} from "@/design-system";
import {useFeedback,useSession} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import type {OrganizationSubscription} from "@/modules/platform";
import type {MyProfile,MyProfileDraft} from "../domain/types";
import {profileResolver} from "../domain/profile-schema";
import {getOrganizationSubscription,saveMyProfile} from "../infrastructure/identity-api";
import {useCurrentProfile} from "../application/use-current-profile";

export function ProfilePage(){
  const{can,organization,location}=useSession();
  const profile=useCurrentProfile();
  const canViewSubscription=can("subscription.read");
  const subscription=useQuery({queryKey:["organization-subscription"],queryFn:getOrganizationSubscription,enabled:canViewSubscription});
  if(profile.isLoading)return <><ProfileHeader/><ProfileSkeleton/>{canViewSubscription&&<SubscriptionSkeleton/>}</>;
  if(profile.isError)return <><ProfileHeader/><section className="profile-card profile-state"><p>{profile.error.message}</p><Button kind="secondary" onClick={()=>profile.refetch()}>Reintentar</Button></section></>;
  if(!profile.data)return <><ProfileHeader/><section className="profile-card profile-state"><p>No pudimos cargar tu perfil.</p></section></>;
  return <><ProfileHeader/>
    <div className="profile-layout">
      <ProfileForm profile={profile.data} organizationName={organization?.name} locationName={location?.name}/>
      {canViewSubscription&&<SubscriptionPanel subscription={subscription.data??null} loading={subscription.isLoading} error={subscription.isError?subscription.error.message:""} retry={()=>subscription.refetch()} country={location?.country} timeZone={location?.timezone}/>}
    </div>
  </>;
}

function ProfileHeader(){
  return <PageHeader eyebrow="CUENTA" title="Mi perfil y seguridad" description="Tus datos personales, tu contraseña y la suscripción de tu empresa."/>;
}

function initials(value:string){
  return value.split(/\s+/).filter(Boolean).slice(0,2).map(part=>part[0]?.toUpperCase()??"").join("")||"FU";
}

function ProfileForm({profile,organizationName,locationName}:{profile:MyProfile;organizationName?:string;locationName?:string}){
  const qc=useQueryClient();
  const{notify}=useFeedback();
  const{register,handleSubmit,reset,formState:{errors,isDirty}}=useForm<MyProfileDraft>({
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
  const role=profile.platformAdmin?"Administrador de plataforma":profile.roleNames.length?profile.roleNames.join(" · "):"Sin rol efectivo";
  const context=[organizationName,locationName].filter(Boolean).join(" · ");

  return <form className="profile-card profile-account" onSubmit={handleSubmit(draft=>save.mutate(draft))} noValidate>
    <header className="profile-account-head">
      <span className="profile-avatar">{initials(profile.fullName)}</span>
      <div className="profile-account-copy">
        <h2>{profile.fullName}</h2>
        <p>{role}{context?" · "+context:""}</p>
      </div>
    </header>

    <div className="profile-fields">
      <FormField label="Nombre completo" error={errors.fullName?.message}><Input maxLength={180} {...register("fullName")}/></FormField>
      <label className="profile-field form-field">
        <span className="form-field-label">Correo electrónico<em>Solo lectura</em></span>
        <Input className="ds-input" readOnly value={profile.email} tabIndex={-1}/>
      </label>
    </div>

    <section className="profile-password">
      <h3>Cambiar contraseña</h3>
      <div className="profile-fields">
        <FormField label="Contraseña actual" error={errors.currentPassword?.message}><Input type="password" autoComplete="current-password" {...register("currentPassword")}/></FormField>
        <FormField label="Nueva contraseña" help="Mínimo 8 caracteres. Deja ambos campos vacíos si no vas a cambiarla." error={errors.newPassword?.message}><Input type="password" autoComplete="new-password" {...register("newPassword")}/></FormField>
      </div>
    </section>

    {!profile.platformAdmin&&profile.permissions.length===0&&<div className="profile-access-warning" role="status">
      <Icon name="alert" size={16}/>
      <span><b>Sin permisos en el local activo.</b> Puedes iniciar sesión, pero tu rol no habilita acciones. Pide a un administrador que revise tu asignación.</span>
    </div>}

    <footer className="profile-actions">
      <Button type="submit" icon="check" disabled={save.isPending||!isDirty}>{save.isPending?"Guardando…":"Guardar"}</Button>
    </footer>
  </form>;
}

const statusLabel:Record<OrganizationSubscription["status"],string>={trial:"Prueba",active:"Activa",past_due:"Pago pendiente",cancelled:"Cancelada"};
function dateLabel(value:string|null,country?:string,timeZone?:string){
  if(!value)return "—";
  return formatRegionalDateTime(value,{country,timeZone},{dateStyle:"medium"});
}

function SubscriptionPanel({subscription,loading,error,retry,country,timeZone}:{subscription:OrganizationSubscription|null;loading:boolean;error:string;retry:()=>void;country?:string;timeZone?:string}){
  if(loading)return <SubscriptionSkeleton/>;
  if(error||!subscription)return <section className="profile-card profile-state"><p>{error||"No hay una suscripción registrada para esta empresa."}</p><Button kind="secondary" onClick={retry}>Reintentar</Button></section>;

  const payment=subscription.payments.find(item=>item.status==="paid")??subscription.payments[0];
  const legacy=subscription.plan.code==="legacy";
  const price=legacy?"Sin precio registrado":subscription.currency+" "+Number(subscription.priceAmount).toFixed(2)+(subscription.billingCycle==="annual"?" / año":" / mes");
  const nextLabel=subscription.status==="trial"?"Fin de prueba":subscription.autoRenew?"Próxima renovación":"Fin del periodo";
  const nextDate=subscription.status==="trial"?subscription.trialEndsAt:subscription.renewsAt??subscription.currentPeriodEndsAt;

  return <section className="profile-card profile-subscription">
    <header className="profile-subscription-head">
      <div>
        <small>Suscripción de la empresa</small>
        <h2>{subscription.plan.name}</h2>
        <p>{price}{!legacy&&(subscription.autoRenew?" · Renovación automática":" · Renovación manual")}</p>
      </div>
      <span className={"profile-status "+subscription.status}>{statusLabel[subscription.status]}</span>
    </header>

    {legacy&&<div className="profile-note"><Icon name="alert" size={15}/><span>Plan anterior al catálogo comercial. FUDIA migrará precio, renovación y condiciones desde Plataforma.</span></div>}

    <dl className="profile-facts">
      <div><dt>{nextLabel}</dt><dd>{dateLabel(nextDate,country,timeZone)}</dd></div>
      <div><dt>Condiciones</dt><dd>{subscription.termsVersion??"Pendientes"}<small>{subscription.termsAcceptedAt?"Aceptadas "+dateLabel(subscription.termsAcceptedAt,country,timeZone):"Sin aceptación registrada"}</small></dd></div>
      <div><dt>Último pago</dt><dd>{payment?payment.currency+" "+Number(payment.amount).toFixed(2):"—"}{payment&&<small>{dateLabel(payment.paidAt??payment.createdAt,country,timeZone)}</small>}</dd></div>
      <div><dt>Módulos incluidos</dt><dd>{subscription.plan.moduleKeys.length}</dd></div>
    </dl>

    <div className="profile-usage">
      <UsageRow label="Locales" value={subscription.usage.locations} max={subscription.plan.maxLocations}/>
      <UsageRow label="Usuarios" value={subscription.usage.users} max={subscription.plan.maxUsers}/>
    </div>

    <footer className="profile-subscription-foot"><Icon name="lock" size={13}/><span>Plan, estado y pagos se administran desde Plataforma.</span></footer>
  </section>;
}

function UsageRow({label,value,max}:{label:string;value:number;max:number|null}){
  const ratio=max?Math.min(1,value/max):0;
  const tone=max&&value>=max?" full":max&&ratio>=.8?" near":"";
  return <div className={"profile-usage-row"+tone}>
    <div><b>{label}</b><span>{max===null?value+" · sin límite":value+" de "+max}</span></div>
    <div className="profile-usage-track" role="progressbar" aria-label={label} aria-valuemin={0} aria-valuemax={max??undefined} aria-valuenow={value}><i style={{width:(max?ratio*100:0)+"%"}}/></div>
  </div>;
}

function ProfileSkeleton(){
  return <section className="profile-card profile-account profile-skeleton" aria-label="Cargando perfil" aria-busy="true">
    <header className="profile-account-head"><i className="profile-sk profile-sk-avatar"/><div className="profile-account-copy"><i className="profile-sk" style={{width:"40%",height:"var(--size-16)"}}/><i className="profile-sk" style={{width:"60%",height:"var(--size-11)"}}/></div></header>
    <div className="profile-fields">{[0,1].map(i=><div className="form-field" key={i}><i className="profile-sk" style={{width:"var(--size-100)",height:"var(--size-11)"}}/><i className="profile-sk profile-sk-control"/></div>)}</div>
    <section className="profile-password"><i className="profile-sk" style={{width:"var(--size-140)",height:"var(--size-13)"}}/><div className="profile-fields">{[0,1].map(i=><div className="form-field" key={i}><i className="profile-sk" style={{width:"var(--size-110)",height:"var(--size-11)"}}/><i className="profile-sk profile-sk-control"/></div>)}</div></section>
    <footer className="profile-actions"><i className="profile-sk" style={{width:"var(--size-120)",height:"var(--control-height)",borderRadius:"var(--radius-10)"}}/></footer>
  </section>;
}

function SubscriptionSkeleton(){
  return <section className="profile-card profile-subscription profile-skeleton" aria-label="Cargando suscripción" aria-busy="true">
    <header className="profile-subscription-head"><div><i className="profile-sk" style={{width:"var(--size-150)",height:"var(--size-10)"}}/><i className="profile-sk" style={{width:"var(--size-120)",height:"var(--size-20)"}}/><i className="profile-sk" style={{width:"var(--size-180)",height:"var(--size-11)"}}/></div><i className="profile-sk" style={{width:"var(--size-70)",height:"var(--size-22)",borderRadius:"var(--radius-99)"}}/></header>
    <dl className="profile-facts">{[0,1,2,3].map(i=><div key={i}><i className="profile-sk" style={{width:"var(--size-90)",height:"var(--size-10)"}}/><i className="profile-sk" style={{width:"var(--size-70)",height:"var(--size-14)"}}/></div>)}</dl>
    <div className="profile-usage">{[0,1].map(i=><div className="profile-usage-row" key={i}><i className="profile-sk" style={{width:"50%",height:"var(--size-11)"}}/><i className="profile-sk" style={{width:"100%",height:"var(--size-6)",borderRadius:"var(--radius-99)"}}/></div>)}</div>
  </section>;
}
