"use client";
import {useRouter} from "next/navigation";
import {Button} from "@/design-system";
import {pageRoutes} from "@/shared/routing/page-routes";
import {AdminSessionError} from "@/shell/admin-session-error";

export default function NoAccessPage(){
  const router=useRouter();
  return <AdminSessionError icon="lock" title="Sin accesos asignados" description="Tu cuenta está activa, pero tu rol no tiene opciones habilitadas para este local. Solicita a un administrador que revise tus accesos."><Button kind="secondary" icon="users" onClick={()=>router.push(pageRoutes.profileSettings)}>Ver mi perfil</Button></AdminSessionError>;
}
