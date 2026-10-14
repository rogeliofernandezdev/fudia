import {fileURLToPath} from "node:url";
import {resolve} from "node:path";

// Diagnostic metadata only: never use this decoding to authorize requests.
// The token is neither printed nor persisted; Google verifies it in the auth step.
export function selectedClaims(token) {
  try {
    if (typeof token !== "string" || token.split(".").length !== 3) throw new Error();
    const payload = JSON.parse(Buffer.from(token.split(".")[1], "base64url").toString("utf8"));
    const selected = {};
    for (const key of ["sub", "repository", "ref", "environment"]) {
      selected[key] = typeof payload?.[key] === "string" ? payload[key] : null;
    }
    if (!selected.sub || !selected.repository || !selected.ref) throw new Error();
    return selected;
  } catch {
    throw new Error("No se pudieron leer los campos de diagnóstico OIDC.");
  }
}

export async function oidcDiagnostics(env, request = fetch) {
  const endpoint = env.ACTIONS_ID_TOKEN_REQUEST_URL;
  const credential = env.ACTIONS_ID_TOKEN_REQUEST_TOKEN;
  if (!endpoint || !credential) throw new Error("Diagnóstico OIDC disponible solo dentro del job con id-token: write.");
  let token;
  try {
    const response = await request(endpoint, {
      headers: {Authorization: `Bearer ${credential}`},
      signal: AbortSignal.timeout(10000),
    });
    if (!response.ok) throw new Error();
    token = (await response.json()).value;
  } catch {
    // Do not log HTTP bodies, headers, credentials or exception details.
    throw new Error("No se pudo solicitar el token para el diagnóstico OIDC.");
  }
  return selectedClaims(token);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    console.log(JSON.stringify(await oidcDiagnostics(process.env), null, 2));
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
