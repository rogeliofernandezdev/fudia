import {PlatformCompanyDetailPage} from "@/modules/platform";

export default async function Page({params}:{params:Promise<{id:string}>}){
  const {id}=await params;
  return <PlatformCompanyDetailPage organizationId={id}/>;
}
