"use client";
import {EmptyState,PageHeader,type IconName} from "@/design-system";

export type ConfigurationComingSoonFeature="billing"|"integrations"|"operations";

const features:Record<ConfigurationComingSoonFeature,{icon:IconName;title:string;description:string;subject:string}>={
 billing:{icon:"receipt",title:"Facturación electrónica",description:"Series, comprobantes y proveedor SUNAT.",subject:"la facturación electrónica"},
 integrations:{icon:"plug",title:"Integraciones",description:"WhatsApp, pagos, impresión y notificaciones.",subject:"las integraciones"},
 operations:{icon:"layout",title:"Mesas, cajas y estaciones",description:"Estructura operativa de cada local.",subject:"mesas, cajas y estaciones"},
};

/** Configuración aún no disponible: misma cabecera y estado «Próximamente» en todas las secciones. */
export function ConfigurationComingSoon({feature}:{feature:ConfigurationComingSoonFeature}){
 const{icon,title,description,subject}=features[feature];
 return <><PageHeader eyebrow="CONFIGURACIÓN" title={title} description={description}/>
  <section className="panel management"><EmptyState icon={icon} title="Próximamente" text={`Estamos preparando la configuración de ${subject}.`}/></section>
 </>;
}
