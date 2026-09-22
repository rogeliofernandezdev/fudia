"use client";
import {useRouter} from "next/navigation";
import {Button,Icon} from "@/design-system";

export default function NoAccessPage(){
  const router=useRouter();
  return <section className="admin-session-error">
    <Icon name="lock" size={24}/>
    <h1>Sin accesos asignados</h1>
    <p>Tu cuenta está activa, pero tu rol no tiene opciones habilitadas para este local. Solicita a un administrador que revise tus accesos.</p>
    <Button kind="secondary" icon="users" onClick={()=>router.push("/configuracion/perfil")}>Ver mi perfil</Button>
  </section>;
}
