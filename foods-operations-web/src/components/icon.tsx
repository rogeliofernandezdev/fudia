type IconName =
  | "home" | "tables" | "pos" | "kitchen" | "cookingPot" | "orders" | "cash"
  | "bell" | "chevron" | "search" | "user" | "lock" | "eye"
  | "store" | "clock" | "plus" | "minus" | "trash" | "check"
  | "printer" | "whatsapp" | "bike" | "receipt" | "logout"
  | "wifi" | "filter" | "card" | "wallet" | "menu" | "close" | "fingerprint" | "refresh";

const paths: Record<IconName, React.ReactNode> = {
  home: <><path d="m3 11 9-8 9 8"/><path d="M5 10v10h14V10M9 20v-6h6v6"/></>,
  tables: <><rect x="4" y="5" width="16" height="12" rx="2"/><path d="M8 17v3m8-3v3M4 10h16M9 5v5m6-5v5"/></>,
  pos: <><rect x="3" y="4" width="18" height="16" rx="3"/><path d="M7 8h10M8 12h.01M12 12h.01M16 12h.01M8 16h.01M12 16h4"/></>,
  kitchen: <><path d="M6 3v7a3 3 0 0 0 3 3V3M6 7h3M9 13v8M16 3c-2 3-2 7 1 9v9M17 12h2V3"/></>,
  cookingPot: <><path d="M5 10h14l-1 9H6zM4 10h16M8 10V8h8v2M3 13h2m14 0h2"/><path d="M8 5c1-1 1-2 0-3M12 5c1-1 1-2 0-3M16 5c1-1 1-2 0-3"/></>,
  orders: <><path d="M7 3h10v4H7zM5 5H4v16h16V5h-1M8 12h8M8 16h5"/></>,
  cash: <><rect x="3" y="6" width="18" height="13" rx="2"/><path d="M3 10h18M7 15h3"/></>,
  bell: <><path d="M18 8a6 6 0 0 0-12 0c0 7-3 7-3 9h18c0-2-3-2-3-9M10 21h4"/></>,
  chevron: <path d="m9 18 6-6-6-6"/>,
  search: <><circle cx="11" cy="11" r="7"/><path d="m20 20-4-4"/></>,
  user: <><circle cx="12" cy="8" r="4"/><path d="M4 21a8 8 0 0 1 16 0"/></>,
  lock: <><rect x="4" y="10" width="16" height="11" rx="2"/><path d="M8 10V7a4 4 0 0 1 8 0v3"/></>,
  eye: <><path d="M2 12s4-7 10-7 10 7 10 7-4 7-10 7S2 12 2 12"/><circle cx="12" cy="12" r="2.5"/></>,
  store: <><path d="M4 10v11h16V10M3 10l2-7h14l2 7"/><path d="M3 10a3 3 0 0 0 5 2 3 3 0 0 0 4 0 3 3 0 0 0 4 0 3 3 0 0 0 5-2M9 21v-5h6v5"/></>,
  clock: <><circle cx="12" cy="12" r="9"/><path d="M12 7v5l3 2"/></>,
  plus: <path d="M12 5v14M5 12h14"/>,
  minus: <path d="M5 12h14"/>,
  trash: <><path d="M4 7h16M9 7V4h6v3M7 7l1 14h8l1-14M10 11v6m4-6v6"/></>,
  check: <path d="m5 12 4 4L19 6"/>,
  printer: <><path d="M7 9V3h10v6M7 18H5a2 2 0 0 1-2-2v-5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2v5a2 2 0 0 1-2 2h-2"/><path d="M7 14h10v7H7z"/></>,
  whatsapp: <><path d="M20 11.5a8.5 8.5 0 0 1-12.5 7.5L3 20l1.2-4A8.5 8.5 0 1 1 20 11.5Z"/><path d="M8 8c1 4 4 7 8 8"/></>,
  bike: <><circle cx="6" cy="17" r="3"/><circle cx="18" cy="17" r="3"/><path d="m6 17 4-8h4l4 8M9 12h7M12 17l-3-5-2-2"/></>,
  receipt: <><path d="M6 3h12v19l-3-2-3 2-3-2-3 2zM9 8h6M9 12h6M9 16h3"/></>,
  logout: <><path d="M10 5H5v14h5M14 8l4 4-4 4M18 12H9"/></>,
  wifi: <><path d="M5 12a10 10 0 0 1 14 0M8 15a6 6 0 0 1 8 0M11 18a2 2 0 0 1 2 0"/></>,
  filter: <path d="M4 5h16l-6 7v6l-4 2v-8z"/>,
  card: <><rect x="3" y="5" width="18" height="14" rx="2"/><path d="M3 10h18"/></>,
  wallet: <><path d="M4 6h14a2 2 0 0 1 2 2v11H4a2 2 0 0 1-2-2V7a3 3 0 0 1 3-3h12"/><path d="M15 12h5"/></>,
  menu: <path d="M4 7h16M4 12h16M4 17h16"/>,
  close: <path d="M6 6l12 12M18 6 6 18"/>,
  fingerprint: <><path d="M12 11a2 2 0 0 1 2 2c0 3.5-.8 6.2-2 8"/><path d="M8.2 20c1.1-2.3 1.8-4.5 1.8-7a2 2 0 0 1 4 0"/><path d="M6 17.5c.6-1.7 1-3.2 1-4.5a5 5 0 0 1 10 0c0 2.9-.5 5.5-1.5 7.7"/><path d="M4.2 14.5A7.8 7.8 0 0 1 4 13a8 8 0 0 1 16 0c0 2.1-.2 4.2-.8 6"/><path d="M6.6 5.4A8 8 0 0 1 12 3"/></>,
  refresh: <><path d="M20 11a8 8 0 1 0-2.34 5.66"/><path d="M20 4v7h-7"/></>,
};

export function Icon({ name, size = 20, className = "" }: { name: IconName; size?: number; className?: string }) {
  return <svg className={className} width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">{paths[name]}</svg>;
}
