export async function ping(): Promise<number> {
    try {
        const response = await fetch('https://api.github.com');
        return response.status;
    } catch (error) {
        if (error instanceof TypeError && error.message.includes('fetch failed')) {
            // Network error
            return 500;
        }
        throw error;
    }
}

