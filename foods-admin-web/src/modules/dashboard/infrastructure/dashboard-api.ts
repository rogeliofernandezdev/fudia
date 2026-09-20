import {apiFetch} from "@/shared/api/client";
import type {DashboardData} from "../domain/types";

export function getDashboard(): Promise<DashboardData> {
  return apiFetch<DashboardData>("dashboard");
}
