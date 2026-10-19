import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {buildImage, configuration, deploy, httpsURL, migrationArgs, serviceArgs} from "../gcp/deploy.mjs";

const env = {
  GCP_DATABASE_READY: "true",
  GCP_PROJECT_ID: "foods-example", GCP_REGION: "southamerica-west1", GCP_ARTIFACT_REPOSITORY: "foods",
  GCP_WORKLOAD_IDENTITY_PROVIDER: "projects/123456789/locations/global/workloadIdentityPools/github/providers/github",
  GCP_DEPLOY_SERVICE_ACCOUNT: "github-deploy@foods-example.iam.gserviceaccount.com",
  GCP_BACKEND_SERVICE: "foods-backend-dev", GCP_FRONTEND_SERVICE: "foods-admin-web-dev",
  GCP_BACKEND_SERVICE_ACCOUNT: "foods-backend@foods-example.iam.gserviceaccount.com",
  GCP_FRONTEND_SERVICE_ACCOUNT: "foods-frontend@foods-example.iam.gserviceaccount.com",
  GCP_MIGRATION_SERVICE_ACCOUNT: "foods-migrate@foods-example.iam.gserviceaccount.com",
  GCP_MIGRATION_JOB: "foods-migrate-dev", GCP_VPC_CONNECTOR: "foods-connector", GCP_UPLOADS_BUCKET: "foods-example-images",
  GCP_IMAGE_PROCESSOR_SERVICE: "fudia-image-processor-dev",
  GCP_BACKEND_SECRETS: "DATABASE_URL=foods-database:1,FUDIA_CONCIERGE_API_KEY=foods-concierge-key:2",
  GCP_MIGRATION_DATABASE_SECRET: "foods-migration-database:3",
  GITHUB_SHA: "a".repeat(40), GITHUB_RUN_ID: "12", GITHUB_RUN_ATTEMPT: "1",
};

const socketEnv = {...env, GCP_VPC_CONNECTOR: "", GCP_CLOUD_SQL_INSTANCE: "foods-example:southamerica-west1:foods-postgres"};

test("Cloud SQL socket mode requires no VPC and rejects ambiguous or foreign instances", () => {
  assert.equal(configuration("backend", socketEnv).cloudSQLInstance, socketEnv.GCP_CLOUD_SQL_INSTANCE);
  for (const instance of ["foods-postgres", "other-project:southamerica-west1:db", "foods-example:us-central1:db", "foods-example:southamerica-west1:db,evil", "foods-example:southamerica-west1:db;echo"]) {
    assert.throws(() => configuration("backend", {...socketEnv, GCP_CLOUD_SQL_INSTANCE: instance}), /GCP_CLOUD_SQL_INSTANCE/);
  }
  assert.throws(() => configuration("backend", {...env, GCP_CLOUD_SQL_INSTANCE: socketEnv.GCP_CLOUD_SQL_INSTANCE}), /sin GCP_VPC_CONNECTOR/);
});

test("documented gcp-dev variables validate all deploy targets without credentials", async () => {
  const values = JSON.parse(readFileSync(new URL("../gcp/gcp-dev.variables.example.json", import.meta.url), "utf8"));
  const input = {...values, GCP_DATABASE_READY: "true", GITHUB_SHA: "a".repeat(40), GITHUB_RUN_ID: "1", GITHUB_RUN_ATTEMPT: "1"};
  const {processorConfiguration} = await import("../gcp/image-processor-deploy.mjs");
  const api = configuration("backend", input), web = configuration("frontend", input), processor = processorConfiguration(input);
  assert.equal(api.cloudSQLInstance, values.GCP_CLOUD_SQL_INSTANCE);
  assert.equal(processor.cloudSQLInstance, api.cloudSQLInstance);
  assert.equal(web.apiURL, ""); // Resolved after authentication from the deployed API.
  assert.ok(!JSON.stringify(values).includes("postgres://"));
  assert.ok(!values.GCP_VPC_CONNECTOR);
});

test("Cloud SQL is attached to API and migration job without unsupported job deploy flags", () => {
  const config = configuration("backend", socketEnv);
  const job = migrationArgs(config, "image@sha256:digest");
  const api = serviceArgs(config, "image@sha256:digest", false);
  for (const args of [job, api]) {
    assert.equal(args[args.indexOf("--set-cloudsql-instances") + 1], socketEnv.GCP_CLOUD_SQL_INSTANCE);
    assert.ok(!args.includes("--vpc-connector"));
    assert.ok(!args.includes("--vpc-egress"));
  }
  assert.ok(api.includes("--clear-vpc-connector"));
  assert.ok(!job.includes("--clear-vpc-connector"));
});

