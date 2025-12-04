// Example usage of the Google Auth API endpoints from frontend

// 1. Initiate Google Login
async function loginWithGoogle() {
  try {
    const response = await fetch('http://localhost:4000/api/auth/google/login');
    const data = await response.json();
    
    // Redirect user to Google OAuth
    window.location.href = data.auth_url;
  } catch (error) {
    console.error('Login failed:', error);
  }
}

// 2. Handle OAuth Callback (in your callback page)
async function handleGoogleCallback() {
  // Extract code and state from URL
  const urlParams = new URLSearchParams(window.location.search);
  const code = urlParams.get('code');
  const state = urlParams.get('state');
  
  if (!code) {
    console.error('No authorization code received');
    return;
  }
  
  try {
    const response = await fetch('http://localhost:4000/api/auth/google/callback', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ code, state }),
    });
    
    const data = await response.json();
    
    // Store session token
    localStorage.setItem('session_token', data.session_token);
    localStorage.setItem('user', JSON.stringify(data.user));
    
    // Redirect to dashboard or home
    window.location.href = '/dashboard';
  } catch (error) {
    console.error('Callback handling failed:', error);
  }
}

// 3. Get Current User
async function getCurrentUser() {
  const token = localStorage.getItem('session_token');
  
  if (!token) {
    throw new Error('Not authenticated');
  }
  
  try {
    const response = await fetch('http://localhost:4000/api/auth/me', {
      headers: {
        'Authorization': token,
      },
    });
    
    if (!response.ok) {
      throw new Error('Failed to get user');
    }
    
    const data = await response.json();
    return data.user;
  } catch (error) {
    console.error('Get user failed:', error);
    // Clear invalid token
    localStorage.removeItem('session_token');
    throw error;
  }
}

// 4. Logout
async function logout() {
  const token = localStorage.getItem('session_token');
  
  if (!token) {
    return;
  }
  
  try {
    await fetch('http://localhost:4000/api/auth/logout', {
      method: 'POST',
      headers: {
        'Authorization': token,
      },
    });
  } catch (error) {
    console.error('Logout failed:', error);
  } finally {
    // Always clear local storage
    localStorage.removeItem('session_token');
    localStorage.removeItem('user');
    window.location.href = '/';
  }
}

// 5. Make Authenticated API Calls
async function makeAuthenticatedRequest(endpoint, options = {}) {
  const token = localStorage.getItem('session_token');
  
  if (!token) {
    throw new Error('Not authenticated');
  }
  
  const headers = {
    ...options.headers,
    'Authorization': token,
  };
  
  const response = await fetch(endpoint, {
    ...options,
    headers,
  });
  
  if (response.status === 401 || response.status === 403) {
    // Token expired or invalid
    localStorage.removeItem('session_token');
    window.location.href = '/login';
    throw new Error('Session expired');
  }
  
  return response;
}

// Example: Using authenticated request
async function createN8NInstance(instanceData) {
  try {
    const response = await makeAuthenticatedRequest(
      'http://localhost:4000/provisioning/CreateInstance',
      {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify(instanceData),
      }
    );
    
    return await response.json();
  } catch (error) {
    console.error('Create instance failed:', error);
    throw error;
  }
}

export {
  loginWithGoogle,
  handleGoogleCallback,
  getCurrentUser,
  logout,
  makeAuthenticatedRequest,
  createN8NInstance,
};
