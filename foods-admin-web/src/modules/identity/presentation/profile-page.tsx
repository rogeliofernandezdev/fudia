"use client";
import "./profile-page.css";

import {useForm} from "react-hook-form";
import {useMutation,useQuery,useQueryClient} from "@tanstack/react-query";
import {Button,Icon,Input,PageHeader} from "@/design-system";
import {useFeedback,useSession} from "@/providers";
import {formatRegionalDateTime} from "@/shared/i18n/regional-format";
import type {OrganizationSubscription,SubscriptionPlan} from "@/modules/platform";
import type {MyProfile,MyProfileDraft} from "../domain/types";
import {profileResolver} from "../domain/profile-schema";
import {getAvailableSubscriptionPlans,getMyProfile,getOrganizationSubscription,saveMyProfile} from "../infrastructure/identity-api";

export function ProfilePage(){
  const{can,organization,location}=useSession();
  const profile=useQuery({queryKey:["my-profile"],queryFn:getMyProfile});
  const plans=useQuery({queryKey:["available-subscription-plans"],queryFn:getAvailableSubscriptionPlans});
  const canViewSubscription=can("subscription.read");
  const subscription=useQuery({queryKey:["organization-subscription"],queryFn:getOrganizationSubscription,enabled:canViewSubscription});
  if(profile.isLoading)return <><ProfileHeader/><ProfileSkeleton/></>;
  if(profile.isError)return <><ProfileHeader/><section className="profile-state"><p>{profile.error.message}</p><Button kind="secondary" onClick={()=>profile.refetch()}>Reintentar</Button></section></>;
  if(!profile.data)return <><ProfileHeader/><section className="profile-state"><p>No pudimos cargar tu perfil.</p></section></>;
  return <><ProfileHeader/>
    <ProfileForm profile={profile.data} organizationName={organization?.name} locationName={location?.name}/>
    {canViewSubscription&&<SubscriptionPanel subscription={subscription.data??null} loading={subscription.isLoading} error={subscription.isError?subscription.error.message:""} retry={()=>subscription.refetch()} country={location?.country} timeZone={location?.timezone}/>}
    <ProfilePlansPanel profile={profile.data} plans={plans.data?.items??[]} loading={plans.isLoading} error={plans.isError?plans.error.message:""} retry={()=>plans.refetch()}/>
  </>;
}

function ProfileHeader(){
  return <PageHeader eyebrow="CUENTA" title="Mi perfil y seguridad" description="Gestiona tus datos personales, acceso y seguridad desde un solo lugar."/>;
}

function initials(value:string){
  return value.split(/\s+/).filter(Boolean).slice(0,2).map(part=>part[0]?.toUpperCase()??"").join("")||"FU";
}

