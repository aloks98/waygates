import type {Metadata} from "next";
import "./globals.css";
import {Providers} from "@/app/providers";
import {ReactNode} from "react";

export const metadata: Metadata = {
    title: "Waygate Proxy",
    description: "A secure and efficient proxy solution.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: ReactNode;
}>) {
    return (
        <html lang="en">
        <body>
        <Providers>
            {children}
        </Providers>

        </body>
        </html>
    );
}
