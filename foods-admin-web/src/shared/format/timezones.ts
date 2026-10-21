export type TimezoneOption={value:string;label:string};

/** Keep IANA values intact, including aliases not listed by the browser. */
export function timezoneOptions(current:string):TimezoneOption[]{
  let zones:string[]=[];
  try{zones=Intl.supportedValuesOf("timeZone")}catch{/* Older engines keep the selected zone and UTC. */}
  return [...new Set([current,"UTC",...zones].filter(Boolean))]
    .map(value=>({value,label:value.replaceAll("_"," ")}));
}