function ProfileForm({profile,organizationName,locationName}:{profile:MyProfile;organizationName?:string;locationName?:string}){
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
    <section className="profile-card profile-account">
      <div className="profile-account-main">
        <span className="profile-avatar">{initials(profile.fullName)}</span>
        <div className="profile-account-copy">
          <small>CUENTA PERSONAL</small>
          <div className="profile-account-title">
            <h2>{profile.fullName}</h2>
            <span className="profile-account-status"><i/>Activa</span>
          </div>
          <p><Icon name="mail" size={13}/>{profile.email}</p>
        </div>
      </div>
      <div className="profile-account-meta">
        <div><small>ROL</small><b>{profile.platformAdmin?"Administrador de plataforma":profile.roleNames.length?profile.roleNames.join(" · "):"Sin rol efectivo"}</b></div>
        {organizationName&&<div><small>EMPRESA</small><b>{organizationName}</b></div>}
        {locationName&&<div><small>LOCAL ACTIVO</small><b>{locationName}</b></div>}
        <div><small>PERMISOS</small><b>{profile.platformAdmin?"Acceso total":profile.permissions.length}</b></div>
      </div>
    </section>

    {!profile.platformAdmin&&profile.permissions.length===0&&<section className="profile-access-warning">
      <span><Icon name="alert" size={17}/></span>
      <div><b>Sin permisos efectivos en el local activo</b><p>La cuenta puede iniciar sesión, pero su rol actual no habilita endpoints protegidos. Un administrador debe revisar la asignación en Usuarios y permisos.</p></div>
    </section>}

    <form className="profile-card profile-settings-workspace" onSubmit={handleSubmit(draft=>save.mutate(draft))} noValidate>
      <header className="profile-settings-header">
        <div>
          <span><Icon name="settings" size={17}/></span>
          <div><small>CONFIGURACIÓN DE CUENTA</small><h2>Datos y seguridad</h2><p>Actualiza únicamente la información que necesites cambiar.</p></div>
        </div>
      </header>

      <div className="profile-settings-columns">
        <section className="profile-settings-section">
          <header>
            <span><Icon name="users" size={16}/></span>
            <div><small>IDENTIDAD</small><h3>Datos personales</h3></div>
          </header>
          <div className="profile-settings-fields">
            <label className="profile-field">
              <span>Nombre completo</span>
              <Input maxLength={180} {...register("fullName")} aria-invalid={Boolean(errors.fullName)}/>
              {errors.fullName?.message&&<small className="wizard-field-error">{errors.fullName.message}</small>}
            </label>
            <label className="profile-field">
              <span>Correo electrónico <small>Solo lectura</small></span>
              <div className="profile-readonly-field"><Icon name="mail" size={15}/><span>{profile.email}</span><Icon name="lock" size={13}/></div>
            </label>
          </div>
        </section>

        <section className="profile-settings-section security">
          <header>
            <span><Icon name="lock" size={16}/></span>
            <div><small>SEGURIDAD</small><h3>Cambiar contraseña</h3></div>
          </header>
          <div className="profile-settings-fields">
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
            <div className="profile-security-note"><Icon name="lock" size={14}/><span>Confirma tu contraseña actual solo si deseas reemplazarla.</span></div>
          </div>
        </section>
      </div>

      <footer className="profile-savebar">
        <div>
          <span className={isDirty?"dirty":""}><i/>{isDirty?"Cambios sin guardar":"Todo actualizado"}</span>
          <small>{isDirty?"Guarda para aplicar los cambios de esta cuenta.":"Tu perfil está sincronizado con FUDIA."}</small>
        </div>
        <Button type="submit" icon="check" disabled={save.isPending||!isDirty}>{save.isPending?"Guardando…":"Guardar"}</Button>
      </footer>
    </form>
  </div>;
}

function ProfilePlansPanel({profile,plans,loading,error,retry}:{profile:MyProfile;plans:SubscriptionPlan[];loading:boolean;error:string;retry:()=>void}){
  if(loading)return <section className="profile-card profile-plans-loading">Cargando planes disponibles…</section>;
  if(error)return <section className="profile-card profile-plans-loading error"><b>No pudimos cargar los planes.</b><Button kind="secondary" onClick={retry}>Reintentar</Button></section>;
  const current=profile.currentPlan;
  return <section className="profile-card profile-plans">
    <header>
      <div><small>PLANES FUDIA</small><h2>Tu plan y opciones disponibles</h2><p>Compara capacidad y módulos. El plan que usa tu empresa aparece resaltado.</p></div>
      <div className="profile-current-plan">
        <small>PLAN ACTUAL</small>
        <b>{current?.name??"Sin plan asignado"}</b>
        {current&&<span className={"plan-state-pill "+current.status}>{statusLabel[current.status]}</span>}
      </div>
    </header>
    <div className="profile-plan-grid">
      {plans.map(plan=>{
        const active=current?.id===plan.id;
        return <article className={"profile-plan-card plan-"+plan.code+(active?" current":"")} key={plan.id}>
          {active&&<div className="profile-plan-current"><Icon name="check" size={12}/>TU PLAN ACTUAL</div>}
          <div className="profile-plan-head">
            <span><Icon name={plan.code==="emprende"?"store":plan.code==="escala"?"grid":"sales"} size={18}/></span>
            <div><small>{plan.code.toUpperCase()}</small><h3>{plan.name}</h3></div>
          </div>
          <p>{plan.description}</p>
          <div className="profile-plan-price">
            <div><span>{plan.currency}</span><b>{Number(plan.monthlyPrice).toFixed(2)}</b><small>/ mes</small></div>
            <em>{plan.currency+" "+Number(plan.annualPrice).toFixed(2)+" / año"}</em>
          </div>
          <div className="profile-plan-facts">
            <div><span>Locales</span><b>{plan.maxLocations??"∞"}</b></div>
            <div><span>Usuarios</span><b>{plan.maxUsers??"∞"}</b></div>
            <div><span>Prueba</span><b>{plan.trialDays?plan.trialDays+" días":"No"}</b></div>
          </div>
          <div className="profile-plan-modules">
            <span><Icon name="grid" size={13}/>{plan.moduleKeys.length} módulos incluidos</span>
            <div>{plan.moduleKeys.slice(0,6).map(key=><small key={key}><Icon name="check" size={9}/>{planModuleLabel(key)}</small>)}</div>
            {plan.moduleKeys.length>6&&<em>+{plan.moduleKeys.length-6} módulos más</em>}
          </div>
          <footer>{active?<span><Icon name="check" size={13}/>Plan contratado</span>:<span><Icon name="check" size={12}/>Disponible para cambio de plan</span>}</footer>
        </article>;
      })}
    </div>
    {current?.code==="legacy"&&<div className="profile-plan-legacy"><Icon name="alert" size={15}/><span>La empresa sigue en Plan legado. Plataforma debe migrarla a Emprende, Impulso o Escala cuando corresponda.</span></div>}
  </section>;
}

