import Image from "next/image";

export function Logo({ inverse = false }: { inverse?: boolean }) {
  return <span className={`logo ${inverse ? "inverse" : ""}`}><span className="logo-mark"><Image src="/assets/images/logo.png" alt="" width={36} height={36} priority /></span><strong>fudIA</strong><em>Admin</em></span>;
}
