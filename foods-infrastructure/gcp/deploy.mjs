import {spawnSync} from "node:child_process";
import {appendFileSync} from "node:fs";
import {fileURLToPath} from "node:url";
import {resolve} from "node:path";

const resourceName = /^[a-z][a-z0-9-]{0,61}[a-z0-9]$/;
const secretEntry = /^([A-Z][A-Z0-9_]*)=([a-zA-Z0-9_-]+):([1-9][0-9]*)$/;

function required(env, name, pattern) {
  const value = env[name] ?? "";
  if (!value || (pattern && !pattern.test(value))) throw new Error(`Configura ${name} con un valor válido.`);
  return value;
}

function integer(env, name, fallback, minimum, maximum) {
  const value = env[name] || String(fallback);
  if (!/^\d+$/.test(value) || Number(value) < minimum || Number(value) > maximum) {
    throw new Error(`${name} debe estar entre ${minimum} y ${maximum}.`);
  }
  return value;
}

export function httpsURL(value, name) {
  const url = new URL(value);
  if (url.protocol !== "https:" || url.username || url.password || url.search || url.hash || url.pathname !== "/" || url.port || !/^[a-z0-9.-]+$/i.test(url.hostname)) {
    throw new Error(`${name} debe ser un origen HTTPS sin credenciales, path ni parámetros.`);
  }
  return url.origin;
}

export function secretReferences(value, allowedKeys) {
  if (!value) return new Map();
  const entries = value.split(",").map(entry => entry.trim());
  const result = new Map();
  for (const entry of entries) {
    const match = secretEntry.exec(entry);
    if (!match || !allowedKeys.includes(match[1]) || result.has(match[1])) {
      throw new Error("Referencias de Secret Manager inválidas: usa VARIABLE=nombre:versión numérica, sin duplicados.");
    }
    result.set(match[1], `${match[2]}:${match[3]}`);
  }
  return result;
}

