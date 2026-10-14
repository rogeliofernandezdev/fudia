import test from "node:test";
import assert from "node:assert/strict";
import {changedComponents, changedFiles} from "../gcp/changed-components.mjs";

const only = component => ({processor: component === "processor", backend: component === "backend", frontend: component === "frontend"});
const all = {processor: true, backend: true, frontend: true};

test("frontend, backend and processor changes select only their own workflow", () => {
  assert.deepEqual(changedComponents(["foods-admin-web/src/app/page.tsx"]), only("frontend"));
  assert.deepEqual(changedComponents(["foods-backend/internal/modules/menu/service.go", "foods-backend/migrations/000081_example.up.sql"]), only("backend"));
  assert.deepEqual(changedComponents(["fudia-image-processor/internal/processor/process.go"]), only("processor"));
});

test("component workflow changes do not rebuild unrelated services", () => {
  for (const [file, component] of [["foods-admin-web-gcp.yml", "frontend"], ["foods-backend-gcp.yml", "backend"], ["foods-image-processor-gcp.yml", "processor"]]) {
    assert.deepEqual(changedComponents([`.github/workflows/${file}`]), only(component));
  }
});

test("mixed changes select the union of affected components", () => {
  assert.deepEqual(changedComponents(["foods-admin-web/Dockerfile", "foods-backend/go.mod"]), {processor: false, backend: true, frontend: true});
  assert.deepEqual(changedComponents(["foods-backend/go.mod", "fudia-image-processor/go.mod"]), {processor: true, backend: true, frontend: false});
});

test("shared infrastructure, coordinator changes and first branch push select all components", () => {
  for (const path of ["foods-infrastructure/gcp/deploy.mjs", "foods-infrastructure/gcp/changed-components.mjs", "foods-infrastructure/tests/cloud-run-deploy.test.mjs", ".github/workflows/foods-gcp.yml"]) {
    assert.deepEqual(changedComponents([path]), all);
  }
  assert.deepEqual(changedComponents(null), all);
});

test("empty or unrelated changes do not select a GCP deployment", () => {
  assert.deepEqual(changedComponents([]), only(null));
  assert.deepEqual(changedComponents(["README.md", "docs/01_PRODUCT_SCOPE.md", "fudia-concierge/main.py", ".github/workflows/foods-backend.yml"]), only(null));
});

test("diff covers both ends of a multi-commit push and deleted/renamed paths without a shell", () => {
  const before = "a".repeat(40), after = "b".repeat(40);
  const paths = changedFiles(before, after, (binary, args, options) => {
    assert.equal(binary, "git");
    assert.deepEqual(args, ["diff", "--name-only", "--no-renames", "-z", before, after, "--"]);
    assert.equal(options.shell, undefined);
    return {status: 0, stdout: "foods-admin-web/old file.tsx\0foods-backend/new file.go\0"};
  });
  assert.deepEqual(changedComponents(paths), {processor: false, backend: true, frontend: true});
});

test("first push requires no nonexistent base commit, while invalid or failed diffs stop selection", () => {
  assert.equal(changedFiles("0".repeat(40), "b".repeat(40), () => assert.fail("No diff for a new branch")), null);
  for (const before of ["", "HEAD~1", "a; command"]) assert.throws(() => changedFiles(before, "b".repeat(40)), /commits válidos/);
  assert.throws(() => changedFiles("a".repeat(40), "b".repeat(40), () => ({status: 1})), /No se pudo comparar/);
});
