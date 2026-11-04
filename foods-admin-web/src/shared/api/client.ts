"use client";
import {expireBrowserSession} from "@/shared/session/expire-session";

export class ApiClientError extends Error {
  constructor(
    message: string,
    readonly code: string,
    readonly status: number,
    readonly correlationId?: string,
  ) {
    super(message);
    this.name = "ApiClientError";
  }
}

/**
 * El rol no tiene la acción que exige el endpoint (403). Reintentar no lo resuelve:
 * la pantalla muestra el mensaje del backend, que nombra el permiso que falta y dónde
 * se agrega, sin botón de reintento.
 */
export function isPermissionError(error:unknown):error is ApiClientError{
  return error instanceof ApiClientError&&(error.status===403||error.code==="forbidden");
}

/** Etiqueta del permiso que falta, tal como la nombra el mensaje del backend («Ver inventario»); "" si no la trae. */
export function missingPermissionLabel(error:unknown){
  if(!isPermissionError(error))return "";
  return /«([^»]+)»/.exec(error.message)?.[1]??"";
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  return authenticatedFetch<T>(`/api/admin/${path}`, init);
}

export async function authenticatedFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(path, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });
  if(response.status===401)expireBrowserSession();
  if (response.status === 204) return undefined as T;
  const raw = await response.text();
  let body: {message?: string; code?: string} & Record<string, unknown>;
  try {
    body = raw ? JSON.parse(raw) : {};
  } catch {
    throw new ApiClientError(
      response.ok
        ? "El servidor devolvió una respuesta inválida. Intenta nuevamente."
        : response.status === 404
          ? "El servicio solicitado aún no está disponible. Reinicia el backend e intenta nuevamente."
          : "No pudimos interpretar la respuesta del servidor.",
      "invalid_server_response",
      response.status,
    );
  }
  if (!response.ok) {
    throw new ApiClientError(
      body.message ?? "No pudimos completar la operación.",
      body.code ?? "unknown_error",
      response.status,
      typeof body.correlationId === "string" ? body.correlationId : undefined,
    );
  }
  return body as T;
}
