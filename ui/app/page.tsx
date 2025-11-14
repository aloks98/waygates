import { redirect } from 'next/navigation';
import { SuccessResponse, StatusResponseData } from '@/types/api';

async function checkUserExists() {
    try {
        const res = await fetch('http://localhost:8080/api/status', { cache: 'no-store' });
        const json: SuccessResponse<StatusResponseData> = await res.json();

        if (json.success && json.data) {
            return json.data.user_setup_complete;
        } else {
            // If API call is successful but doesn't indicate success or data,
            // assume user does not exist or there's an issue, redirect to signup
            return false;
        }
    } catch (error) {
        console.error("Failed to check if user exists:", error);
        // Default to login on error to prevent infinite redirect loop if backend is down
        return true;
    }
}

export default async function Home() {
    const userExists = await checkUserExists();

    if (userExists) {
        redirect('/login');
    } else {
        redirect('/signup?new=true');
    }
}
