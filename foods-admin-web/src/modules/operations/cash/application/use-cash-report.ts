import {useQuery} from "@tanstack/react-query";
import {useSession} from "@/providers";
import {getCashShiftReport} from "../infrastructure/cash-api";

export function useCashReport(id:string,enabled=true){
 const {organization,location}=useSession();
 return useQuery({queryKey:["cash-report",organization?.id,location?.id,id],queryFn:()=>getCashShiftReport(id),enabled,retry:false,staleTime:0,refetchOnMount:"always",refetchOnWindowFocus:false});
}