const moduleLabels:Record<string,string>={
 reportes:"Reportes",pos:"Punto de venta",pedidos:"Pedidos",cocina:"Cocina",mesas:"Mesas",caja:"Caja",
 reservas:"Reservas",productos:"Productos",combos:"Combos",recetas:"Recetas",inventario:"Inventario",
 kardex:"Kardex",compras:"Compras",clientes:"Clientes",locales:"Locales",fiscal:"Fiscal",usuarios:"Usuarios",
};
function planModuleLabel(key:string){return moduleLabels[key]??key}


const statusLabel:Record<OrganizationSubscription["status"],string>={trial:"Prueba",active:"Activa",past_due:"Pago pendiente",cancelled:"Cancelada"};
function dateLabel(value:string|null,country?:string,timeZone?:string){
  if(!value)return "—";
  return formatRegionalDateTime(value,{country,timeZone},{dateStyle:"medium"});
}
function usagePercent(value:number,max:number|null){
  if(max===null)return value>0?Math.min(20+value*4,72):6;
  if(max<=0)return 0;
  return Math.min(100,Math.max(4,Math.round(value/max*100)));
}

function SubscriptionPanel({subscription,loading,error,retry,country,timeZone}:{subscription:OrganizationSubscription|null;loading:boolean;error:string;retry:()=>void;country?:string;timeZone?:string}){
  if(loading)return <section className="profile-subscription profile-card"><div className="profile-subscription-loading"><span/><div><b>Cargando suscripción</b><small>Consultando plan, límites y facturación…</small></div></div></section>;
  if(error||!subscription)return <section className="profile-subscription profile-card"><div className="profile-state"><p>{error||"No hay una suscripción registrada para esta empresa."}</p><Button kind="secondary" onClick={retry}>Reintentar</Button></div></section>;

  const payment=subscription.payments[0];
  const legacy=subscription.plan.code==="legacy";
  const price=legacy?"Sin precio registrado":subscription.currency+" "+Number(subscription.priceAmount).toFixed(2);
  const renewal=subscription.renewsAt?dateLabel(subscription.renewsAt,country,timeZone):"Sin fecha programada";
  const trial=subscription.plan.trialDays?subscription.plan.trialDays+" días":"No incluida";
  const terms=subscription.termsVersion??"Pendientes";

  return <section className={"profile-subscription profile-card"+(legacy?" legacy":"")}>
    <div className="subscription-hero">
      <div className="subscription-plan-identity">
        <span className="subscription-plan-icon"><Icon name="payment" size={22}/></span>
        <div>
          <small>SUSCRIPCIÓN DE LA EMPRESA</small>
          <div className="subscription-title-row"><h2>{subscription.plan.name}</h2>{legacy&&<span className="legacy-tag">HISTÓRICO</span>}</div>
          <p>{subscription.plan.description||"Plan comercial vigente para la empresa."}</p>
        </div>
      </div>
      <div className="subscription-hero-side">
        <span className={"subscription-status "+subscription.status}><i/>{statusLabel[subscription.status]}</span>
        <div className="subscription-price">
          <small>{subscription.billingCycle==="annual"?"FACTURACIÓN ANUAL":"FACTURACIÓN MENSUAL"}</small>
          <b>{price}</b>
          {!legacy&&<span>{subscription.autoRenew?"Renovación automática":"Renovación manual"}</span>}
        </div>
      </div>
    </div>

    {legacy&&<div className="subscription-legacy-note">
      <span><Icon name="alert" size={16}/></span>
      <div><b>Plan anterior al catálogo comercial</b><small>La empresa conserva acceso mientras FUDIA migra precio, renovación y condiciones a un plan comercial.</small></div>
    </div>}

    <div className="subscription-metrics">
      <article>
        <span className="subscription-metric-icon"><Icon name="payment" size={17}/></span>
        <div><small>PRECIO CONTRATADO</small><b>{price}</b><p>{subscription.billingCycle==="annual"?"Ciclo anual":"Ciclo mensual"}</p></div>
      </article>
      <article>
        <span className="subscription-metric-icon"><Icon name="clock" size={17}/></span>
        <div><small>PRÓXIMA RENOVACIÓN</small><b>{renewal}</b><p>{subscription.autoRenew?"Automática":"Manual"}</p></div>
      </article>
      <article>
        <span className="subscription-metric-icon"><Icon name="power" size={17}/></span>
        <div><small>PRUEBA GRATUITA</small><b>{trial}</b><p>{subscription.trialEndsAt?"Hasta "+dateLabel(subscription.trialEndsAt,country,timeZone):"Sin periodo activo"}</p></div>
      </article>
      <article>
        <span className="subscription-metric-icon"><Icon name="receipt" size={17}/></span>
        <div><small>CONDICIONES</small><b>{terms}</b><p>{subscription.termsAcceptedAt?"Aceptadas "+dateLabel(subscription.termsAcceptedAt,country,timeZone):"Aceptación pendiente"}</p></div>
      </article>
    </div>

    <div className="subscription-body">
      <section className="subscription-usage-card">
        <header><div><small>USO DEL PLAN</small><h3>Capacidad contratada</h3></div><span><Icon name="grid" size={16}/></span></header>
        <div className="subscription-usage-list">
          <UsageRow icon="store" label="Locales" value={subscription.usage.locations} max={subscription.plan.maxLocations}/>
          <UsageRow icon="users" label="Usuarios" value={subscription.usage.users} max={subscription.plan.maxUsers}/>
          <div className="subscription-usage-row modules">
            <span className="usage-icon"><Icon name="grid" size={15}/></span>
            <div className="usage-copy"><div><b>Módulos incluidos</b><strong>{subscription.plan.moduleKeys.length}</strong></div><small>{subscription.plan.moduleKeys.length?"Habilitados según el plan contratado":"Sin módulos comerciales registrados"}</small></div>
          </div>
        </div>
      </section>

      <section className="subscription-payment-card">
        <header><div><small>FACTURACIÓN</small><h3>Último pago</h3></div><span className="payment-icon"><Icon name="receipt" size={16}/></span></header>
        {payment?<div className="subscription-payment-detail">
          <div className="payment-amount"><small>MONTO</small><b>{payment.currency+" "+Number(payment.amount).toFixed(2)}</b></div>
          <span className={"payment-state "+payment.status}>{payment.status==="paid"?"Pagado":payment.status==="pending"?"Pendiente":payment.status==="failed"?"Fallido":"Reembolsado"}</span>
          <dl>
            <div><dt>Proveedor</dt><dd>{payment.provider}</dd></div>
            <div><dt>Fecha</dt><dd>{dateLabel(payment.paidAt??payment.createdAt,country,timeZone)}</dd></div>
            {payment.externalReference&&<div><dt>Referencia</dt><dd>{payment.externalReference}</dd></div>}
          </dl>
        </div>:<div className="subscription-payment-empty">
          <span><Icon name="receipt" size={22}/></span>
          <b>Sin pagos registrados</b>
          <small>Los pagos de la suscripción aparecerán aquí cuando se registren desde Plataforma.</small>
        </div>}
      </section>
    </div>

    <footer className="subscription-footer-note"><Icon name="lock" size={13}/><span>Los cambios de plan, estado y pagos los administra FUDIA desde Plataforma.</span></footer>
  </section>;
}

function UsageRow({icon,label,value,max}:{icon:"store"|"users";label:string;value:number;max:number|null}){
  const percent=usagePercent(value,max);
  return <div className="subscription-usage-row">
    <span className="usage-icon"><Icon name={icon} size={15}/></span>
    <div className="usage-copy">
      <div><b>{label}</b><strong>{value+" / "+(max??"∞")}</strong></div>
      <div className="usage-track" aria-label={label+" utilizado"}><i style={{width:percent+"%"}}/></div>
      <small>{max===null?"Sin límite definido":Math.max(0,max-value)+" disponibles"}</small>
    </div>
  </div>;
}

function ProfileSkeleton(){
  return <div className="profile-shell" aria-label="Cargando perfil" aria-busy="true">
    <section className="profile-card profile-skeleton profile-skeleton-account">
      <i/><div><span/><span/><span/></div>
    </section>
    <section className="profile-card profile-skeleton profile-skeleton-settings">
      <header><i/><span/></header>
      <div><section><i/><span/><span/></section><section><i/><span/><span/></section></div>
      <footer><i/></footer>
    </section>
  </div>;
}
