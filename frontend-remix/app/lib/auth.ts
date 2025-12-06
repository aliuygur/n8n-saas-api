// Helper function to get JWT token from cookies
export function getJWTToken(): string | null {
  const cookies = document.cookie.split(';');
  for (const cookie of cookies) {
    const [name, value] = cookie.trim().split('=');
    if (name === 'jwt') {
      return value;
    }
  }
  return null;
}

// Helper function to make authenticated API requests
export async function authenticatedFetch(url: string, options: RequestInit = {}): Promise<Response> {
  const token = getJWTToken();
  
  const headers = new Headers(options.headers);
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  
  return fetch(url, {
    ...options,
    headers,
    credentials: 'include', // Still include credentials for cookie management
  });
}