export function configuration(target, env) {
  if (!["backend", "frontend"].includes(target)) throw new Error("Destino inválido: backend o frontend.");
  const project = required(env, "GCP_PROJECT_ID", /^[a-z][a-z0-9-]{4,28}[a-z0-9]$/);
  const region = required(env, "GCP_REGION", /^[a-z]+-[a-z]+\d+$/);
  const repository = required(env, "GCP_ARTIFACT_REPOSITORY", /^[a-z][a-z0-9_-]{0,62}$/);
  required(env, "GCP_WORKLOAD_IDENTITY_PROVIDER", /^projects\/\d+\/locations\/global\/workloadIdentityPools\/[a-z0-9-]+\/providers\/[a-z0-9-]+$/);
  const accountPattern = new RegExp(`^[a-z][a-z0-9-]+@${project}\\.iam\\.gserviceaccount\\.com$`);
  required(env, "GCP_DEPLOY_SERVICE_ACCOUNT", accountPattern);
  const backendService = required(env, "GCP_BACKEND_SERVICE", resourceName);
  const sha = required(env, "GITHUB_SHA", /^[a-f0-9]{40}$/);
  const runID = required(env, "GITHUB_RUN_ID", /^\d+$/);
  const attempt = required(env, "GITHUB_RUN_ATTEMPT", /^\d+$/);
  const config = {
    target, project, region, backendService, sha,
    registry: `${region}-docker.pkg.dev`,
    image: `${region}-docker.pkg.dev/${project}/${repository}/foods-${target === "backend" ? "backend" : "admin-web"}:${sha}-${runID}-${attempt}`,
    service: target === "backend" ? backendService : required(env, "GCP_FRONTEND_SERVICE", resourceName),
    identity: required(env, target === "backend" ? "GCP_BACKEND_SERVICE_ACCOUNT" : "GCP_FRONTEND_SERVICE_ACCOUNT", accountPattern),
    maxInstances: integer(env, target === "backend" ? "GCP_BACKEND_MAX_INSTANCES" : "GCP_FRONTEND_MAX_INSTANCES", 3, 1, 20),
    concurrency: integer(env, target === "backend" ? "GCP_BACKEND_CONCURRENCY" : "GCP_FRONTEND_CONCURRENCY", 40, 1, 80),
  };
  if (target === "backend") {
    if (env.GCP_DATABASE_READY !== "true") throw new Error("Confirma primero Cloud SQL restaurado, datos verificados y backup; no se ejecutará ninguna migración.");
    if (env.GCP_CLOUD_SQL_INSTANCE) {
      if (env.GCP_VPC_CONNECTOR) throw new Error("Usa GCP_CLOUD_SQL_INSTANCE sin GCP_VPC_CONNECTOR para conectar sin VPC.");
      config.cloudSQLInstance = required(env, "GCP_CLOUD_SQL_INSTANCE",
        new RegExp(`^${project}:${region}:[a-z][a-z0-9-]{0,96}[a-z0-9]$`));
    } else {
      // Preserve the explicit private-network option for existing GCP environments.
      config.connector = required(env, "GCP_VPC_CONNECTOR", resourceName);
    }
    config.bucket = required(env, "GCP_UPLOADS_BUCKET", /^[a-z0-9][a-z0-9-]{1,61}[a-z0-9]$/);
    config.processorService = required(env,"GCP_IMAGE_PROCESSOR_SERVICE",resourceName);
    if(config.processorService===backendService)throw new Error("El procesador debe ser un servicio independiente.");
    config.job = required(env, "GCP_MIGRATION_JOB", resourceName);
    config.migrationIdentity = required(env, "GCP_MIGRATION_SERVICE_ACCOUNT", accountPattern);
    config.maxConnections = integer(env, "DATABASE_MAX_CONNS", 12, 1, 100);
    config.secrets = secretReferences(required(env, "GCP_BACKEND_SECRETS"), ["DATABASE_URL", "FUDIA_CONCIERGE_API_KEY"]);
    if (!config.secrets.has("DATABASE_URL")) throw new Error("GCP_BACKEND_SECRETS debe referenciar DATABASE_URL.");
    config.migrationSecret = secretReferences(`DATABASE_URL=${required(env, "GCP_MIGRATION_DATABASE_SECRET")}`, ["DATABASE_URL"]).get("DATABASE_URL");
  } else {
    if (config.service === backendService) throw new Error("Backend y frontend deben usar servicios distintos.");
    config.apiURL = env.FOODS_API_URL ? httpsURL(env.FOODS_API_URL, "FOODS_API_URL") : "";
    config.conciergeURL = env.FUDIA_CONCIERGE_URL ? httpsURL(env.FUDIA_CONCIERGE_URL, "FUDIA_CONCIERGE_URL") : "";
    config.mapProvider = env.NEXT_PUBLIC_MAP_PROVIDER || "mapbox";
    if (config.mapProvider !== "mapbox") throw new Error("El frontend actual solo implementa mapbox.");
    config.mapToken = env.NEXT_PUBLIC_MAPBOX_ACCESS_TOKEN || "";
    if (config.mapToken && !config.mapToken.startsWith("pk.")) throw new Error("NEXT_PUBLIC_MAPBOX_ACCESS_TOKEN debe ser un token público pk.; nunca publicar un token secreto sk.");
    config.secrets = secretReferences(env.GCP_FRONTEND_SECRETS, ["MAPBOX_ACCESS_TOKEN"]);
    if (config.mapToken && !config.secrets.has("MAPBOX_ACCESS_TOKEN")) {
      throw new Error("Mapbox requiere MAPBOX_ACCESS_TOKEN en GCP_FRONTEND_SECRETS para geocodificación.");
    }
  }
  return config;
}

function command(binary, args, capture = false, tolerateMissing = false) {
  // No shell: configuration cannot inject executable fragments. Never print build tokens.
  const result = spawnSync(binary, args, {encoding: "utf8", stdio: capture ? "pipe" : "inherit"});
  if (result.error) throw result.error;
  if (result.status !== 0) {
    if (tolerateMissing && /NOT_FOUND|Cannot find service/i.test(result.stderr ?? "")) return null;
    throw new Error(`${binary} falló (${result.status}).${capture ? ` ${result.stderr}` : ""}`);
  }
  return capture ? result.stdout.trim() : "";
}

