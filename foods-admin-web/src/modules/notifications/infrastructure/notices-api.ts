import {apiFetch} from "@/shared/api/client";
import type {NoticeFeed} from "../domain/types";

export function getNotices():Promise<NoticeFeed>{return apiFetch<NoticeFeed>("notices")}
export function markNoticeRead(id:string):Promise<void>{return apiFetch<void>(`notices/${encodeURIComponent(id)}/read`,{method:"POST"})}
export function markAllNoticesRead():Promise<void>{return apiFetch<void>("notices/read",{method:"POST"})}
