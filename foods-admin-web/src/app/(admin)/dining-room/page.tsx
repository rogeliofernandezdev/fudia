import type {Metadata} from "next";
import {SalonManager} from "@/modules/operations";
export const metadata:Metadata={title:"Salón — fudIA Admin"};
export default function Page(){return <SalonManager/>}
