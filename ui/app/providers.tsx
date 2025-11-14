"use client";

import {ReactNode} from "react";
import {ThemeProvider} from "@e412/titanium";

export const Providers = ({children}: { children: ReactNode }) => {
    return (
        <ThemeProvider defaultTheme="dark" storageKey="titanium-theme">
            {children}
        </ThemeProvider>
    )
}