import {spawnSync} from "node:child_process";
import {appendFileSync} from "node:fs";
import {fileURLToPath} from "node:url";
import {resolve} from "node:path";

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

if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) {
  try {
    if (!process.env.GITHUB_OUTPUT) throw new Error("Falta GITHUB_OUTPUT para publicar los componentes modificados.");
    const selected = changedComponents(changedFiles(process.env.BASE_SHA, process.env.HEAD_SHA));
    appendFileSync(process.env.GITHUB_OUTPUT, Object.entries(selected).map(([key, value]) => `${key}=${value}\n`).join(""));
    console.log(JSON.stringify(selected));
  } catch (error) {
    console.error(error.message);
    process.exitCode = 1;
  }
}