test("validates all required values before invoking GCP", () => {
  for (const key of ["GCP_PROJECT_ID", "GCP_REGION", "GCP_BACKEND_SERVICE", "GCP_WORKLOAD_IDENTITY_PROVIDER", "GCP_VPC_CONNECTOR", "GCP_UPLOADS_BUCKET", "GCP_MIGRATION_DATABASE_SECRET"]) {
    assert.throws(() => configuration("backend", {...env, [key]: ""}), new RegExp(key));
  }
  assert.throws(() => configuration("unknown", env));
  assert.throws(() => configuration("backend", {...env, GCP_DATABASE_READY: "false"}), /restaurado/);
  assert.throws(() => configuration("backend", {...env, GCP_REGION: "us;echo hacked"}));
  assert.throws(() => configuration("backend", {...env, GCP_BACKEND_MAX_INSTANCES: "0"}));
  assert.throws(() => configuration("backend", {...env, GCP_BACKEND_MAX_INSTANCES: "21"}));
  assert.throws(() => configuration("backend", {...env, GCP_BACKEND_SERVICE_ACCOUNT: "backend@another-project.iam.gserviceaccount.com"}));
});

test("secrets are versioned references, never raw credentials or public variables", () => {
  for (const value of ["DATABASE_URL=postgres://user:password@db/foods", "DATABASE_URL=foods-database:latest", "DATABASE_URL=foods-database:1,DATABASE_URL=other:2", "NEXT_PUBLIC_DATABASE_URL=foods-database:1", "FUDIA_CONCIERGE_API_KEY=key:1"]) {
    assert.throws(() => configuration("backend", {...env, GCP_BACKEND_SECRETS: value}));
  }
  assert.throws(() => configuration("frontend", {...env, GCP_FRONTEND_SECRETS: "DATABASE_URL=database:1"}));
  assert.throws(() => configuration("frontend", {...env, NEXT_PUBLIC_MAPBOX_ACCESS_TOKEN: "pk.public"}));
  assert.throws(() => configuration("frontend", {...env, NEXT_PUBLIC_MAPBOX_ACCESS_TOKEN: "sk.private", GCP_FRONTEND_SECRETS: "MAPBOX_ACCESS_TOKEN=mapbox-server:1"}), /público/);
  assert.doesNotThrow(() => configuration("frontend", {...env, NEXT_PUBLIC_MAPBOX_ACCESS_TOKEN: "pk.public", GCP_FRONTEND_SECRETS: "MAPBOX_ACCESS_TOKEN=mapbox-server:1"}));
});

test("origins reject credentials, non-HTTPS, env-list injection and path rewriting", () => {
  for (const url of ["http://localhost:8080", "https://user:pass@api.example.com", "https://api.example.com/v1", "https://api.example.com?x=1", "https://api.example.com#x", "https://api.example.com,KEY=value"]) {
    assert.throws(() => httpsURL(url, "FOODS_API_URL"));
  }
  assert.equal(httpsURL("https://api.example.com/", "FOODS_API_URL"), "https://api.example.com");
  assert.throws(() => configuration("frontend", {...env, GCP_FRONTEND_SERVICE: env.GCP_BACKEND_SERVICE}));
});

test("migration is a single task using its own identity, database secret and private network", () => {
  const args = migrationArgs(configuration("backend", env), "image@sha256:digest");
  for (const value of ["/app/foods-migrate", "DATABASE_URL=foods-migration-database:3", "DATABASE_MAX_CONNS=2", "private-ranges-only", env.GCP_MIGRATION_SERVICE_ACCOUNT]) assert.ok(args.includes(value));
  assert.equal(args[args.indexOf("--tasks") + 1], "1");
  assert.equal(args[args.indexOf("--max-retries") + 1], "0");
  assert.ok(!args.join(" ").includes("foods-concierge-key"));
});

test("backend service keeps photos persistent and cookies secure without root or startup migrations", () => {
  const config = configuration("backend", env);
  const args = serviceArgs(config, "image@sha256:digest", true, "123", "456");
  assert.ok(args.includes("--no-traffic"));
  assert.ok(args.includes("--no-invoker-iam-check")); // Existing app sessions authorize API; no IAM token BFF yet.
  assert.ok(args.includes(`HTTP_ADDR=:8080,FOODS_COOKIE_SECURE=true,DATABASE_MAX_CONNS=12,PRODUCT_IMAGE_STORAGE=gcs,PRODUCT_IMAGES_BUCKET=${env.GCP_UPLOADS_BUCKET},PRODUCT_IMAGES_SIGNER=${env.GCP_BACKEND_SERVICE_ACCOUNT}`));
  assert.ok(!args.includes("--add-volume"));
  assert.ok(args.includes("--remove-volume"));
  assert.ok(!args.includes("/app/foods-migrate"));
  assert.throws(() => serviceArgs(config, "digest", true, "0", "0"));
  assert.ok(!serviceArgs(config, "digest", false).includes("--no-traffic"));
});

