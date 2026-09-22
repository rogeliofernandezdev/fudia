import {apiFetch} from "@/shared/api/client";
import type {OrganizationSubscription} from "@/modules/platform";
import type {Location,MyProfile,MyProfileDraft,PermissionGroup,Role,RoleDraft,User,UserDraft} from "../domain/types";

export function listUsers(q:string,page:number,pageSize:number){
  const params=new URLSearchParams({q,page:String(page),pageSize:String(pageSize)});
  return apiFetch<{items:User[];total:number}>(`users?${params.toString()}`);
}

export function listRoles(){
  return apiFetch<{items:Role[];total:number}>("roles");
}

export function listLocations(){
  return apiFetch<{items:Location[]}>("locations?page=1&pageSize=100");
}

export function getPermissionCatalog(){
  return apiFetch<{groups:PermissionGroup[];menuGroups:PermissionGroup[]}>("permission-catalog");
}

export function saveUser(draft:UserDraft){
  return apiFetch(draft.id?`users/${draft.id}`:"users",{
    method:draft.id?"PATCH":"POST",
    body:JSON.stringify(draft),
  });
}

export function saveRole(draft:RoleDraft){
  return apiFetch(draft.id?`roles/${draft.id}`:"roles",{
    method:draft.id?"PATCH":"POST",
    body:JSON.stringify(draft),
  });
}

export function setUserActive(id:string,active:boolean){
  return apiFetch<void>(`users/${id}/status`,{method:"PATCH",body:JSON.stringify({active})});
}

export function setRoleActive(id:string,active:boolean){
  return apiFetch<void>(`roles/${id}/status`,{method:"PATCH",body:JSON.stringify({active})});
}

export function getMyProfile(){
  return apiFetch<MyProfile>("me");
}

export function saveMyProfile(draft:MyProfileDraft){
  return apiFetch<MyProfile>("me",{
    method:"PATCH",
    body:JSON.stringify(draft),
  });
}

export function getOrganizationSubscription(){
  return apiFetch<OrganizationSubscription>("subscription");
}
