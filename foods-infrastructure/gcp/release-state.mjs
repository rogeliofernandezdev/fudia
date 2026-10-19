export const COMMIT_LABEL = "fudia-commit";
export const COMPONENT_LABEL = "fudia-component";
export const RELEASE_TAG = "deployed";

export function releaseLabels(sha, component) {
  if (!/^[a-f0-9]{40}$/.test(sha ?? "") || !["processor", "backend", "frontend"].includes(component)) {
    throw new Error("Identidad de revisión inválida.");
  }
  return `${COMMIT_LABEL}=${sha},${COMPONENT_LABEL}=${component}`;
}

export function candidateRevision(service) {
  const candidate = service?.status?.traffic?.find(target => target.tag === "candidate");
  if (!candidate?.revisionName || candidate.revisionName !== service.status.latestReadyRevisionName) {
    throw new Error("No se pudo verificar la revisión candidata; no se promueve tráfico.");
  }
  return candidate;
}

// Move traffic and the completed-release marker together, only after stage gates.
// A first deployment is automatically live before smoke; without this tag it
// must still be considered pending. Failed candidates never advance the marker.
export function promotionArgs(service, revision) {
  return ["run", "services", "update-traffic", service, "--to-revisions", `${revision}=100`,
    "--update-tags", `${RELEASE_TAG}=${revision}`];
}

export function activeRelease(service) {
  const traffic = service?.status?.traffic;
  if (!Array.isArray(traffic)) throw new Error("Cloud Run no devolvió un estado de tráfico válido.");
  const completed = traffic.filter(target => target.tag === RELEASE_TAG);
  if (completed.length > 1) throw new Error("Cloud Run devolvió marcas de despliegue ambiguas.");
  if (!completed.length) return null; // Existing installation without provenance: deploy once.
  const active = traffic.filter(target => Number(target.percent ?? 0) > 0);
  if (!active.length || active.some(target => !Number.isInteger(target.percent) || target.percent > 100 || !target.revisionName) ||
      active.reduce((sum, target) => sum + target.percent, 0) !== 100) {
    throw new Error("Cloud Run no devolvió una distribución de tráfico verificable.");
  }
  const revision = completed[0].revisionName;
  if (!revision || !/^[a-z][a-z0-9-]{0,61}[a-z0-9]$/.test(revision)) {
    throw new Error("Marca de despliegue con revisión inválida.");
  }
  // Never use latestReadyRevisionName, candidate tags or the mutable service labels.
  if (active.some(target => target.revisionName !== revision)) return null;
  return revision;
}
