import "./platform-skeletons.css";

export function PlatformOnboardingSkeleton({stepCount}:{stepCount:number}){
  return <div className="onb-layout onb-skeleton" role="status" aria-label="Cargando configuración de empresa" aria-busy="true">
    <nav className="onb-steps" aria-hidden="true"><ol>
      {Array.from({length:stepCount},(_,index)=><li key={index}><div className="onb-step">
        <span className="platform-skeleton-block onb-step-icon"/>
        <span className="onb-step-copy"><i className="platform-skeleton-block onb-sk-title"/><i className="platform-skeleton-block onb-sk-sub"/></span>
      </div></li>)}
    </ol></nav>
    <div className="onb-card" aria-hidden="true">
      <header className="onb-card-head"><span className="platform-skeleton-block onb-card-icon"/><i className="platform-skeleton-block onb-sk-heading"/></header>
      <div className="onb-card-body"><div className="onb-grid">
        {Array.from({length:4},(_,index)=><div className={"onb-field"+(index===0||index===3?" wide":"")} key={index}><i className="platform-skeleton-block onb-sk-label"/><i className="platform-skeleton-block onb-sk-control"/></div>)}
      </div></div>
      <footer className="onb-footer"><span/><i className="platform-skeleton-block onb-sk-action"/></footer>
    </div>
  </div>;
}

export function PlatformPlansSkeleton(){
  return <div className="plan-grid" aria-label="Cargando planes" aria-busy="true">
    {Array.from({length:3},(_,index)=><article className="panel plan-card platform-plan-skeleton" key={index}>
      <header><div className="plan-heading"><i className="platform-skeleton-block plan-icon-placeholder"/><div><i className="platform-skeleton-block eyebrow"/><i className="platform-skeleton-block heading"/></div></div><i className="platform-skeleton-block status"/></header>
      <div className="plan-description platform-skeleton-copy"><i/><i/></div>
      <div className="plan-price"><i className="platform-skeleton-block eyebrow"/><i className="platform-skeleton-block price"/><i className="platform-skeleton-block caption"/></div>
      <div className="plan-capacity">{Array.from({length:3},(_,item)=><div key={item}><i className="platform-skeleton-block capacity-icon"/><span><i className="platform-skeleton-block eyebrow"/><i className="platform-skeleton-block value"/></span></div>)}</div>
      <section className="plan-includes"><header><div><i className="platform-skeleton-block eyebrow"/><i className="platform-skeleton-block section-title"/></div><i className="platform-skeleton-block counter"/></header><div className="plan-feature-list">{Array.from({length:5},(_,item)=><div key={item}><i className="platform-skeleton-block feature-icon"/><span><i className="platform-skeleton-block feature-title"/><i className="platform-skeleton-block feature-copy"/></span></div>)}</div></section>
      <footer><div><i className="platform-skeleton-block eyebrow"/><i className="platform-skeleton-block footer-copy"/></div><i className="platform-skeleton-block footer-action"/></footer>
    </article>)}
  </div>;
}

export function PlatformSubscriptionSkeleton(){
  return <>
    <div className="platform-subscription-grid" aria-label="Cargando suscripción" aria-busy="true">
      {Array.from({length:2},(_,index)=><section className="panel subscription-editor platform-subscription-card-skeleton" key={index}>
        <header><div><i className="platform-skeleton-block eyebrow"/><i className="platform-skeleton-block heading"/></div><i className="platform-skeleton-block header-action"/></header>
        <div className="form-grid platform-subscription-skeleton-fields">{Array.from({length:6},(_,field)=><div className={field===0&&index===0?"span-2":""} key={field}><i className="platform-skeleton-block label"/><i className="platform-skeleton-block control"/></div>)}</div>
        {index===0&&<div className="subscription-plan-preview platform-subscription-preview-skeleton">{Array.from({length:5},(_,item)=><div key={item}><i className="platform-skeleton-block eyebrow"/><i className="platform-skeleton-block value"/></div>)}</div>}
        <footer><i className="platform-skeleton-block button"/></footer>
      </section>)}
    </div>
    <section className="panel subscription-history platform-subscription-history-skeleton" aria-label="Cargando historial de pagos" aria-busy="true">
      <header><div><i className="platform-skeleton-block eyebrow"/><i className="platform-skeleton-block heading"/></div><i className="platform-skeleton-block counter"/></header>
      <div className="table-wrap"><table><thead><tr><th>FECHA</th><th>MONTO</th><th>ESTADO</th><th>PROVEEDOR</th><th>REFERENCIA</th></tr></thead><tbody>{Array.from({length:4},(_,row)=><tr key={row}>{Array.from({length:5},(_,cell)=><td key={cell}><i className="platform-skeleton-block table-line"/></td>)}</tr>)}</tbody></table></div>
    </section>
  </>;
}
