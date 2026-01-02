"use client";

import {ReactNode} from "react";
import {ThemeProvider} from "@e412/titanium";
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import TokenRefresher from '@/components/token-refresher';

const queryClient = new QueryClient();

export const Providers = ({children}: { children: ReactNode }) => {
    return (
        <QueryClientProvider client={queryClient}>
            <TokenRefresher />
            <ThemeProvider defaultTheme="dark" storageKey="titanium-theme">
                {children}
            </ThemeProvider>
        </QueryClientProvider>
    )
}