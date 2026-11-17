/** @jsxImportSource react */
import {
  CardWrapper,
  FormInput,
  FormButton,
  FormAlert,
} from "../components/shared";
import { useAuth } from "@openauthjs/react/hooks";
import { useEffect } from "react";
import { LoaderProps } from "../types";

export default function RegisterPage({ serverText }: LoaderProps) {
  useEffect(() => {
    console.log("Server text is:", serverText);
  }, []);

  const { state = { type: "start" }, error, form } = useAuth();

  const errorType = (error?.type as string) || "";
  const errorMessage = (error?.message as string) || "";

  // Otherwise render Register page
  const emailError = ["invalid_email", "email_taken"].includes(errorType || "");
  const passwordError = [
    "invalid_password",
    "password_mismatch",
    "validation_error",
  ].includes(errorType);

  return (
    <CardWrapper>
      <div className="space-y-6">
        <div>
          <h2 className="text-3xl font-extrabold text-white tracking-tight">
            Create account
          </h2>
          <p className="text-slate-400 text-sm mt-2">
            Get started with your secure account
          </p>
        </div>

        {error && (
          <FormAlert
            title="Registration error"
            message={
              errorType === "email_taken"
                ? "An account with this email already exists."
                : errorType === "password_mismatch"
                  ? "Passwords do not match."
                  : errorType === "validation_error"
                    ? errorMessage || "Password does not meet requirements."
                    : "Could not register your account. Please try again."
            }
          />
        )}

        {state.type === "start" ? (
          <form method="post" className="space-y-4">
            <input type="hidden" name="action" value="register" />

            <FormInput
              label="Email Address"
              id="email"
              name="email"
              type="email"
              required
              defaultValue={!emailError ? form?.get("email")?.toString() : ""}
              autoFocus={!error || emailError}
              placeholder="name@example.com"
            />

            <FormInput
              label="Password"
              id="password"
              name="password"
              type="password"
              required
              autoFocus={passwordError}
              placeholder="********"
            />

            <FormInput
              label="Confirm Password"
              id="repeat"
              name="repeat"
              type="password"
              required
              placeholder="********"
            />

            <FormButton type="submit">Register</FormButton>

            <div className="text-center text-sm text-slate-400 mt-6">
              Already have an account?{" "}
              <a
                href="authorize"
                className="text-indigo-400 hover:text-indigo-300 font-semibold transition-colors"
              >
                Sign in
              </a>
            </div>
          </form>
        ) : (
          <form method="post" className="space-y-4">
            <input type="hidden" name="action" value="verify" />
            <div className="text-center bg-indigo-500/5 border border-indigo-500/10 rounded-2xl p-6 mb-4">
              <span className="text-slate-400 text-sm">
                We've sent a 6-digit confirmation code to
              </span>
              <p className="text-indigo-300 font-semibold mt-1 text-sm">
                {state.email as string}
              </p>
            </div>

            <div>
              <label
                className="block text-slate-300 text-xs font-semibold uppercase tracking-wider mb-2 text-center"
                htmlFor="code"
              >
                Verification Code
              </label>
              <input
                id="code"
                name="code"
                type="text"
                required
                autoFocus
                maxLength={6}
                minLength={6}
                placeholder="000000"
                className="w-full bg-slate-950/60 border border-slate-800 rounded-xl px-4 py-3 text-white placeholder-slate-600 focus:outline-none focus:border-indigo-500 focus:ring-1 focus:ring-indigo-500 transition-all font-semibold tracking-[0.5em] text-center text-lg"
              />
            </div>

            <FormButton type="submit">Verify Code</FormButton>
          </form>
        )}
      </div>
    </CardWrapper>
  );
}
