import {redirect} from "next/navigation";

// El enlace privado de la central dejó de existir (2026-11-15): la central entra con su cuenta.
export default function RetiredPartnerLinkPage(){
  redirect("/delivery-panel/login");
}
