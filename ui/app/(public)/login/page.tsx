"use client"

import { zodResolver } from "@hookform/resolvers/zod"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { useRouter } from "next/navigation"
import Link from "next/link"

import { Button } from "@e412/titanium"
import {
    Form,
    FormControl,
    FormField,
    FormItem,
    FormLabel,
    FormMessage,
} from "@e412/titanium"
import { Input } from "@e412/titanium"
import {
    Card,
    CardContent,
    CardDescription,
    CardHeader,
    CardTitle,
} from "@e412/titanium"

import { api } from "@/lib/api";
import { LoginRequest, LoginResponseData, ErrorResponse } from "@/types/api";
import { useMutation } from '@tanstack/react-query';

const formSchema = z.object({
    identifier: z.string().refine(value => {
        // Check if it's a valid email
        if (z.string().email().safeParse(value).success) {
            return true;
        }
        // Otherwise, check if it's a non-empty string (username)
        return value.trim().length > 0;
    }, {
        message: "Please enter a valid email address or username.",
    }),
    password: z.string().min(8, {
        message: "Password must be at least 8 characters.",
    }),
})

export default function Login() {
    const router = useRouter()
    const form = useForm<z.infer<typeof formSchema>>({
        resolver: zodResolver(formSchema),
        defaultValues: {
            identifier: "",
            password: "",
        },
    })

    const loginMutation = useMutation({
        mutationFn: async (loginData: LoginRequest) => {
            return api<LoginResponseData, LoginRequest>('/auth/login', {
                method: 'POST',
                body: loginData,
                authenticate: false,
            });
        },
        onSuccess: (response) => {
            if (response.success && response.data?.access_token) {
                localStorage.setItem('authToken', response.data.access_token);
                localStorage.setItem('refreshToken', response.data.refresh_token);
                console.log("Login successful, token saved and redirecting to /proxies");
                router.push("/proxies");
            } else {
                console.error("Login failed:", response.message || "Unknown error");
                // TODO: Display error message to the user
            }
        },
        onError: (error: Error) => {
            let errorMessage = "An unexpected error occurred during login";
            try {
                const errorResponse: ErrorResponse = JSON.parse(error.message);
                errorMessage = errorResponse?.error?.message || errorMessage;
            } catch {
                // If parsing fails, use the default message
            }
            console.error(errorMessage);
            // TODO: Display error message to the user using a toast notification
        },
    });

    function onSubmit(values: z.infer<typeof formSchema>) {
        const loginData: LoginRequest = {
            identifier: values.identifier,
            password: values.password,
        };
        loginMutation.mutate(loginData);
    }

    return (
        <div className="flex min-h-screen flex-col items-center justify-center px-6">
            <Card className="mx-auto max-w-lg">
                <CardHeader>
                    <CardTitle className="text-2xl">Login</CardTitle>
                    <CardDescription>
                        Enter your username or email below to login to your account
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <Form {...form}>
                        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-8">
                            <FormField
                                control={form.control}
                                name="identifier"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Username or Email</FormLabel>
                                        <FormControl>
                                            <Input placeholder="username or m@example.com" {...field} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name="password"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Password</FormLabel>
                                        <FormControl>
                                            <Input type="password" {...field} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <Button type="submit" className="w-full" disabled={loginMutation.isPending}>
                                {loginMutation.isPending ? 'Logging in...' : 'Login'}
                            </Button>
                        </form>
                    </Form>
                    <div className="mt-4 text-center text-sm">
                        Don&apos;t have an account?{" "}
                        <Link href="/signup" className="underline">
                            Sign up
                        </Link>
                    </div>
                </CardContent>
            </Card>
        </div>
    )
}
