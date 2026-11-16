"use client";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";import {useEffect,useState} from "react";
import {FeedbackProvider} from "./feedback-provider";
import {listenForSessionChange} from "@/shared/session/session-events";
import {TooltipLayer} from "@/design-system/tooltip-layer";
import {NumberInputBehavior} from "@/design-system/number-input-behavior";
import {PayCollectionProvider,PayNoticeLayer} from "@/modules/payment-devices";
export function Providers({children}:{children:React.ReactNode}){const[client]=useState(()=>new QueryClient({defaultOptions:{queries:{staleTime:20_000,retry:1}}}));useEffect(()=>listenForSessionChange(()=>{client.clear();window.location.assign("/login")}),[client]);return <QueryClientProvider client={client}><FeedbackProvider><PayCollectionProvider>{children}<PayNoticeLayer/><TooltipLayer/><NumberInputBehavior/></PayCollectionProvider></FeedbackProvider></QueryClientProvider>}
