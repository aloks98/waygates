"use client";

import {AppSidebar} from "@/components/sidebar/sidebar"
import {
    SidebarInset,
    SidebarProvider,
} from "@e412/titanium"
import {ReactNode} from "react";

export default function Page({children}: { children: ReactNode }) {
    return (
        <SidebarProvider
        >
            <AppSidebar/>
            <SidebarInset>
                {children}
            </SidebarInset>
        </SidebarProvider>
    )
}
