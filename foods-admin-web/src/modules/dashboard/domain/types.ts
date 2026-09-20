export type DashboardKpi = {
  label: string;
  value: string;
  note: string;
  icon: string;
  tone: "green" | "blue" | "violet";
};

export type DashboardData = {
  kpis: DashboardKpi[];
};
