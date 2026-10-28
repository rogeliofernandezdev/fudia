import {PageHeader} from "@/design-system/page-header";
import {EmptyState} from "@/design-system/states";
import {pageRoutes} from "@/shared/routing/page-routes";

export function ComingSoonPage({eyebrow,title,description}:{eyebrow:string;title:string;description:string}){
  return <><PageHeader title={title} description={description} back={{href:pageRoutes.home,label:"Volver al inicio"}}/>
  <section className="panel management"><EmptyState icon="settings" title="Próximamente" text="Este módulo está en desarrollo. Pronto estará disponible para tu empresa."/></section>
  </>;
}
