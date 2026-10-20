import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {changedComponents} from "../gcp/changed-components.mjs";

const workflow = name => readFileSync(new URL(`../../.github/workflows/${name}.yml`, import.meta.url), "utf8");
const source = workflow("foods-gcp");

test("GCP coordinator deploys pushes only from the agreed branch into development", () => {
  assert.match(source, /on:\s+push:\s+branches: \[feat\/ui-standard\]/);
  for (const path of ["foods-backend/**", "foods-admin-web/**", "fudia-image-processor/**", "fudia-concierge/**", "foods-infrastructure/**", ".github/workflows/foods-*-gcp.yml", ".github/workflows/foods-gcp.yml", ".github/workflows/fudia-concierge-gcp.yml"]) {
    assert.ok(source.includes(`- "${path}"`));
  }
  assert.equal((source.match(/environment: gcp-dev/g) ?? []).length, 5);
  assert.ok(source.includes('- "!**/*.md"'));
  assert.ok(source.lastIndexOf('- "fudia-concierge/prompts/**"') > source.indexOf('- "!**/*.md"'));
  assert.ok(!source.includes("gcp-production"));
  assert.ok(!source.includes("AWS_"));
  assert.ok(!source.includes("workflow_run:"));
  assert.match(source, /concurrency:\s+group: foods-gcp-dev-push-deploy\s+queue: max\s+cancel-in-progress: false/);
});

test("coordinator selects changed components and accepts skipped dependencies but not failures", () => {
  assert.match(source, /processor:\s+needs: changes\s+if: \$\{\{ needs.changes.outputs.processor == 'true' \}\}/);
  assert.match(source, /backend:\s+needs: \[changes, processor\]/);
  assert.match(source, /frontend:\s+needs: \[changes, processor, backend\]/);
  assert.match(source, /concierge:\s+needs: \[changes, processor, backend\]/);
  for (const name of ["processor", "backend", "frontend", "concierge"]) assert.ok(source.includes(`needs.changes.outputs.${name} == 'true'`));
  assert.equal((source.match(/needs.processor.result == 'success' \|\| needs.processor.result == 'skipped'/g) ?? []).length, 3);
  assert.ok(source.includes("needs.backend.result == 'success' || needs.backend.result == 'skipped'"));
  assert.equal((source.match(/!cancelled\(\) && needs.changes.result == 'success'/g) ?? []).length, 3);
  assert.ok(source.includes("fetch-depth: 0"));
  assert.ok(!source.includes("BASE_SHA:"));
  assert.ok(source.includes("HEAD_SHA: ${{ github.sha }}"));
  const detection = source.slice(source.indexOf("  changes:"), source.indexOf("\n  processor:"));
  for (const key of ["GCP_PROJECT_ID", "GCP_REGION", "GCP_IMAGE_PROCESSOR_SERVICE", "GCP_BACKEND_SERVICE", "GCP_FRONTEND_SERVICE", "GCP_CONCIERGE_HTTP_SERVICE", "GCP_CONCIERGE_WORKER_POOL"]) {
    assert.ok(detection.includes(key + ': ${{ vars.' + key + ' }}'));
  }
  assert.ok(detection.includes("environment: gcp-dev")); // Required by the existing WIF subject.
  assert.ok(detection.includes("id-token: write"));
  assert.ok(detection.includes("create_credentials_file: false"));
  assert.ok(detection.indexOf("uses: google-github-actions/auth") < detection.indexOf("- id: detect"));
  assert.ok(detection.includes("CLOUDSDK_AUTH_ACCESS_TOKEN: ${{ steps.auth.outputs.access_token }}"));
  assert.ok(source.includes("run: node foods-infrastructure/gcp/changed-components.mjs"));
  assert.equal((source.match(/database_ready: true/g) ?? []).length, 2);
  assert.equal((source.match(/secrets: inherit/g) ?? []).length, 4);
  assert.match(source, /permissions:\s+contents: read\s+id-token: write/);
  assert.ok(!source.includes("always()"));
  assert.ok(!source.includes("continue-on-error"));
  assert.ok(!source.includes("@master"));
  assert.ok(source.includes("deploy_backend: false"));
  assert.ok(source.includes("activate_concierge: true"));
  assert.ok(source.indexOf("run: node --test foods-infrastructure/tests/*.test.mjs") < source.indexOf("uses: google-github-actions/auth"));
});

