import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";

const workflow = name => readFileSync(new URL(`../../.github/workflows/${name}.yml`, import.meta.url), "utf8");
const source = workflow("foods-gcp");

test("GCP coordinator deploys pushes only from the agreed branch into development", () => {
  assert.match(source, /on:\s+push:\s+branches: \[feat\/ui-standard\]/);
  for (const path of ["foods-backend/**", "foods-admin-web/**", "fudia-image-processor/**", "foods-infrastructure/**", ".github/workflows/foods-*-gcp.yml", ".github/workflows/foods-gcp.yml"]) {
    assert.ok(source.includes(`- "${path}"`));
  }
  assert.equal((source.match(/environment: gcp-dev/g) ?? []).length, 3);
  assert.ok(!source.includes("gcp-production"));
  assert.ok(!source.includes("AWS_"));
  assert.ok(!source.includes("workflow_run:"));
  assert.match(source, /concurrency:\s+group: foods-gcp-dev-push-deploy\s+cancel-in-progress: false/);
});

test("coordinator awaits processor and backend success before frontend using the same commit", () => {
  assert.match(source, /processor:\s+uses: \.\/\.github\/workflows\/foods-image-processor-gcp.yml/);
  assert.match(source, /backend:\s+needs: processor\s+uses: \.\/\.github\/workflows\/foods-backend-gcp.yml/);
  assert.match(source, /frontend:\s+needs: backend\s+uses: \.\/\.github\/workflows\/foods-admin-web-gcp.yml/);
  assert.equal((source.match(/database_ready: true/g) ?? []).length, 2);
  assert.equal((source.match(/secrets: inherit/g) ?? []).length, 3);
  assert.match(source, /permissions:\s+contents: read\s+id-token: write/);
  assert.ok(!source.includes("always()"));
  assert.ok(!source.includes("continue-on-error"));
  assert.ok(!source.includes("@master"));
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
      assert.equal((child.match(/default: false/g) ?? []).length, 2);
      assert.ok(child.includes("GCP_DATABASE_READY: ${{ inputs.database_ready }}"));
    }
  }
  const deploy = readFileSync(new URL("../gcp/deploy.mjs", import.meta.url), "utf8");
  assert.ok(deploy.includes('config.target === "frontend" && !config.apiURL'));
  assert.ok(deploy.includes("config.apiURL = httpsURL(JSON.parse(backend).status.url"));
});
