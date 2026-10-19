import {spawnSync} from "node:child_process";
import {appendFileSync} from "node:fs";
import {fileURLToPath} from "node:url";
import {resolve} from "node:path";
import {activeRelease, COMMIT_LABEL, COMPONENT_LABEL} from "./release-state.mjs";

export function changedComponents(paths) {
  const selected = {processor: false, backend: false, frontend: false};
  if (paths === null) return {processor: true, backend: true, frontend: true}; // First push of a branch.
  const workflows = {
    ".github/workflows/foods-image-processor-gcp.yml": "processor",
    ".github/workflows/foods-backend-gcp.yml": "backend",
    ".github/workflows/foods-admin-web-gcp.yml": "frontend",
  };
  for (const path of paths) {
    if (path.startsWith("foods-admin-web/")) selected.frontend = true;
    if (path.startsWith("foods-backend/")) selected.backend = true;
    if (path.startsWith("fudia-image-processor/")) selected.processor = true;
    if (workflows[path]) selected[workflows[path]] = true;
    if (path.startsWith("foods-infrastructure/") || path === ".github/workflows/foods-gcp.yml") {
      return {processor: true, backend: true, frontend: true}; // Shared deployment configuration/tests.
    }
  }
  return selected;
}

export function changedFiles(before, after, run = spawnSync) {
  if (!/^[a-f0-9]{40}$/.test(before ?? "") || !/^[a-f0-9]{40}$/.test(after ?? "")) {
    throw new Error("El push no contiene commits válidos para detectar cambios.");
  }
  if (before === "0".repeat(40)) return null;
  const result = run("git", ["diff", "--name-only", "--no-renames", "-z", before, after, "--"], {
    encoding: "utf8", maxBuffer: 16 * 1024 * 1024,
  });
  if (result.error || result.status !== 0) throw new Error("No se pudo comparar el push; no se seleccionarán despliegues a ciegas.");
  return result.stdout.split("\0").filter(Boolean);
}

export function detectionConfiguration(env) {
  const patterns = {
    GCP_PROJECT_ID: /^[a-z][a-z0-9-]{4,28}[a-z0-9]$/,
    GCP_REGION: /^[a-z]+-[a-z]+\d+$/,
    HEAD_SHA: /^[a-f0-9]{40}$/,
    GCP_IMAGE_PROCESSOR_SERVICE: /^[a-z][a-z0-9-]{0,61}[a-z0-9]$/,
    GCP_BACKEND_SERVICE: /^[a-z][a-z0-9-]{0,61}[a-z0-9]$/,
    GCP_FRONTEND_SERVICE: /^[a-z][a-z0-9-]{0,61}[a-z0-9]$/,
  };
  for (const [key, pattern] of Object.entries(patterns)) {
    if (!pattern.test(env[key] ?? "")) throw new Error(`Configura ${key} para detectar despliegues pendientes.`);
  }
  const services = {processor: env.GCP_IMAGE_PROCESSOR_SERVICE, backend: env.GCP_BACKEND_SERVICE, frontend: env.GCP_FRONTEND_SERVICE};
  if (new Set(Object.values(services)).size !== 3) throw new Error("Los tres servicios deben tener nombres distintos.");
  return {project: env.GCP_PROJECT_ID, region: env.GCP_REGION, head: env.HEAD_SHA, services};
}

function cloudJSON(args, run, allowMissing = false) {
  const result = run("gcloud", args, {encoding: "utf8", maxBuffer: 4 * 1024 * 1024, timeout: 60000});
  if (result.error || result.status !== 0) {
    // Require the requested service name, not a project/IAM error containing NOT_FOUND.
    const error = result.stderr ?? "";
    const missingService = error.includes(`Cannot find service [${args[3]}]`) ||
      (/\bNOT_FOUND\b/.test(error) && error.includes(`Service '${args[3]}'`));
    if (allowMissing && !result.error && missingService) return null;
    throw new Error("No se pudo consultar el estado de Cloud Run. Revisa autenticación, permisos y conectividad; no se omitirán componentes a ciegas.");
  }
  try {
    const value = JSON.parse(result.stdout);
    if (!value || typeof value !== "object" || Array.isArray(value)) throw new Error("Invalid object");
    return value;
  }
  catch { throw new Error("Cloud Run devolvió JSON inválido; no se puede determinar la versión desplegada."); }
}

export function pendingComponents(env, run = spawnSync) {
  const config = detectionConfiguration(env), selected = {}, details = [];
  const scope = ["--project", config.project, "--region", config.region, "--format=json", "--quiet"];
  for (const [component, name] of Object.entries(config.services)) {
    const service = cloudJSON(["run", "services", "describe", name, ...scope], run, true);
    const revision = service === null ? null : activeRelease(service);
    if (!revision) {
      selected[component] = true;
      details.push({component, service: name, reason: service === null ? "first-deployment" : "unverified-release"});
      continue;
    }
    const deployed = cloudJSON(["run", "revisions", "describe", revision, ...scope], run);
    if (deployed?.metadata?.name !== revision || !deployed.status?.conditions?.some(condition => condition.type === "Ready" && condition.status === "True")) {
      throw new Error(`La revisión activa de ${component} no tiene un estado Ready verificable.`);
    }
    const labels = deployed.metadata.labels ?? {}, sha = labels[COMMIT_LABEL];
    if (sha === undefined) {
      selected[component] = true;
      details.push({component, service: name, revision, reason: "missing-commit-label"});
      continue;
    }
    if (!/^[a-f0-9]{40}$/.test(sha) || labels[COMPONENT_LABEL] !== component) {
      throw new Error(`La revisión activa de ${component} tiene identidad de código inválida.`);
    }
    const ancestry = run("git", ["merge-base", "--is-ancestor", sha, config.head], {encoding: "utf8"});
    if (ancestry.error || ancestry.status !== 0) {
      throw new Error(`El commit desplegado de ${component} no es un ancestro disponible de HEAD; revisa el historial o una ejecución antigua antes de desplegar.`);
    }
    selected[component] = changedComponents(changedFiles(sha, config.head, run))[component];
    details.push({component, service: name, revision, deployedSha: sha, headSha: config.head, pending: selected[component]});
  }
  return {selected, details};
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    if (!process.env.GITHUB_OUTPUT) throw new Error("Falta GITHUB_OUTPUT para publicar los componentes modificados.");
    const {selected, details} = pendingComponents(process.env);
    appendFileSync(process.env.GITHUB_OUTPUT, Object.entries(selected).map(([key, value]) => `${key}=${value}\n`).join(""));
    console.log(JSON.stringify({selected, details}));
    if (process.env.GITHUB_STEP_SUMMARY) {
      appendFileSync(process.env.GITHUB_STEP_SUMMARY, "\n### Componentes pendientes frente a Cloud Run\n\n" +
        details.map(detail => `- ${detail.component}: ${selected[detail.component] ? "pendiente" : "sin cambios"}; revisión ${detail.revision ?? "sin verificar"}; commit ${detail.deployedSha ?? detail.reason}.`).join("\n") + "\n");
    }
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
