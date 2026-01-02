"use client";

import { useEffect } from 'react';
import { jwtDecode } from 'jwt-decode';
import { api } from '@/lib/api';
import { RefreshTokenRequest, RefreshTokenResponse } from '@/types/api';
import { useRouter } from 'next/navigation';

const TokenRefresher = () => {
    const router = useRouter();

    useEffect(() => {
        const checkToken = async () => {
            const authToken = localStorage.getItem('authToken');
            const refreshToken = localStorage.getItem('refreshToken');

            if (authToken && refreshToken) {
                try {
                    const decodedToken: { exp: number } = jwtDecode(authToken);
                    const isExpired = decodedToken.exp * 1000 < Date.now();

                    if (isExpired) {
                        console.log('Access token expired, refreshing...');
                        try {
                            const response = await api<RefreshTokenResponse, RefreshTokenRequest>('/auth/refresh', {
                                method: 'POST',
                                body: { refresh_token: refreshToken },
                                authenticate: false,
                            });

                            if (response.success && response.data?.access_token) {
                                localStorage.setItem('authToken', response.data.access_token);
                                console.log('Token refreshed successfully');
                            } else {
                                console.error('Failed to refresh token:', response.message);
                                // Logout if refresh fails
                                localStorage.removeItem('authToken');
                                localStorage.removeItem('refreshToken');
                                router.push('/login');
                            }
                        } catch (error) {
                            console.error('An error occurred during token refresh:', error);
                            // Logout if refresh fails
                            localStorage.removeItem('authToken');
                            localStorage.removeItem('refreshToken');
                            router.push('/login');
                        }
                    }
                } catch (error) {
                    console.error('Invalid token:', error);
                    // Logout if token is invalid
                    localStorage.removeItem('authToken');
                    localStorage.removeItem('refreshToken');
                    router.push('/login');
                }
            }
        };

        checkToken();
        // Set an interval to check the token periodically
        const interval = setInterval(checkToken, 60 * 1000); // Check every minute

        return () => clearInterval(interval);
    }, [router]); // Add router to dependency array

    return null;
};

export default TokenRefresher;
