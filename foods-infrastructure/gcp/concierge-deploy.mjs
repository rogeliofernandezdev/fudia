import {spawnSync} from "node:child_process";
import {appendFileSync} from "node:fs";
import {resolve} from "node:path";
import {fileURLToPath} from "node:url";
import {httpsURL, secretReferences} from "./deploy.mjs";
import {candidateRevision, promotionArgs} from "./release-state.mjs";

const resource = /^[a-z][a-z0-9-]{0,61}[a-z0-9]$/;
const secretKeys = ["REDIS_URL", "FUDIA_CONCIERGE_API_KEY", "WHATSAPP_TOKEN", "WHATSAPP_VERIFY_TOKEN", "WHATSAPP_APP_SECRET"];
export const CA_PATH = "/var/run/secrets/redis/server-ca.pem";

function required(env, key, pattern) {
  const value = env[key] ?? "";
  if (!value || (pattern && !pattern.test(value))) throw new Error(`Configura ${key} con un valor válido.`);
  return value;
}
function bounded(env, key, fallback, min, max) {
  const value = env[key] || String(fallback);
  if (!/^\d+$/.test(value) || +value < min || +value > max) throw new Error(`${key} fuera de rango.`);
  return value;
}

export function conciergeConfiguration(env) {
  const project = required(env, "GCP_PROJECT_ID", /^[a-z][a-z0-9-]{4,28}[a-z0-9]$/);
  const region = required(env, "GCP_REGION", /^[a-z]+-[a-z]+\d+$/);
  const repository = required(env, "GCP_ARTIFACT_REPOSITORY", /^[a-z][a-z0-9_-]{0,62}$/);
  const account = new RegExp(`^[a-z][a-z0-9-]+@${project}\\.iam\\.gserviceaccount\\.com$`);
  required(env, "GCP_WORKLOAD_IDENTITY_PROVIDER", /^projects\/\d+\/locations\/global\/workloadIdentityPools\/[a-z0-9-]+\/providers\/[a-z0-9-]+$/);
  const deployIdentity = required(env, "GCP_DEPLOY_SERVICE_ACCOUNT", account);
  const httpIdentity = required(env, "GCP_CONCIERGE_HTTP_SERVICE_ACCOUNT", account);
  const workerIdentity = required(env, "GCP_CONCIERGE_WORKER_SERVICE_ACCOUNT", account);
  const backendIdentity = required(env, "GCP_BACKEND_SERVICE_ACCOUNT", account);
  if (new Set([deployIdentity, httpIdentity, workerIdentity, backendIdentity]).size !== 4) {
    throw new Error("Deploy, HTTP, worker y backend requieren identidades separadas.");
  }
  const backend = required(env, "GCP_BACKEND_SERVICE", resource);
  const http = required(env, "GCP_CONCIERGE_HTTP_SERVICE", resource);
  const worker = required(env, "GCP_CONCIERGE_WORKER_POOL", resource);
  const job = required(env, "GCP_CONCIERGE_PREFLIGHT_JOB", resource);
  const names = [backend, http, worker, job, env.GCP_FRONTEND_SERVICE, env.GCP_IMAGE_PROCESSOR_SERVICE].filter(Boolean);
  if (new Set(names).size !== names.length) throw new Error("Concierge no puede reutilizar nombres de otros componentes.");
  const secrets = secretReferences(required(env, "GCP_CONCIERGE_SECRETS"), secretKeys);
  if (secretKeys.some(key => !secrets.has(key))) throw new Error("GCP_CONCIERGE_SECRETS debe incluir Redis, integración y los tres secretos Meta.");
  const backendSecrets = secretReferences(required(env, "GCP_BACKEND_SECRETS"), ["DATABASE_URL", "FUDIA_CONCIERGE_API_KEY"]);
  if (!backendSecrets.has("DATABASE_URL") || backendSecrets.get("FUDIA_CONCIERGE_API_KEY") !== secrets.get("FUDIA_CONCIERGE_API_KEY")) {
    throw new Error("GCP_BACKEND_SECRETS debe conservar DATABASE_URL y usar la misma referencia FUDIA_CONCIERGE_API_KEY que Concierge.");
  }
  const ca = secretReferences(`REDIS_CA=${required(env, "GCP_CONCIERGE_REDIS_CA_SECRET")}`, ["REDIS_CA"]).get("REDIS_CA");
  const sha = required(env, "GITHUB_SHA", /^[a-f0-9]{40}$/);
  const runID = required(env, "GITHUB_RUN_ID", /^\d+$/);
  const attempt = required(env, "GITHUB_RUN_ATTEMPT", /^\d+$/);
  const suffix = `r${sha.slice(0, 8)}-${runID}-${attempt}`;
  if (`${worker}-${suffix}`.length > 63) throw new Error("Nombre de worker/revisión demasiado largo.");
  return {
    project, region, http, worker, job, httpIdentity, workerIdentity, backend, backendIdentity,
    secrets, ca, sha, suffix,
    registry: `${region}-docker.pkg.dev`,
    image: `${region}-docker.pkg.dev/${project}/${repository}/fudia-concierge:${sha}-${runID}-${attempt}`,
    network: required(env, "GCP_CONCIERGE_NETWORK", resource),
    subnet: required(env, "GCP_CONCIERGE_SUBNET", resource),
    phone: required(env, "WHATSAPP_PHONE_ID", /^\d+$/),
    graph: required(env, "WHATSAPP_GRAPH_VERSION", /^v[1-9]\d*\.0$/),
    location: env.GOOGLE_CLOUD_LOCATION ? required(env, "GOOGLE_CLOUD_LOCATION", /^(global|[a-z]+-[a-z]+\d+)$/) : "global",
    lite: required({...env, GEMINI_MODEL_LITE: env.GEMINI_MODEL_LITE || "gemini-3.5-flash-lite"}, "GEMINI_MODEL_LITE", /^gemini-[a-z0-9.-]+$/),
    flash: required({...env, GEMINI_MODEL_FLASH: env.GEMINI_MODEL_FLASH || "gemini-3.5-flash"}, "GEMINI_MODEL_FLASH", /^gemini-[a-z0-9.-]+$/),
    instances: bounded(env, "GCP_CONCIERGE_WORKER_INSTANCES", 1, 1, 5),
    maxHTTP: bounded(env, "GCP_CONCIERGE_HTTP_MAX_INSTANCES", 1, 1, 5),
    concurrency: bounded(env, "QUEUE_WORKER_CONCURRENCY", 8, 1, 32),
    sessionTTL: bounded(env, "SESSION_TTL_SECONDS", 3600, 600, 86400),
  };
}

