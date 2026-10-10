import {forwardRef,type InputHTMLAttributes} from "react";
import {Icon} from "./icons";
import {Input} from "./page-header";

type SearchFieldProps=Omit<InputHTMLAttributes<HTMLInputElement>,"type"|"aria-label">&{
  /** Nombre accesible; por defecto usa el placeholder. */
  label?:string;
};

/**
 * Buscador de barra de filtros: contenedor con borde, icono de lupa y campo sin
 * borde propio. Misma altura (`--control-height`) que Select y Button.
 */
export const SearchField=forwardRef<HTMLInputElement,SearchFieldProps>(function SearchField({label,placeholder="Buscar…",className="",...props},ref){
  return <label className={`ds-input-shell ds-search-field ${className}`.trim()}>
    <Icon name="search" size={18}/>
    <Input ref={ref} type="search" placeholder={placeholder} aria-label={label??placeholder} {...props}/>
  </label>;
});
