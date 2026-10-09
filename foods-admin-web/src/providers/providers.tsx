"use client";
import {QueryClient,QueryClientProvider} from "@tanstack/react-query";import {useEffect,useState} from "react";
import {FeedbackProvider} from "./feedback-provider";
import {listenForSessionChange} from "@/shared/session/session-events";
export function Providers({children}:{children:React.ReactNode}){const[client]=useState(()=>new QueryClient({defaultOptions:{queries:{staleTime:20_000,retry:1}}}));useEffect(()=>listenForSessionChange(()=>{client.clear();window.location.assign("/login")}),[client]);return <QueryClientProvider client={client}><FeedbackProvider>{children}</FeedbackProvider></QueryClientProvider>}