test("component workflows preserve tests, environment credentials and manual confirmation without duplicate push triggers", () => {
  for (const name of ["foods-image-processor-gcp", "foods-backend-gcp", "foods-admin-web-gcp"]) {
    const child = workflow(name);
    assert.ok(child.includes("workflow_call:"));
    assert.ok(child.includes("workflow_dispatch:"));
    assert.ok(!child.includes("push:"));
    assert.match(child, /deploy:\s+needs: verify/);
    assert.ok(child.includes("environment: ${{ inputs.environment }}"));
    if (name !== "foods-admin-web-gcp") {
      assert.equal((child.match(/default: false/g) ?? []).length, 1); // Manual confirmation remains false by default.
      assert.match(child, /workflow_call:[\s\S]*?database_ready:\s+type: boolean\s+required: true\s+workflow_dispatch:/);
      assert.ok(child.includes("GCP_DATABASE_READY: ${{ inputs.database_ready }}"));
    }
  }
  const deploy = readFileSync(new URL("../gcp/deploy.mjs", import.meta.url), "utf8");
  assert.ok(deploy.includes('config.target === "frontend" && !config.apiURL'));
  assert.ok(deploy.includes("config.apiURL = httpsURL(JSON.parse(backend).status.url"));
});

// These workflow conditions use the common JS/GitHub expression subset.
// Read the actual conditions instead of testing a separately reimplemented policy.
function condition(job, needs, cancelled = false) {
  const expression = new RegExp(`\\n  ${job}:\\n[\\s\\S]*?\\n    if: (?:>-\\n\\s*)?\\$\\{\\{([\\s\\S]*?)\\}\\}`).exec(source)?.[1];
  assert.ok(expression, `Missing condition for ${job}`);
  return new Function("needs", "cancelled", `return (${expression});`)(needs, () => cancelled);
}

test("all combinations run only selected jobs even when earlier jobs are skipped", () => {
  for (let mask = 0; mask < 16; mask++) {
    const paths = [];
    if (mask & 1) paths.push("fudia-image-processor/main.go");
    if (mask & 2) paths.push("foods-backend/main.go");
    if (mask & 4) paths.push("foods-admin-web/page.tsx");
    if (mask & 8) paths.push("fudia-concierge/src/worker.py");
    const selected = changedComponents(paths);
    const needs = {
      changes: {result: "success", outputs: Object.fromEntries(Object.entries(selected).map(([key, value]) => [key, String(value)]))},
      processor: {result: selected.processor ? "success" : "skipped"},
      backend: {result: selected.backend ? "success" : "skipped"},
    };
    for (const job of ["processor", "backend", "frontend", "concierge"]) assert.equal(condition(job, needs), selected[job], `${job}, mask=${mask}`);
  }
});

test("failed detection, failed selected dependencies and cancellation block downstream deployments", () => {
  const needs = {
    changes: {result: "success", outputs: {processor: "true", backend: "true", frontend: "true", concierge: "true"}},
    processor: {result: "success"}, backend: {result: "success"},
  };
  for (const result of ["failure", "cancelled"]) {
    const failedProcessor = {...needs, processor: {result}, backend: {result: "skipped"}};
    assert.equal(condition("backend", failedProcessor), false);
    assert.equal(condition("frontend", failedProcessor), false);
    assert.equal(condition("frontend", {...needs, backend: {result}}), false);
    assert.equal(condition("concierge", failedProcessor), false);
    assert.equal(condition("concierge", {...needs, backend: {result}}), false);
  }
  for (const job of ["backend", "frontend", "concierge"]) {
    assert.equal(condition(job, {...needs, changes: {...needs.changes, result: "failure"}}), false);
    assert.equal(condition(job, needs, true), false);
  }
});
