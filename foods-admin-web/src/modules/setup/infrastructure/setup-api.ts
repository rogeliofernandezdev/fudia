import {apiFetch} from "@/shared/api/client";
import type {RestaurantSetup,ServiceMode} from "../domain/types";

export function getRestaurantSetup(){
  return apiFetch<RestaurantSetup>("restaurant-setup");
}

export function updateRestaurantServiceMode(serviceMode:Exclude<ServiceMode,"">){
  return apiFetch<RestaurantSetup>("restaurant-setup",{
    method:"PATCH",
    body:JSON.stringify({serviceMode}),
  });
}

export function completeRestaurantSetup(){
  return apiFetch<RestaurantSetup>("restaurant-setup/complete",{method:"POST"});
}
