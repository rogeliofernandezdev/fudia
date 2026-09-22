import "./styles/full-screen-loader.css";
import {Logo} from "./logo";

export function FullScreenLoader({label="Cargando FUDIA"}:{label?:string}){
  return <main className="full-screen-loader" role="status" aria-live="polite" aria-label={label} aria-busy="true">
    <div className="full-screen-loader-brand" aria-hidden="true">
      <div className="full-screen-loader-logo full-screen-loader-logo-base"><Logo/></div>
      <div className="full-screen-loader-logo full-screen-loader-logo-fill"><Logo/></div>
    </div>
    <span className="full-screen-loader-label">{label}</span>
  </main>;
}
