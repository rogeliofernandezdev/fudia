import {cookies} from "next/headers";
import {LoginForm} from "@/modules/auth";

export default async function Login(){
  // Sin cookie no hay sesión que validar: el formulario aparece de inmediato.
  const hasSession=Boolean((await cookies()).get("foods_session")?.value);
  return <LoginForm hasSession={hasSession}/>;
}
