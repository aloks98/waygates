"use client"

import { zodResolver } from "@hookform/resolvers/zod"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { useSearchParams } from 'next/navigation'
import Link from "next/link"
import { useRouter } from "next/navigation"
import { Suspense } from "react"

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
import {ErrorResponse, RegisterRequest, RegisterResponseData} from "@/types/api";

const formSchema = z.object({
    username: z.string().min(2, {
        message: "Username must be at least 2 characters.",
    }),
    email: z.string().email({
        message: "Please enter a valid email address.",
    }),
    password: z.string().min(8, {
        message: "Password must be at least 8 characters.",
    }),
    confirmPassword: z.string(),
}).refine((data) => data.password === data.confirmPassword, {
    message: "Passwords do not match.",
    path: ["confirmPassword"],
});

function SignupForm() {
    const router = useRouter()
    const searchParams = useSearchParams()
    const isNew = searchParams.get('new') === 'true'

    const form = useForm<z.infer<typeof formSchema>>({
        resolver: zodResolver(formSchema),
        defaultValues: {
            username: "",
            email: "",
            password: "",
            confirmPassword: "",
        },
    })

    async function onSubmit(values: z.infer<typeof formSchema>) {
        const registerData: RegisterRequest = {
            name: values.username, // Using username as name for now
            username: values.username,
            email: values.email,
            password: values.password,
        };

        try {
            const response = await api<RegisterResponseData, RegisterRequest>('/auth/register', {
                method: 'POST',
                body: registerData,
            });

            if (response.success) {
                router.push("/login");
            } else {
                console.error("Registration failed:", response.message);
                // TODO: Display error message to the user
            }
        } catch (error: unknown) {
            let errorMessage = "An unexpected error occurred during registration";
            try {
                const errorResponse: ErrorResponse = JSON.parse(error.message);
                errorMessage = errorResponse?.error?.message || errorMessage;
            } catch {
                // If parsing fails, use the default message
            }
            console.error(errorMessage);
            // TODO: Display error message to the user
        }
    }

    return (
        <div className="flex min-h-screen flex-col items-center justify-center px-6">
            <Card className="mx-auto max-w-lg">
                <CardHeader>
                    <CardTitle className="text-2xl">Sign Up</CardTitle>
                    <CardDescription>
                        {isNew
                            ? "No users found. Please create the first user."
                            : "Enter your information to create an account"}
                    </CardDescription>
                </CardHeader>
                <CardContent>
                    <Form {...form}>
                        <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-8">
                            <FormField
                                control={form.control}
                                name="username"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Username</FormLabel>
                                        <FormControl>
                                            <Input placeholder="your_username" {...field} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <FormField
                                control={form.control}
                                name="email"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Email</FormLabel>
                                        <FormControl>
                                            <Input placeholder="m@example.com" {...field} />
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
                            <FormField
                                control={form.control}
                                name="confirmPassword"
                                render={({ field }) => (
                                    <FormItem>
                                        <FormLabel>Confirm Password</FormLabel>
                                        <FormControl>
                                            <Input type="password" {...field} />
                                        </FormControl>
                                        <FormMessage />
                                    </FormItem>
                                )}
                            />
                            <Button type="submit" className="w-full">
                                Sign Up
                            </Button>
                        </form>
                    </Form>
                    <div className="mt-4 text-center text-sm">
                        Already have an account?{" "}
                        <Link href="/login" className="underline">
                            Login
                        </Link>
                    </div>
                </CardContent>
            </Card>
        </div>
    )
}

import Loader from '@/components/loader';

// ... (rest of the file)

export default function Signup() {
    return (
        <Suspense fallback={<Loader />}>
            <SignupForm />
        </Suspense>
    )
}
