import type {MeasurementUnit} from "@/shared/units/measurement-unit";
export type InventoryUnit=MeasurementUnit&{id:string;countable:boolean};
export type InventoryUnitDraft={code:string;name:string};
