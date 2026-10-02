import Link from "next/link";
import {Icon} from "@/design-system/icons";
import {PageHeader} from "@/design-system/page-header";

export function ComingSoonPage({eyebrow,title,description}:{eyebrow:string;title:string;description:string}){
  return <><PageHeader eyebrow={eyebrow} title={title} description={description}/>
  <Link href="/dashboard" className="settings-back"><Icon name="chevronLeft" size={16}/>Volver al inicio</Link>
  <section className="panel management"><div className="catalog-state"><span><Icon name="settings" size={24}/></span><b>Próximamente</b><p>Este módulo está en desarrollo. Pronto estará disponible para tu empresa.</p></div></section>
  </>;
}
