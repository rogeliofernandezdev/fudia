"use client";

export class ApiClientError extends Error {
  constructor(
    message: string,
    readonly code: string,
    readonly status: number,
  ) {
    super(message);
    this.name = "ApiClientError";
  }
}

let logoutInProgress = false;

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`/api/admin/${path}`, {
    ...init,
    headers: { "Content-Type": "application/json", ...init?.headers },
  });
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
    if (response.status === 401 && !logoutInProgress) {
      logoutInProgress = true;
      void fetch("/api/session", { method: "DELETE" })
        .catch(() => {})
        .finally(() => window.location.assign("/login"));
    }
    throw new ApiClientError(
      body.message ?? "No pudimos completar la operación.",
      body.code ?? "unknown_error",
      response.status,
    );
  }
  return body as T;
}
