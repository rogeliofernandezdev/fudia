import {apiFetch} from "@/shared/api/client";
import type {DashboardData,DashboardPeriodKey} from "../domain/types";

export function getDashboard(period:DashboardPeriodKey="today"): Promise<DashboardData> {
  return apiFetch<DashboardData>(`dashboard?period=${period}`);
}
