import "./platform-skeletons.css";

export function PlatformOnboardingSkeleton({stepCount}:{stepCount:number}){
  return <div className="onboarding-wizard platform-onboarding-skeleton" role="status" aria-label="Cargando configuración de empresa" aria-busy="true">
    <div className="wizard-steps" aria-hidden="true">
      {Array.from({length:stepCount},(_,index)=><div className="wizard-step" key={index}>
        <span className="platform-skeleton-block onboarding-skeleton-icon"/>
        <i className="platform-skeleton-block onboarding-skeleton-step-copy"/>
      </div>)}
    </div>
    <div className="onboarding-form" aria-hidden="true">
      <section className="panel management">
        <header><span className="platform-skeleton-block onboarding-skeleton-icon"/><div className="onboarding-skeleton-heading"><i className="platform-skeleton-block"/><i className="platform-skeleton-block"/></div></header>
        <div className="form-grid">
          {Array.from({length:4},(_,index)=><div className={"onboarding-skeleton-field"+(index===0||index===3?" span-2":"")} key={index}><i className="platform-skeleton-block onboarding-skeleton-label"/><i className="platform-skeleton-block onboarding-skeleton-control"/></div>)}
        </div>
      </section>
      <div className="wizard-footer"><div/><div><i className="platform-skeleton-block onboarding-skeleton-action"/></div></div>
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