function runner(config, {missingAPI = false, missingService = false, failMigration = false} = {}) {
  const calls = [];
  let deployed = false;
  const digest = `${config.image.split(":")[0]}@sha256:${"b".repeat(64)}`;
  const run = (binary, args) => {
    calls.push([binary, ...args]);
    if (binary === "docker" && args[0] === "image") return digest;
    if (binary === "docker" && args[0] === "run") return args.at(-1) === "-u" ? "100" : "101";
    if (binary === "gcloud" && args.slice(0, 3).join(" ") === "run jobs execute" && failMigration) throw new Error("migration failed");
    if (binary === "gcloud" && args.slice(0, 2).join(" ") === "run deploy") deployed = true;
    if (binary === "gcloud" && args.slice(0, 3).join(" ") === "run services describe") {
      if (config.target === "frontend" && args[3] === config.backendService) return missingAPI ? null : JSON.stringify({status: {url: "https://api.example.com"}});
      if (!deployed && missingService) return null;
      return JSON.stringify({status: {url: "https://app.example.com", latestReadyRevisionName: "ready-002", traffic: [{tag: "candidate", revisionName: "ready-002", url: "https://candidate.example.com"}]}});
    }
    return "";
  };
  return {run, calls, digest};
}

test("socket migration clears previous job network before executing, then deploys API", async () => {
  const config = configuration("backend", socketEnv);
  const mock = runner(config);
  await deploy(config, {}, mock.run, async () => {});
  const step = prefix => mock.calls.findIndex(args => args.slice(0, prefix.split(" ").length).join(" ") === prefix);
  const create = step("gcloud run jobs deploy"), clear = step("gcloud run jobs update"), execute = step("gcloud run jobs execute"), api = step("gcloud run deploy");
  assert.ok(create >= 0 && create < clear && clear < execute && execute < api);
  assert.ok(mock.calls[clear].includes("--clear-vpc-connector"));
  assert.ok(mock.calls[clear].includes("--clear-network"));
});

test("deploy uses digest, awaits migrations, checks candidate then promotes exact revision", async () => {
  const config = configuration("backend", env);
  const mock = runner(config);
  await deploy(config, {}, mock.run, async (url, target, version) => {
    assert.equal(url, "https://candidate.example.com"); assert.equal(target, "backend"); assert.equal(version, "aaaaaaaa");
    mock.calls.push(["smoke"]);
  });
  const migration = mock.calls.findIndex(args => args.slice(0, 4).join(" ") === "gcloud run jobs execute");
  const deployment = mock.calls.findIndex(args => args.slice(0, 3).join(" ") === "gcloud run deploy");
  const smoke = mock.calls.findIndex(args => args[0] === "smoke");
  const promotion = mock.calls.findIndex(args => args.includes("update-traffic"));
  assert.ok(migration < deployment && deployment < smoke && smoke < promotion);
  assert.ok(mock.calls[deployment].includes(mock.digest));
  assert.ok(mock.calls[promotion].includes("ready-002=100"));
  assert.ok(mock.calls[promotion].includes("deployed=ready-002"));
  assert.ok(mock.calls[deployment].includes(`fudia-commit=${config.sha},fudia-component=backend`));
});

test("migration and smoke failures do not promote a revision", async () => {
  const config = configuration("backend", env);
  const failedMigration = runner(config, {failMigration: true});
  await assert.rejects(deploy(config, {}, failedMigration.run, async () => {}), /migration failed/);
  assert.ok(!failedMigration.calls.some(args => args[2] === "deploy" && args[1] === "run"));
  const failedSmoke = runner(config);
  await assert.rejects(deploy(config, {}, failedSmoke.run, async () => {throw new Error("smoke failed");}));
  assert.ok(!failedSmoke.calls.some(args => args.includes("update-traffic")));
  const firstSmoke = runner(config, {missingService: true});
  await assert.rejects(deploy(config, {}, firstSmoke.run, async () => {throw new Error("first smoke failed");}));
  assert.ok(!firstSmoke.calls.some(args => args.includes("update-traffic"))); // No completed marker even on first auto-live revision.
});

