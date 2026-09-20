"use client";
import "./modules.css";
import {useMutation, useQuery, useQueryClient} from "@tanstack/react-query";
import {Icon, type IconName} from "@/design-system/icons";
import {PageHeader} from "@/design-system/page-header";
import {useFeedback} from "@/providers/feedback-provider";
import {useSession} from "@/providers/session-context";
import {listModules, toggleModule} from "../infrastructure/modules-api";
import type {Module} from "../domain/types";

export function ModulesView() {
  const {user, organization} = useSession();
  const {notify} = useFeedback();
  const client = useQueryClient();
  const modules = useQuery({queryKey: ["modules"], queryFn: listModules});
  const toggle = useMutation({
    mutationFn: toggleModule,
    onSuccess: () => {
      void client.invalidateQueries({queryKey: ["modules"]});
      void client.invalidateQueries({queryKey: ["session-context"]});
    },
    onError: (e: Error) => notify({tone: "danger", title: "No se pudo actualizar", message: e.message}),
  });

  function onToggle(m: Module) {
    toggle.mutate({key: m.key, active: !m.active});
    notify({
      tone: "success",
      title: m.active ? "Módulo desactivado" : "Módulo activado",
      message: `${m.name} ${m.active ? "ya no está disponible" : "ahora está disponible"} para tu empresa.`,
    });
  }

  const cats = modules.data?.modules ? Array.from(new Set(modules.data.modules.map(m => m.category))) : [];
  const list = modules.data?.modules ?? [];

  if (!user?.platformAdmin) {
    return (
      <>
        <PageHeader eyebrow="CONFIGURACIÓN" title="Módulos" description="Activa o desactiva los módulos disponibles para tu empresa." />
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
      <PageHeader eyebrow="CONFIGURACIÓN" title="Módulos" description="Activa o desactiva los módulos disponibles para tu empresa." />
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
                {list.filter(m => m.category === cat).map(m => (
                  <div key={m.key} className={`module-item${m.active ? " active" : ""}`}>
                    <span className="module-icon"><Icon name={m.icon as IconName} size={20} /></span>
                    <div className="module-info"><b>{m.name}</b><small>{m.description}</small></div>
                    <button className={`module-toggle${m.active ? " on" : ""}`} onClick={() => onToggle(m)} aria-pressed={m.active} aria-label={m.active ? `Desactivar ${m.name}` : `Activar ${m.name}`}>
                      <span className="toggle-track"><span className="toggle-thumb" /></span>
                    </button>
                  </div>
                ))}
              </div>
            </section>
          ))}
        </div>
      )}
    </>
  );
}
