"use client";

import {AppSidebar} from "@/components/sidebar/sidebar"
import {
    SidebarInset,
    SidebarProvider,
} from "@e412/titanium"

export default function Page({children}) {
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
