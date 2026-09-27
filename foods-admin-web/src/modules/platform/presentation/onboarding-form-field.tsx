"use client";

import type {ReactNode} from "react";

export function OnboardingFormField({label,error,className,children}:{label:string;error?:string;className?:string;children:ReactNode}){
  return <label className={["onboarding-field",className,error?"has-error":""].filter(Boolean).join(" ")}>
    <span>{label}</span>
    {children}
    {error&&<small className="field-error" role="alert">{error}</small>}
  </label>;
}
