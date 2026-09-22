"use client";
import "./modules.css";
import {useMutation, useQuery, useQueryClient} from "@tanstack/react-query";
import {Icon, type IconName} from "@/design-system/icons";
import {PageHeader} from "@/design-system/page-header";
import {useFeedback} from "@/providers/feedback-provider";
import {useSession} from "@/providers/session-context";
import {listModules, toggleModule} from "../infrastructure/modules-api";
import type {Module, ModuleAvailability} from "../domain/types";

const availabilityMeta:Record<ModuleAvailability,{label:string;className:string;description:string}>={
  ready:{label:"Disponible",className:"ready",description:"Puede habilitarse para la empresa."},
  development:{label:"En desarrollo",className:"development",description:"Visible para plataforma, todavía no activable para empresas."},
  planned:{label:"Planificado",className:"planned",description:"Incluido en el roadmap, todavía no activable para empresas."},
};

export function ModulesView() {
  const {user, organization} = useSession();
  const {notify} = useFeedback();
  const client = useQueryClient();
  const modules = useQuery({queryKey: ["modules"], queryFn: listModules});
  const toggle = useMutation({
    mutationFn: toggleModule,
    onSuccess: (_data, variables) => {
      void client.invalidateQueries({queryKey: ["modules"]});
      void client.invalidateQueries({queryKey: ["session-context"]});
      const module=modules.data?.modules.find(item=>item.key===variables.key);
      notify({
        tone: "success",
        title: variables.active ? "Módulo activado" : "Módulo desactivado",
        message: `${module?.name??"El módulo"} ${variables.active ? "ahora está disponible" : "ya no está disponible"} para esta empresa.`,
      });
    },
    onError: (e: Error) => notify({tone: "danger", title: "No se pudo actualizar", message: e.message}),
  });

  function onToggle(module: Module) {
    if(module.availability!=="ready")return;
    toggle.mutate({key: module.key, active: !module.active});
  }

  const cats = modules.data?.modules ? Array.from(new Set(modules.data.modules.map(m => m.category))) : [];
  const list = modules.data?.modules ?? [];

  if (!user?.platformAdmin) {
    return (
      <>
        <PageHeader eyebrow="CONFIGURACIÓN" title="Módulos" description="Consulta la disponibilidad de módulos de la plataforma." />
        <div className="panel management">
          <div className="catalog-state">
            <span><Icon name="lock" size={24} /></span>
            <b>Acceso restringido</b>
            <p>Solo el administrador de plataforma puede gestionar los módulos.</p>
          </div>
        </div>
      </>
    );
  }

  return (
    <>
      <PageHeader eyebrow="CONFIGURACIÓN" title="Módulos" description="Catálogo completo de FUDIA. Solo los módulos disponibles pueden habilitarse para una empresa." />
      {organization && <div className="modules-org-banner"><Icon name="store" size={16} /><span>Gestionando módulos de <b>{organization.name}</b> — los cambios aplican a todos sus locales y usuarios.</span></div>}
      {modules.isLoading ? (
        <div className="panel management"><div className="catalog-state"><b>Cargando módulos…</b></div></div>
      ) : modules.isError ? (
        <div className="panel management"><div className="catalog-state"><span><Icon name="alert" size={24} /></span><b>No pudimos cargar los módulos</b><p>Intenta nuevamente en unos segundos.</p></div></div>
      ) : (
        <div className="modules-grid">
          {cats.map(cat => (
            <section key={cat} className="panel module-category">
              <header><h2>{cat}</h2></header>
              <div className="module-list">
                {list.filter(m => m.category === cat).map(m => {
                  const availability=availabilityMeta[m.availability];
                  const activable=m.availability==="ready";
                  return <div key={m.key} className={`module-item${m.active ? " active" : ""}${!activable?" unavailable":""}`}>
                    <span className="module-icon"><Icon name={m.icon as IconName} size={20} /></span>
                    <div className="module-info">
                      <div className="module-title-row"><b>{m.name}</b><span className={`module-availability ${availability.className}`}>{availability.label}</span></div>
                      <small>{m.description}</small>
                      <em>{availability.description}</em>
                    </div>
                    <button
                      className={`module-toggle${m.active ? " on" : ""}`}
                      onClick={() => onToggle(m)}
                      aria-pressed={m.active}
                      aria-label={activable?(m.active ? `Desactivar ${m.name}` : `Activar ${m.name}`):`${m.name}: ${availability.label}`}
                      disabled={!activable||toggle.isPending}
                      title={!activable?availability.description:undefined}
                    >
                      <span className="toggle-track"><span className="toggle-thumb" /></span>
                    </button>
                  </div>;
                })}
              </div>
            </section>
          ))}
        </div>
      )}
    </>
  );
}
