export function Logo({ inverse = false }: { inverse?: boolean }) {
  return <span className={inverse ? "logo inverse" : "logo"}><span className="logo-symbol"><i /><b /></span><strong>fudIA</strong></span>;
}
