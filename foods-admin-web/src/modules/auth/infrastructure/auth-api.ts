import type { LoginCredentials } from "../domain/types";

export async function login(credentials: LoginCredentials): Promise<void> {
  const response = await fetch("/api/session", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(credentials),
  });
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.message ?? "Correo o contraseña incorrectos.");
  }
}

export async function logout(): Promise<void> {
  await fetch("/api/session", { method: "DELETE" }).catch(() => {});
}
