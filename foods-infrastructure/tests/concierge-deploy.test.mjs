import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {CA_PATH, conciergeConfiguration, deployConcierge, httpArgs, runtimeArgs} from "../gcp/concierge-deploy.mjs";

const base = JSON.parse(readFileSync(new URL("../gcp/gcp-dev.variables.example.json", import.meta.url)));
const concierge = JSON.parse(readFileSync(new URL("../gcp/concierge-dev.variables.example.json", import.meta.url)));
const env = {...base, ...concierge,
  GCP_BACKEND_SECRETS: `${base.GCP_BACKEND_SECRETS},FUDIA_CONCIERGE_API_KEY=concierge-api-key-fudia-dev:1`,
  WHATSAPP_PHONE_ID: "123456", WHATSAPP_GRAPH_VERSION: "v26.0",
  GITHUB_SHA: "a".repeat(40), GITHUB_RUN_ID: "123", GITHUB_RUN_ATTEMPT: "1",
  GCP_CONCIERGE_DEPLOY_APPROVED: "true", GCP_IMAGE_BUILT: "true"};
const digest = `${conciergeConfiguration(env).image.split(":")[0]}@sha256:${"b".repeat(64)}`;
const apiURL = "https://api-example.run.app";

function harness(options = {}) {
  const calls = [];
  const backend = {status: {url: apiURL, traffic: [{percent: 100, revisionName: "api-release"}]}};
  const active = {spec: {serviceAccountName: base.GCP_BACKEND_SERVICE_ACCOUNT, containers: [{env: [
    {name: "FUDIA_CONCIERGE_API_KEY", valueFrom: {secretKeyRef: {name: "concierge-api-key-fudia-dev", key: "1"}}},
  ]}]}};
  if (options.wrongKey) active.spec.containers[0].env[0].valueFrom.secretKeyRef.key = "2";
  const candidate = {status: {url: "https://concierge-example.run.app", latestReadyRevisionName: "concierge-candidate",
    traffic: [{tag: "candidate", revisionName: "concierge-candidate", url: "https://candidate-example.run.app"}]}};
  let httpReads = 0;
  const run = (binary, args) => {
    calls.push([binary, ...args]);
    const command = args.slice(0, 3).join(" ");
    if (command === "run services describe") {
      if (args[3] === base.GCP_BACKEND_SERVICE) return JSON.stringify(backend);
      return ++httpReads === 1 && options.newService ? null : JSON.stringify(candidate);
    }
    if (command === "run revisions describe") return JSON.stringify(active);
    if (command === "run worker-pools describe") return options.newService ? null : "{}";
    if (args[0] === "image" && args[1] === "inspect") return options.badDigest ? "wrong@sha256:no" : digest;
    if (command === "run jobs execute" && options.preflightFails) throw new Error("preflight failed");
    if (command === "run worker-pools deploy" && options.workerFails) throw new Error("worker failed");
    return "";
  };
  return {calls, run};
}

test("Concierge uses existing environment identity, numeric secret refs and safe defaults", () => {
  const config = conciergeConfiguration(env);
  assert.equal(config.sessionTTL, "3600");
  assert.equal(config.instances, "1");
  assert.equal(config.location, "global");
  assert.equal(config.secrets.get("REDIS_URL"), "redis-url-fudia-dev:1");
  for (const key of ["GCP_CONCIERGE_NETWORK", "GCP_CONCIERGE_SUBNET", "WHATSAPP_PHONE_ID", "WHATSAPP_GRAPH_VERSION", "GCP_CONCIERGE_REDIS_CA_SECRET"]) {
    assert.throws(() => conciergeConfiguration({...env, [key]: ""}));
  }
  for (const overrides of [
    {WHATSAPP_GRAPH_VERSION: "https://graph.facebook.com/v26.0"},
    {GCP_CONCIERGE_SECRETS: concierge.GCP_CONCIERGE_SECRETS.replace("redis-url-fudia-dev:1", "redis-url-fudia-dev:latest")},
    {GCP_CONCIERGE_REDIS_CA_SECRET: "CA:latest"},
    {GCP_CONCIERGE_HTTP_SERVICE_ACCOUNT: env.GCP_CONCIERGE_WORKER_SERVICE_ACCOUNT},
    {GCP_CONCIERGE_HTTP_SERVICE: base.GCP_BACKEND_SERVICE},
    {GCP_CONCIERGE_HTTP_SERVICE: base.GCP_FRONTEND_SERVICE},
    {GCP_CONCIERGE_NETWORK: "network;echo hello"},
    {GCP_BACKEND_SECRETS: base.GCP_BACKEND_SECRETS},
    {GCP_CONCIERGE_WORKER_INSTANCES: "0"},
  ]) assert.throws(() => conciergeConfiguration({...env, ...overrides}));
});