export function databaseArgs(config, job = false) {
  if (config.cloudSQLInstance) {
    const args = ["--set-cloudsql-instances", config.cloudSQLInstance];
    // jobs deploy does not support --clear-vpc-connector; clear it via jobs update.
    if (!job) args.push("--clear-vpc-connector", "--clear-network");
    return args;
  }
  return ["--vpc-connector", config.connector, "--vpc-egress", "private-ranges-only"];
}

export function migrationArgs(config, digest) {
  return ["run", "jobs", "deploy", config.job, "--image", digest, "--command", "/app/foods-migrate",
    "--service-account", config.migrationIdentity, ...databaseArgs(config, true),
    "--set-secrets", `DATABASE_URL=${config.migrationSecret}`,
    "--set-env-vars", "DATABASE_MAX_CONNS=2", "--tasks", "1", "--parallelism", "1",
    "--max-retries", "0", "--task-timeout", "120s", "--cpu", "1", "--memory", "512Mi"];
}

export function runMigrations(config, digest, cloud) {
  cloud(migrationArgs(config, digest));
  if (config.cloudSQLInstance) {
    cloud(["run", "jobs", "update", config.job, "--clear-vpc-connector", "--clear-network"]);
  }
  cloud(["run", "jobs", "execute", config.job, "--wait"]);
}

export function serviceArgs(config, digest, exists, uid = "100", gid = "101") {
  const args = ["run", "deploy", config.service, "--image", digest, "--service-account", config.identity,
    "--execution-environment", "gen2", "--port", "8080", "--cpu", "1", "--memory", "1Gi",
    "--min-instances", "0", "--max-instances", config.maxInstances, "--concurrency", config.concurrency,
    "--timeout", "60s", "--ingress", "all", "--no-invoker-iam-check", "--tag", "candidate",
    "--startup-probe", `httpGet.path=${config.target === "backend" ? "/health" : "/login"},httpGet.port=8080,periodSeconds=10,timeoutSeconds=3,failureThreshold=24`];
  if (exists) args.push("--no-traffic");
  const references = [...config.secrets].map(([key, ref]) => `${key}=${ref}`).join(",");
  args.push(references ? "--set-secrets" : "--clear-secrets");
  if (references) args.push(references);
  if (config.target === "backend") {
    if (!/^\d+$/.test(uid) || !/^\d+$/.test(gid) || uid === "0") throw new Error("La imagen backend debe usar un usuario no-root.");
    args.push(...databaseArgs(config),
      "--set-env-vars", `HTTP_ADDR=:8080,FOODS_COOKIE_SECURE=true,DATABASE_MAX_CONNS=${config.maxConnections},PRODUCT_IMAGE_STORAGE=gcs,PRODUCT_IMAGES_BUCKET=${config.bucket},PRODUCT_IMAGES_SIGNER=${config.identity}`,
      "--remove-volume-mount", "/app/uploads/products", "--remove-volume", "product-images");
  } else {
    args.push("--set-env-vars", `FOODS_API_URL=${httpsURL(config.apiURL, "FOODS_API_URL")}${config.conciergeURL ? `,FUDIA_CONCIERGE_URL=${config.conciergeURL}` : ""}`);
  }
  return args;
}

async function smoke(url, target, version) {
  for (let attempt = 0; attempt < 5; attempt++) {
    try {
      const response = await fetch(`${url}${target === "backend" ? "/health" : "/login"}`, {redirect: "error", signal: AbortSignal.timeout(15000)});
      if (response.status === 200) {
        if (target === "frontend") return;
        const body = await response.json();
        if (body.status === "ok" && body.service === "foods-backend" && body.version === version) return;
      }
    } catch { /* Retry read-only smoke checks; never retry writes or migrations. */ }
    await new Promise(resolve => setTimeout(resolve, 2000));
  }
  throw new Error("La revisión candidata no pasó la prueba HTTP; no se promueve tráfico en servicios existentes.");
}

