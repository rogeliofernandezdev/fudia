import type {Metadata} from "next";
import {SalonManager} from "@/modules/operations";
export const metadata:Metadata={title:"Salón"};
export default function Page(){return <SalonManager/>}