test("HTTP and worker share verified TLS, private-only egress and provider config, never plaintext secrets", () => {
  const c = conciergeConfiguration(env);
  const args = runtimeArgs(c, apiURL);
  const vars = args[args.indexOf("--set-env-vars") + 1];
  assert.match(vars, /WORKER_ENABLED=false/);
  assert.match(vars, /LLM_PROVIDER=vertex/);
  assert.match(vars, /SESSION_TTL_SECONDS=3600/);
  assert.match(vars, /GOOGLE_CLOUD_PROJECT=project-/);
  assert.ok(vars.includes(`REDIS_CA_CERTFILE=${CA_PATH}`));
  assert.ok(!vars.includes("REDIS_URL="));
  assert.ok(args[args.indexOf("--set-secrets") + 1].includes(`${CA_PATH}=redis-ca-fudia-dev:1`));
  assert.ok(args.includes("private-ranges-only"));
  assert.ok(runtimeArgs(c, apiURL, true).join(" ").includes("WORKER_ENABLED=true"));
  assert.throws(() => runtimeArgs(c, "http://localhost:8080"));
  assert.ok(httpArgs(c, digest, apiURL, true).includes("--no-traffic"));
  assert.ok(!httpArgs(c, digest, apiURL, false).includes("--no-traffic"));
});

test("actual backend secret is checked, preflight precedes consumers, smoke precedes promotions", async () => {
  const h = harness();
  await deployConcierge(conciergeConfiguration(env), env, h.run, async url => h.calls.push(["smoke", url]));
  const index = prefix => h.calls.findIndex(call => call.slice(0, prefix.length).join(" ") === prefix.join(" "));
  assert.ok(index(["gcloud", "run", "revisions", "describe"]) < index(["docker", "push"]));
  assert.ok(index(["gcloud", "run", "jobs", "execute"]) < index(["gcloud", "run", "deploy"]));
  assert.ok(index(["smoke"]) < index(["gcloud", "run", "worker-pools", "deploy"]));
  assert.ok(index(["gcloud", "run", "worker-pools", "update-instance-split"]) < index(["gcloud", "run", "services", "update-traffic"]));
  const job = h.calls.find(call => call.slice(0, 4).join(" ") === "gcloud run jobs deploy");
  assert.ok(job.includes("--args=-m,src.preflight"));
  assert.ok(!job.includes("--clear-vpc-connector"));
  const pool = h.calls.find(call => call.slice(0, 4).join(" ") === "gcloud run worker-pools deploy");
  assert.ok(pool.includes("--no-promote"));
  assert.ok(pool.includes("--args=-m,src.worker"));
  assert.ok(!pool.includes("--port"));
  assert.ok(!h.calls.some(call => call.includes("add-iam-policy-binding")));
});

test("no confirmation, wrong backend key, invalid digest, failed preflight or smoke never start a worker", async () => {
  for (const options of [{unapproved: true}, {wrongKey: true}, {badDigest: true}, {preflightFails: true}, {smokeFails: true}]) {
    const h = harness(options);
    await assert.rejects(deployConcierge(conciergeConfiguration(env), {...env, GCP_CONCIERGE_DEPLOY_APPROVED: options.unapproved ? "false" : "true"}, h.run,
      async () => { if (options.smokeFails) throw new Error("smoke failed"); }));
    assert.ok(!h.calls.some(call => call.slice(0, 4).join(" ") === "gcloud run worker-pools deploy"));
    assert.ok(!h.calls.some(call => call.includes("update-traffic")));
    if (options.unapproved) assert.equal(h.calls.length, 0);
  }
});

test("worker failure leaves existing HTTP traffic untouched; initial installation has no no-promote", async () => {
  const failed = harness({workerFails: true});
  await assert.rejects(deployConcierge(conciergeConfiguration(env), env, failed.run, async () => {}));
  assert.ok(!failed.calls.some(call => call.includes("update-traffic")));
  const first = harness({newService: true});
  await deployConcierge(conciergeConfiguration(env), env, first.run, async () => {});
  assert.ok(!first.calls.find(call => call.slice(0, 4).join(" ") === "gcloud run worker-pools deploy").includes("--no-promote"));
});

test("release is manual, verifies before backend and deploy, uses WIF without credential files", () => {
  const workflow = readFileSync(new URL("../../.github/workflows/fudia-concierge-gcp.yml", import.meta.url), "utf8");
  assert.match(workflow, /workflow_dispatch:/);
  assert.doesNotMatch(workflow, /^  push:|^  schedule:/m);
  assert.match(workflow, /uses: \.\/\.github\/workflows\/foods-backend-gcp.yml/);
  assert.match(workflow, /needs: \[verify, backend\]/);
  assert.match(workflow, /inputs.activate_concierge && inputs.database_ready/);
  assert.match(workflow, /create_credentials_file: false/);
  assert.ok(workflow.indexOf("concierge-deploy.mjs build") < workflow.indexOf("uses: google-github-actions/auth"));
  assert.doesNotMatch(workflow, /secrets\.WHATSAPP|secrets\.REDIS|NEXT_PUBLIC_.*CONCIERGE/);
});
