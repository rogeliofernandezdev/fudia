import type { LoginCredentials } from "../domain/types";

export class ActiveSessionError extends Error {
  constructor(public readonly userName: string) {
    super("Ya existe una sesión activa en este navegador.");
    this.name = "ActiveSessionError";
  }
}

export async function login(credentials: LoginCredentials): Promise<void> {
  const response = await fetch("/api/session", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(credentials),
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({})) as {code?:string;message?:string;userName?:string};
    if(response.status===409&&body.code==="session_already_active")throw new ActiveSessionError(body.userName??"Usuario");
    throw new Error(body.message ?? "Correo o contraseña incorrectos.");
  }
}

export async function logout(): Promise<void> {
  await fetch("/api/session", { method: "DELETE" }).catch(() => {});
}
