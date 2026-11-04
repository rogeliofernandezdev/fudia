import {Status} from "@/design-system";
import {fiscalStatusView} from "../domain/fiscal";
import type {FiscalStatus as FiscalStatusValue} from "../domain/types";

/** Fiscal document state with the central tone and label mapping. */
export function FiscalStatus({status}:{status:FiscalStatusValue}){
 const view=fiscalStatusView[status]??fiscalStatusView.PENDING;
 return <Status tone={view.tone}>{view.label}</Status>;
}
