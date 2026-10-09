"use client";

import {RestaurantSetupTour} from "@/modules/setup";

/**
 * Compatibility wrapper for legacy configuration imports.
 * The setup flow was moved to modules/setup.
 */
export function RestaurantSetupTourGuide(){
  return <RestaurantSetupTour />;
}

export default RestaurantSetupTourGuide;
