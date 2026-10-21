"use client";

import {RestaurantSetupPage} from "@/modules/setup";

/**
 * Compatibility wrapper for legacy configuration imports.
 * The setup flow was moved to modules/setup; keep this entry point
 * while older references are migrated.
 */
export function RestaurantSetupGuide(){
  return <RestaurantSetupPage/>;
}

export default RestaurantSetupGuide;