test("frontend discovers backend URL at runtime, injects no DB and supports separate public/private Mapbox tokens", async () => {
  const config = configuration("frontend", {...env, NEXT_PUBLIC_MAPBOX_ACCESS_TOKEN: "pk.public", GCP_FRONTEND_SECRETS: "MAPBOX_ACCESS_TOKEN=mapbox-server:1"});
  const mock = runner(config);
  await deploy(config, {}, mock.run, async () => {});
  const service = mock.calls.find(args => args.slice(0, 3).join(" ") === "gcloud run deploy");
  assert.ok(service.includes("FOODS_API_URL=https://api.example.com"));
  assert.ok(service.includes(`fudia-commit=${config.sha},fudia-component=frontend`));
  assert.ok(service.includes("MAPBOX_ACCESS_TOKEN=mapbox-server:1"));
  assert.ok(!service.join(" ").includes("DATABASE_URL"));
  assert.ok(!mock.calls.some(args => args.includes("jobs")));
  const build = mock.calls.find(args => args[0] === "docker" && args[1] === "build");
  assert.ok(build.includes("NEXT_PUBLIC_MAPBOX_ACCESS_TOKEN=pk.public"));
});

test("missing backend blocks frontend before publishing anything", async () => {
  const config = configuration("frontend", env);
  const mock = runner(config, {missingAPI: true});
  await assert.rejects(deploy(config, {}, mock.run), /backend primero/);
  assert.ok(!mock.calls.some(args => args[0] === "docker"));
});

test("frontend honors an explicit HTTPS API URL and never receives Cloud SQL settings", async () => {
  const config = configuration("frontend", {...socketEnv, FOODS_API_URL: "https://configured-api.run.app"});
  const mock = runner(config, {missingAPI: true});
  await deploy(config, {}, mock.run, async () => {});
  const service = mock.calls.find(args => args.slice(0, 3).join(" ") === "gcloud run deploy");
  assert.ok(service.includes("FOODS_API_URL=https://configured-api.run.app"));
  for (const value of ["--set-cloudsql-instances", "DATABASE_URL", "--vpc-connector"]) assert.ok(!service.join(" ").includes(value));
  assert.ok(!mock.calls.some(args => args.slice(0, 4).join(" ") === `gcloud run services describe ${env.GCP_BACKEND_SERVICE}`));
  assert.throws(() => serviceArgs(configuration("frontend", env), "image", false));
});

test("image can be built before OIDC; authenticated deploy does not rebuild", async () => {
  const config = configuration("backend", env);
  const mock = runner(config);
  buildImage(config, mock.run);
  assert.equal(mock.calls.length, 1);
  assert.equal(mock.calls[0][0], "docker");
  await deploy(config, {GCP_IMAGE_BUILT: "true"}, mock.run, async () => {});
  assert.equal(mock.calls.filter(args => args[0] === "docker" && args[1] === "build").length, 1);
});

test("GCP component workflows are reusable, gated and keyless without changing AWS", () => {
  for (const file of ["foods-backend-gcp.yml", "foods-admin-web-gcp.yml"]) {
    const source = readFileSync(new URL(`../../.github/workflows/${file}`, import.meta.url), "utf8");
    for (const value of ["workflow_call:", "workflow_dispatch:", "needs: verify", "environment: ${{ inputs.environment }}", "gcp-production", "google-github-actions/auth@v3", "id-token: write", "cancel-in-progress: false", "node --test foods-infrastructure/tests/*.test.mjs"]) assert.ok(source.includes(value));
    assert.ok(!source.includes("push:")); assert.ok(!source.includes("credentials_json")); assert.ok(!source.includes("AWS_"));
    assert.ok(source.indexOf("before requesting short-lived credentials") < source.indexOf("uses: google-github-actions/auth"));
    assert.ok(source.includes("access_token_lifetime: 3600s"));
    assert.ok(source.includes("create_credentials_file: false"));
    assert.ok(source.includes("CLOUDSDK_AUTH_ACCESS_TOKEN: ${{ steps.auth.outputs.access_token }}"));
  }
  for (const file of ["foods-backend.yml", "foods-admin-web.yml"]) assert.match(readFileSync(new URL(`../../.github/workflows/${file}`, import.meta.url), "utf8"), /aws-actions\/configure-aws-credentials/);
});

test("only GCP database workflows require Cloud SQL configuration and verify PostgreSQL 18", () => {
  for (const file of ["foods-backend-gcp.yml", "foods-image-processor-gcp.yml"]) {
    const source = readFileSync(new URL(`../../.github/workflows/${file}`, import.meta.url), "utf8");
    assert.ok(source.includes("image: postgres:18"));
    assert.ok(source.includes("GCP_CLOUD_SQL_INSTANCE: ${{ vars.GCP_CLOUD_SQL_INSTANCE }}"));
    assert.ok(source.includes("GCP_DATABASE_READY: ${{ inputs.database_ready }}"));
    assert.ok(source.includes("default: false")); // Manual execution still requires confirmation.
  }
});
