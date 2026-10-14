import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {oidcDiagnostics, selectedClaims} from "../gcp/oidc-diagnostics.mjs";

const identity = {
  sub: "repo:example@123/project@456:environment:gcp-dev",
  repository: "example/project",
  ref: "refs/heads/feat/ui-standard",
  environment: "gcp-dev",
};
const jwt = claims => `header.${Buffer.from(JSON.stringify(claims)).toString("base64url")}.signature`;

test("OIDC diagnostics returns only the four approved fields, including immutable subjects", () => {
  const token = jwt({...identity, aud: "private-audience", sensitive: "DO_NOT_PRINT", job_workflow_ref: "ignored"});
  const result = selectedClaims(token);
  assert.deepEqual(result, identity);
  assert.ok(!JSON.stringify(result).includes("DO_NOT_PRINT"));
  assert.ok(!JSON.stringify(result).includes(token));
});

test("missing environment stays null instead of pretending the token has an environment", () => {
  assert.deepEqual(selectedClaims(jwt({...identity, environment: undefined})), {...identity, environment: null});
});

test("malformed tokens do not appear in error messages", () => {
  for (const token of [undefined, "DO_NOT_PRINT", "header.DO_NOT_PRINT.signature", jwt({sub: "DO_NOT_PRINT"})]) {
    assert.throws(() => selectedClaims(token), {message: "No se pudieron leer los campos de diagnóstico OIDC."});
  }
});

test("request authenticates only to the injected OIDC endpoint and returns selected metadata", async () => {
  const env = {ACTIONS_ID_TOKEN_REQUEST_URL: "https://example.test/oidc", ACTIONS_ID_TOKEN_REQUEST_TOKEN: "DO_NOT_PRINT"};
  const result = await oidcDiagnostics(env, async (url, options) => {
    assert.equal(url, env.ACTIONS_ID_TOKEN_REQUEST_URL);
    assert.equal(options.headers.Authorization, "Bearer DO_NOT_PRINT");
    assert.ok(options.signal instanceof AbortSignal);
    return {ok: true, json: async () => ({value: jwt(identity)})};
  });
  assert.deepEqual(result, identity);
});

test("request errors never expose response bodies, headers or exception details", async () => {
  const env = {ACTIONS_ID_TOKEN_REQUEST_URL: "https://example.test/oidc", ACTIONS_ID_TOKEN_REQUEST_TOKEN: "DO_NOT_PRINT"};
  for (const request of [async () => {throw new Error("DO_NOT_PRINT");}, async () => ({ok: false, json: async () => ({value: "DO_NOT_PRINT"})})]) {
    await assert.rejects(oidcDiagnostics(env, request), {message: "No se pudo solicitar el token para el diagnóstico OIDC."});
  }
  await assert.rejects(oidcDiagnostics({}, () => assert.fail("No request without credentials")), /solo dentro del job/);
});

test("temporary diagnostics executes only in the processor deploy job before Google authentication", () => {
  const source = readFileSync(new URL("../../.github/workflows/foods-image-processor-gcp.yml", import.meta.url), "utf8");
  const build = source.indexOf("run: node foods-infrastructure/gcp/image-processor-deploy.mjs build");
  const diagnostic = source.indexOf("run: node foods-infrastructure/gcp/oidc-diagnostics.mjs");
  const auth = source.indexOf("uses: google-github-actions/auth@v3");
  assert.ok(build < diagnostic && diagnostic < auth);
  assert.ok(source.includes("id-token: write"));
  assert.ok(source.indexOf("  deploy:") < diagnostic);
});
