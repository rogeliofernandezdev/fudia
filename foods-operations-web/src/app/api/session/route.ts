import { cookies } from "next/headers";
import { NextResponse } from "next/server";

export async function DELETE() {
  const cookieStore = await cookies();
  const session = cookieStore.get("foods_session")?.value;
  if (session) {
    const api = process.env.FOODS_API_URL ?? "http://localhost:8080";
    try {
      await fetch(`${api}/v1/auth/logout`, {
        method: "POST",
        headers: { Cookie: `foods_session=${session}` },
        cache: "no-store",
      });
    } catch {
      // Local sign-out remains available if the API is temporarily unreachable.
    }
  }
  cookieStore.set("foods_session", "", {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: 0,
  });
  return NextResponse.json({ signedOut: true });
}
