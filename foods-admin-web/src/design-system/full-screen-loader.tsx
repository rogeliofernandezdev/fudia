import "./styles/full-screen-loader.css";
import {Logo} from "./logo";

/** Loader FUDIA: el logo hierve como una olla al fuego (líquido agitado, burbujas y vapor). */
export function FullScreenLoader({label="Cargando FUDIA"}:{label?:string}){
  return <main className="full-screen-loader" role="status" aria-live="polite" aria-label={label} aria-busy="true">
    <div className="full-screen-loader-brand" aria-hidden="true">
      <span className="full-screen-loader-steam"><b/><b/><b/></span>
      <div className="full-screen-loader-logo full-screen-loader-logo-base"><Logo/></div>
      <div className="full-screen-loader-logo full-screen-loader-logo-wave"><Logo/></div>
      <div className="full-screen-loader-logo full-screen-loader-logo-fill"><Logo/></div>
      <span className="full-screen-loader-bubbles">{Array.from({length:9},(_,index)=><i key={index}/>)}</span>
    </div>
    <span className="full-screen-loader-label">{label}</span>
  </main>;
}
