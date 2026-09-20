import "@/styles/loading.css";
export default function Loading(){return <main className="route-skeleton" aria-label="Cargando contenido" aria-busy="true"><span/><section><i/><i/><i/><i/></section><article>{Array.from({length:6},(_,index)=><i key={index}/>)}</article></main>}
