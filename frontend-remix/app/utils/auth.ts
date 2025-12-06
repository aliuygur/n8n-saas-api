// Authentication utilities
import { API_BASE_URL } from "~/lib/config";

export function getToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('jwt_token');
}

export function setToken(token: string): void {
  if (typeof window === 'undefined') return;
  localStorage.setItem('jwt_token', token);
}

export function removeToken(): void {
  if (typeof window === 'undefined') return;
  localStorage.removeItem('jwt_token');
}

export function isAuthenticated(): boolean {
  return !!getToken();
}

export async function logout(): Promise<void> {
  const token = getToken();
  
  // Call logout endpoint
  try {
    await fetch(`${API_BASE_URL}/api/auth/logout`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${token}`,
      },
    });
  } catch (error) {
    console.error('Logout failed:', error);
  }
  
  // Remove token from localStorage
  removeToken();
  
  // Redirect to login
  window.location.href = '/login';
}
