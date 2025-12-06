import { useEffect } from "react";
import { useNavigate, useSearchParams } from "react-router";

export default function AuthCallback() {
  const [searchParams] = useSearchParams();
  const navigate = useNavigate();

  useEffect(() => {
    const token = searchParams.get("token");
    const error = searchParams.get("error");

    if (error) {
      // Redirect to login with error
      navigate(`/login?error=${error}`);
      return;
    }

    if (token) {
      // Store JWT token in localStorage
      localStorage.setItem("jwt_token", token);
      
      // Redirect to dashboard
      navigate("/dashboard");
    } else {
      // No token, redirect to login
      navigate("/login?error=no_token");
    }
  }, [searchParams, navigate]);

  return (
    <div className="min-h-screen bg-gray-950 flex items-center justify-center">
      <div className="text-center">
        <div className="inline-block animate-spin rounded-full h-12 w-12 border-b-2 border-indigo-500 mb-4"></div>
        <p className="text-gray-400">Completing authentication...</p>
      </div>
    </div>
  );
}
