/** @jsxImportSource react */
import { useEffect } from "react";
import {
  CardWrapper,
  FormInput,
  FormButton,
  FormAlert,
} from "../components/shared";
import { useAuth } from "@openauthjs/react/hooks";
import { LoaderProps } from "../types";

interface ErrorProps {
  type: string;
}

export interface LoginData {
  serverText: string;
}

export default function LoginPage({ serverText }: LoaderProps) {
  useEffect(() => {
    console.log("Server text is:", serverText);
  }, []);

  const { form, error } = useAuth();

  return (
    <CardWrapper>
      <form method="post" className="space-y-6">
        <div>
          <h2 className="text-3xl font-extrabold text-white tracking-tight">
            Welcome back
          </h2>
          <p className="text-slate-400 text-sm mt-2">
            Sign in to your account to continue
          </p>
        </div>

        {error && (
          <FormAlert
            title="Authentication failed"
            message={
              error.type === "invalid_password"
                ? "Incorrect password. Please try again."
                : error.type === "invalid_email"
                  ? "Invalid email address."
                  : "Invalid credentials."
            }
          />
        )}

        <div className="space-y-4">
          <FormInput
            label="Email Address"
            id="email"
            name="email"
            type="email"
            required
            defaultValue={form?.get("email")?.toString()}
            autoFocus={!error}
            placeholder="name@example.com"
          />

          <div>
            <div className="flex justify-between items-center mb-2">
              <label
                className="block text-slate-300 text-xs font-semibold uppercase tracking-wider"
                htmlFor="password"
              >
                Password
              </label>
              <a
                href="change"
                className="text-xs text-indigo-400 hover:text-indigo-300 font-medium transition-colors"
              >
                Forgot password?
              </a>
            </div>
            <input
              id="password"
              name="password"
              type="password"
              required
              autoFocus={error?.type === "invalid_password"}
              placeholder="********"
              className="w-full bg-slate-950/60 border border-slate-800 rounded-xl px-4 py-3 text-white placeholder-slate-600 focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-all font-medium text-sm"
            />
          </div>
        </div>

        <FormButton type="submit">Sign In</FormButton>

        <div className="text-center text-sm text-slate-400 mt-6">
          Don't have an account?{" "}
          <a
            href="register"
            className="text-indigo-400 hover:text-indigo-300 font-semibold transition-colors"
          >
            Create an account
          </a>
        </div>
      </form>
    </CardWrapper>
  );
}
