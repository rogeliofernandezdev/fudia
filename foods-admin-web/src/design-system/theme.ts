/** Resolve tokens for APIs such as Canvas and Mapbox that cannot consume var(). */
export function readCssToken(name: `--${string}`): string {
  const value = getComputedStyle(document.body).getPropertyValue(name).trim();
  if (!value) throw new Error(`Missing design token: ${name}`);
  return value;
}
