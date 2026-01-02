import { redirect } from 'next/navigation';
import { SuccessResponse, StatusResponseData } from '@/types/api';

export const dynamic = 'force-dynamic';

async function checkUserExists() {
    let userSetupComplete = false; // Initialize with a default value
    try {
        const res = await fetch('http://localhost:8080/api/status', { cache: 'no-store' });
        const json: SuccessResponse<StatusResponseData> = await res.json();

        if (json.success && json.data) {
            userSetupComplete = json.data.user_setup_complete;
        } else {
            // If API call is successful but doesn't indicate success or data,
            // assume user does not exist or there's an issue, redirect to signup
            userSetupComplete = false;
        }
    } catch (error) {
        console.error("Failed to check if user exists:", error);
        // Default to log in on error to prevent infinite redirect loop if backend is down
        userSetupComplete = true; // Assuming true means redirect to login
    }
    return userSetupComplete;
}

export default async function Home() {
    const userExists = await checkUserExists();

    if (userExists) {
        redirect('/login');
    } else {
        redirect('/signup?new=true');
    }
}
