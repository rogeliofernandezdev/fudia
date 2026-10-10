"use client";
import "./modules.css";
import {useState} from "react";
import {useMutation, useQuery, useQueryClient} from "@tanstack/react-query";
import {ConfirmDialog, EmptyState, ErrorState, Icon, type IconName, PageHeader, RowActionButton, Skeleton, Status} from "@/design-system";
import {useFeedback} from "@/providers/feedback-provider";
import {useSession} from "@/providers/session-context";
import {listModules, toggleModule} from "../infrastructure/modules-api";
import type {Module, ModuleAvailability} from "../domain/types";

const availabilityMeta:Record<ModuleAvailability,{label:string;tone:"green"|"blue"|"gray";description:string}>={
  ready:{label:"Disponible",tone:"green",description:"Puede habilitarse para la empresa."},
  development:{label:"En desarrollo",tone:"blue",description:"Visible para plataforma, todavía no activable para empresas."},
  planned:{label:"Planificado",tone:"gray",description:"Incluido en el roadmap, todavía no activable para empresas."},
};

function ModulesSkeleton() {
  return <div className="modules-grid" role="status" aria-busy="true" aria-label="Cargando módulos">
    {Array.from({length: 2}, (_, section) => <section key={section} className="panel module-category" aria-hidden="true">
      <header><Skeleton width="var(--size-120)" height="var(--size-14)"/></header>
      <div className="module-list">
        {Array.from({length: 3}, (_, row) => <div key={row} className="module-item">
          <Skeleton className="module-skeleton-fixed" width="var(--size-40)" height="var(--size-40)" radius="var(--radius-10)"/>
          <div className="module-skeleton-info"><Skeleton width="var(--size-140)"/><Skeleton width="80%" height="var(--size-9)"/><Skeleton width="60%" height="var(--size-9)"/></div>
          <Skeleton className="module-skeleton-fixed" width="var(--control-height)" height="var(--control-height)"/>
        </div>)}
      </div>
    </section>)}
  </div>;
}

export function ModulesView() {
  const {user, organization} = useSession();
  const {notify} = useFeedback();
  const client = useQueryClient();
  const [confirm, setConfirm] = useState<Module|null>(null);
  const modules = useQuery({queryKey: ["modules"], queryFn: listModules});
  const toggle = useMutation({
    mutationFn: toggleModule,
    onSuccess: (_data, variables) => {
      setConfirm(null);
      void client.invalidateQueries({queryKey: ["modules"]});
      void client.invalidateQueries({queryKey: ["session-context"]});
      const currentModule=modules.data?.modules.find(item=>item.key===variables.key);
      notify({
        tone: "success",
        title: variables.active ? "Módulo activado" : "Módulo desactivado",
        message: `${currentModule?.name??"El módulo"} ${variables.active ? "ahora está disponible" : "ya no está disponible"} para esta empresa.`,
      });
    },
    onError: (e: Error) => {setConfirm(null);notify({tone: "danger", title: "No se pudo actualizar", message: e.message});},
  });
  const pendingKey = toggle.isPending ? toggle.variables?.key : undefined;

  function onToggle(module: Module) {
    if(module.availability!=="ready"||toggle.isPending)return;
    if(module.active){setConfirm(module);return;}
    toggle.mutate({key: module.key, active: true});
  }

  const cats = modules.data?.modules ? Array.from(new Set(modules.data.modules.map(m => m.category))) : [];
  const list = modules.data?.modules ?? [];

  if (!user?.platformAdmin) {
    return (
      <>
        <PageHeader eyebrow="CONFIGURACIÓN" title="Módulos" description="Consulta la disponibilidad de módulos de la plataforma." />
        <div className="panel management">
          <EmptyState icon="lock" title="Acceso restringido" text="Solo el administrador de plataforma puede gestionar los módulos."/>
        </div>
      </>
    );
  }

  return (
    <>
      <PageHeader eyebrow="CONFIGURACIÓN" title="Módulos" description="Catálogo completo de FUDIA. Solo los módulos disponibles pueden habilitarse para una empresa." />
      {organization && <div className="modules-org-banner"><Icon name="store" size={16} /><span>Gestionando módulos de <b>{organization.name}</b> — los cambios aplican a todos sus locales y usuarios.</span></div>}
      {modules.isLoading ? <ModulesSkeleton/> : modules.isError ? (
        <div className="panel management"><ErrorState title="No pudimos cargar los módulos" text={modules.error.message} onRetry={() => void modules.refetch()} retrying={modules.isFetching}/></div>
      ) : !list.length ? (
        <div className="panel management"><EmptyState icon="grid" title="Aún no hay módulos" text="El catálogo de la plataforma no tiene módulos registrados."/></div>
      ) : (
        <div className="modules-grid">
          {cats.map(cat => (
            <section key={cat} className="panel module-category">
              <header><h2>{cat}</h2></header>
              <div className="module-list">
                {list.filter(m => m.category === cat).map(m => {
                  const availability=availabilityMeta[m.availability];
                  const activable=m.availability==="ready";
                  const busy=pendingKey===m.key;
                  return <div key={m.key} className={`module-item${m.active ? " active" : ""}${!activable?" unavailable":""}`} aria-busy={busy}>
                    <span className="module-icon"><Icon name={m.icon as IconName} size={20} /></span>
                    <div className="module-info">
                      <div className="module-title-row"><b>{m.name}</b><Status tone={availability.tone}>{availability.label}</Status></div>
                      <small>{m.description}</small>
                      <em>{availability.description}</em>
                    </div>
                    <RowActionButton
                      action={m.active ? "deactivate" : "activate"}
                      label={activable ? undefined : availability.description}
                      stateLabel={activable ? `Estado de ${m.name}` : `${m.name}: ${availability.label}`}
                      onClick={() => onToggle(m)}
                      aria-busy={busy}
                      disabled={!activable||toggle.isPending}
                    />
                  </div>;
                })}
              </div>
            </section>
          ))}
        </div>
      )}
      <ConfirmDialog open={Boolean(confirm)} title="Desactivar módulo" subject={confirm?{label:"Módulo",name:confirm.name}:undefined} description={`Dejará de estar disponible para todos los locales y usuarios de ${organization?.name??"la empresa"}.`} confirmLabel="Desactivar" pending={toggle.isPending} onCancel={() => {if(!toggle.isPending)setConfirm(null)}} onConfirm={() => confirm&&toggle.mutate({key: confirm.key, active: false})}/>
    </>
  );
}
