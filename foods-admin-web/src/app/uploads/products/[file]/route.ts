// Infrastructure proxy: product photos keep their existing same-origin URL.
const fileName = /^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}\.(?:png|jpg|jpeg|webp)$/i;
const imageTypes = new Set(["image/png", "image/jpeg", "image/webp"]);

async function proxy(request: Request, {params}: {params: Promise<{file: string}>}) {
  const {file} = await params;
  if (!fileName.test(file)) return new Response(null, {status: 404});
  try {
    const api = process.env.FOODS_API_URL ?? "http://localhost:8080";
    const response = await fetch(new URL(`/uploads/products/${file}`, api), {
      method: request.method, cache: "no-store", redirect: "error", signal: AbortSignal.timeout(15000),
    });
    if (!response.ok) return new Response(null, {status: response.status === 404 ? 404 : 502});
    const contentType = response.headers.get("content-type")?.split(";")[0];
    if (!contentType || !imageTypes.has(contentType)) return new Response(null, {status: 502});
    return new Response(request.method === "HEAD" ? null : response.body, {
      headers: {"Content-Type": contentType, "Cache-Control": "no-cache", "X-Content-Type-Options": "nosniff"},
    });
  } catch {
    return new Response(null, {status: 503});
  }
}

export const GET = proxy;
export const HEAD = proxy;