function command(binary, args, capture = false, missing = false) {
  const result = spawnSync(binary, args, {encoding: "utf8", stdio: capture ? "pipe" : "inherit"});
  if (result.error) throw new Error(`${binary} no está disponible.`);
  if (result.status !== 0) {
    const stderr = result.stderr ?? "";
    if (missing && !/permission|denied|forbidden|unauthorized|403/i.test(stderr) &&
      /NOT_FOUND|Cannot find service|Cannot find worker pool/i.test(stderr)) return null;
    // Never print provider response bodies or credential-bearing errors.
    throw new Error(`${binary} falló (${result.status}).`);
  }
  return capture ? result.stdout.trim() : "";
}

export function buildConcierge(config, run = command) {
  run("docker", ["build", "--platform", "linux/amd64", "-t", config.image, "./fudia-concierge"]);
}

export function runtimeArgs(c, apiURL, worker = false) {
  const refs = [...c.secrets].map(([key, value]) => `${key}=${value}`);
  refs.push(`${CA_PATH}=${c.ca}`);
  return ["--network", c.network, "--subnet", c.subnet, "--vpc-egress", "private-ranges-only",
    "--set-secrets", refs.join(","), "--set-env-vars", [
      "ENV=production", "LOG_LEVEL=INFO", `FUDIA_API_URL=${httpsURL(apiURL, "URL backend")}`,
      `WORKER_ENABLED=${worker}`, `REDIS_CA_CERTFILE=${CA_PATH}`,
      "LLM_PROVIDER=vertex", `GOOGLE_CLOUD_PROJECT=${c.project}`, `GOOGLE_CLOUD_LOCATION=${c.location}`,
      `GEMINI_MODEL_LITE=${c.lite}`, `GEMINI_MODEL_FLASH=${c.flash}`,
      `WHATSAPP_PHONE_ID=${c.phone}`, `WHATSAPP_GRAPH_VERSION=${c.graph}`,
      `SESSION_TTL_SECONDS=${c.sessionTTL}`, `QUEUE_WORKER_CONCURRENCY=${c.concurrency}`,
      "QUEUE_SHUTDOWN_GRACE_SECONDS=8",
    ].join(",")];
}

export function httpArgs(c, digest, apiURL, exists) {
  return ["run", "deploy", c.http, "--image", digest, "--service-account", c.httpIdentity,
    "--execution-environment", "gen2", "--port", "8080", "--cpu", "1", "--memory", "1Gi",
    "--min-instances", "0", "--max-instances", c.maxHTTP, "--concurrency", "40", "--timeout", "30s",
    "--ingress", "all", "--no-invoker-iam-check", "--tag", "candidate",
    "--update-labels", `fudia-commit=${c.sha},fudia-component=concierge`,
    "--startup-probe", "httpGet.path=/ready,httpGet.port=8080,periodSeconds=10,timeoutSeconds=3,failureThreshold=24",
    ...(exists ? ["--no-traffic"] : []), ...runtimeArgs(c, apiURL)];
}

export async function smokeConcierge(url) {
  for (const path of ["/live", "/ready"]) {
    const response = await fetch(`${httpsURL(url, "URL Concierge")}${path}`, {redirect: "error", signal: AbortSignal.timeout(30000)});
    const data = await response.json();
    if (response.status !== 200 || (path === "/live" ? data.service !== "fudia-concierge" : data.status !== "ready")) {
      throw new Error("Concierge no pasó las pruebas de salud; no se promueve tráfico.");
    }
  }
}

