import "./platform-skeletons.css";

export function PlatformPlansSkeleton(){
  return <div className="plan-grid" aria-label="Cargando planes" aria-busy="true">
    {Array.from({length:3},(_,index)=><article className="panel plan-card skeleton-card" key={index}>
      <div className="skeleton-line wide"/>
      <div className="skeleton-line title"/>
      <div className="skeleton-line"/>
      <div className="skeleton-price"/>
      <div className="skeleton-row"><i/><i/><i/></div>
      {Array.from({length:5},(_,item)=><div className="skeleton-feature" key={item}><i/><span/></div>)}
    </article>)}
  </div>;
}

export function PlatformSubscriptionSkeleton(){
  return <>
    <div className="platform-subscription-grid" aria-label="Cargando suscripción" aria-busy="true">
      {Array.from({length:2},(_,index)=><section className="panel subscription-editor skeleton-card" key={index}>
        <div className="skeleton-line title"/>
        {Array.from({length:5},(_,field)=><div className="skeleton-field" key={field}><i/><span/></div>)}
      </section>)}
    </div>
    <section className="panel skeleton-history">
      {Array.from({length:4},(_,row)=><div className="skeleton-feature" key={row}><i/><span/></div>)}
    </section>
  </>;
}