export function buildImage(config, run = command) {
  const args = ["build", "--platform", "linux/amd64", "-t", config.image];
  if (config.target === "backend") args.push("--build-arg", `VERSION=${config.sha.slice(0, 8)}`);
  else args.push("--build-arg", `NEXT_PUBLIC_MAP_PROVIDER=${config.mapProvider}`, "--build-arg", `NEXT_PUBLIC_MAPBOX_ACCESS_TOKEN=${config.mapToken}`);
  args.push(config.target === "backend" ? "./foods-backend" : "./foods-admin-web");
  run("docker", args);
}

export async function deploy(config, env, run = command, check = smoke) {
  const scope = ["--project", config.project, "--region", config.region, "--quiet"];
  const gcloud = (args, capture = false, missing = false) => run("gcloud", [...args, ...scope], capture, missing);
  const describe = service => gcloud(["run", "services", "describe", service, "--format=json"], true, true);
  // Resolve BFF destination before publishing an image. A missing API blocks frontend deployment.
  if (config.target === "frontend" && !config.apiURL) {
    const backend = describe(config.backendService);
    if (!backend) throw new Error("Despliega el backend primero o configura FOODS_API_URL.");
    config.apiURL = httpsURL(JSON.parse(backend).status.url, "FOODS_API_URL");
  }
  const previous = describe(config.service);
  if(config.target==="backend"){
    const processor=describe(config.processorService);
    if(!processor||!JSON.parse(processor).status?.latestReadyRevisionName)throw new Error("Despliega primero fudia-image-processor y su trigger Eventarc.");
  }
  run("gcloud", ["auth", "configure-docker", config.registry, "--quiet"]);
  if (env.GCP_IMAGE_BUILT !== "true") buildImage(config, run);
  run("docker", ["push", config.image]);
  const digest = run("docker", ["image", "inspect", config.image, "--format", "{{index .RepoDigests 0}}"], true);
  if (!digest.startsWith(`${config.image.split(":")[0]}@sha256:`) || !/@sha256:[a-f0-9]{64}$/.test(digest)) {
    throw new Error("No se pudo verificar el digest publicado de la imagen.");
  }
  let uid, gid;
  if (config.target === "backend") {
    uid = run("docker", ["run", "--rm", "--entrypoint", "id", config.image, "-u"], true);
    gid = run("docker", ["run", "--rm", "--entrypoint", "id", config.image, "-g"], true);
  }
  const deploymentArgs = serviceArgs(config, digest, Boolean(previous), uid, gid);
  if (config.target === "backend") {
    runMigrations(config, digest, gcloud);
  }
  gcloud(deploymentArgs);
  const current = JSON.parse(describe(config.service));
  const candidate = current.status.traffic.find(traffic => traffic.tag === "candidate");
  if (!candidate?.url || !candidate.revisionName || candidate.revisionName !== current.status.latestReadyRevisionName) {
    throw new Error("No se pudo verificar la revisión candidata; no se promueve tráfico.");
  }
  await check(httpsURL(candidate.url, "URL candidata"), config.target, config.sha.slice(0, 8));
  gcloud(["run", "services", "update-traffic", config.service, "--to-revisions", `${candidate.revisionName}=100`]);
  if (env.GITHUB_STEP_SUMMARY) appendFileSync(env.GITHUB_STEP_SUMMARY,
    `\n### Cloud Run · ${config.target}\n\nServicio: ${current.status.url}\n\nRevisión: \`${candidate.revisionName}\`\n\nImagen: \`${digest}\`\n`);
}

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    const config = configuration(process.argv[2], process.env);
    if (process.argv[3] === "validate") console.log("Configuración GCP validada; no se crearon recursos.");
    else if (process.argv[3] === "build") buildImage(config);
    else if (process.argv[3]) throw new Error("Argumento desconocido.");
    else await deploy(config, process.env);
  } catch (error) {
    console.error(error instanceof Error ? error.message : "Despliegue fallido.");
    process.exitCode = 1;
  }
}