export async function deployConcierge(c, env, run = command, smoke = smokeConcierge) {
  if (env.GCP_CONCIERGE_DEPLOY_APPROVED !== "true") throw new Error("Falta confirmar explícitamente el despliegue y el costo del worker continuo.");
  const cloud = (args, capture = false, missing = false) => run("gcloud", [...args, "--project", c.project, "--region", c.region, "--quiet"], capture, missing);
  const describe = (kind, name, missing = false) => {
    const value = cloud(["run", kind, "describe", name, "--format=json"], true, missing);
    if (value === null) return null;
    try { return JSON.parse(value); }
    catch { throw new Error("Cloud Run no devolvió metadatos JSON válidos."); }
  };
  const backend = describe("services", c.backend);
  if (!backend?.status?.url) throw new Error("Despliega primero el backend GCP.");
  const apiURL = httpsURL(backend.status.url, "URL backend");
  // Validate the actual serving revision, not an unpromoted candidate/template.
  const active = backend.status.traffic?.filter(t => Number(t.percent ?? 0) > 0);
  if (active?.length !== 1 || active[0].percent !== 100 || !active[0].revisionName) throw new Error("El backend debe servir una única revisión al 100 %.");
  const revision = describe("revisions", active[0].revisionName);
  const containers = revision?.spec?.containers;
  const ref = containers?.[0]?.env?.find(entry => entry.name === "FUDIA_CONCIERGE_API_KEY")?.valueFrom?.secretKeyRef;
  const expected = c.secrets.get("FUDIA_CONCIERGE_API_KEY");
  if (containers?.length !== 1 || !ref || `${ref.name}:${ref.key}` !== expected || revision.spec.serviceAccountName !== c.backendIdentity) {
    throw new Error("La revisión activa del backend no usa la identidad/versión del secreto de integración configurados. Despliega backend primero.");
  }
  const previousHTTP = describe("services", c.http, true);
  const previousWorker = describe("worker-pools", c.worker, true);
  run("gcloud", ["auth", "configure-docker", c.registry, "--quiet"]);
  if (env.GCP_IMAGE_BUILT !== "true") buildConcierge(c, run);
  run("docker", ["push", c.image]);
  const digest = run("docker", ["image", "inspect", c.image, "--format", "{{index .RepoDigests 0}}"], true);
  if (!digest.startsWith(`${c.image.split(":")[0]}@sha256:`) || !/@sha256:[a-f0-9]{64}$/.test(digest)) throw new Error("Digest publicado inválido.");

  // Execute dependency checks under the worker's identity and private network.
  cloud(["run", "jobs", "deploy", c.job, "--image", digest, "--service-account", c.workerIdentity,
    "--command", "python", "--args=-m,src.preflight", "--tasks", "1", "--parallelism", "1",
    "--max-retries", "0", "--task-timeout", "300s", "--cpu", "1", "--memory", "1Gi", ...runtimeArgs(c, apiURL)]);
  cloud(["run", "jobs", "execute", c.job, "--wait"]);

  cloud(httpArgs(c, digest, apiURL, Boolean(previousHTTP)));
  const current = describe("services", c.http);
  const candidate = candidateRevision(current);
  await smoke(httpsURL(candidate.url, "Candidata Concierge"));

  cloud(["run", "worker-pools", "deploy", c.worker, "--image", digest, "--service-account", c.workerIdentity,
    "--revision-suffix", c.suffix, "--instances", c.instances,
    "--update-labels", `fudia-commit=${c.sha},fudia-component=concierge`,
    ...(previousWorker ? ["--no-promote"] : []),
    "--command", "python", "--args=-m,src.worker", "--cpu", "1", "--memory", "1Gi", ...runtimeArgs(c, apiURL, true)]);
  cloud(["run", "worker-pools", "update-instance-split", c.worker, "--to-revisions", `${c.worker}-${c.suffix}=100`]);
  cloud(promotionArgs(c.http, candidate.revisionName));
  if (env.GITHUB_STEP_SUMMARY) appendFileSync(env.GITHUB_STEP_SUMMARY,
    `\n### Concierge GCP\nHTTP: ${current.status.url}\nWebhook Meta: ${current.status.url}/webhook\nWorker: ${c.worker} (${c.instances} instancia/s continuas)\nImagen: ${digest}\nPreflight de dependencias: correcto. Pendiente prueba QR → pedido → KDS y estados Meta.\n`);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const config = conciergeConfiguration(process.env);
    if (process.argv[2] === "validate") console.log("Configuración Concierge validada; no se crearon recursos.");
    else if (process.argv[2] === "build") buildConcierge(config);
    else if (process.argv[2] === "deploy") await deployConcierge(config, process.env);
    else throw new Error("Usa validate, build o deploy; no hay despliegue implícito.");
  } catch (error) {
    console.error(error instanceof Error ? error.message : "Despliegue Concierge fallido.");
    process.exitCode = 1;
  }
}
