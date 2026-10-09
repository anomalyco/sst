/** @jsxImportSource react */
import { useAuth } from "@openauthjs/react/hooks";
import {
  CardWrapper,
  FormInput,
  FormButton,
  FormAlert,
} from "../components/shared";
import { useEffect } from "react";
import { LoaderProps } from "../types";

export interface ChangeData {
  serverText: string;
}

export default function ChangePage({ serverText }: LoaderProps) {
  useEffect(() => {
    console.log("Server text is:", serverText);
  }, []);

  const { state = { type: "start" }, error, form } = useAuth();

  const errorType = (error?.type as string) || "";
  const errorMessage = (error?.message as string) || "";

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
            Reset Password
          </h2>
          <p className="text-slate-400 text-sm mt-2">
            Restore access to your account
          </p>
        </div>

        {error && (
          <FormAlert
            title="Reset failed"
            message={
              errorType === "password_mismatch"
                ? "Passwords do not match."
                : errorType === "validation_error"
                  ? errorMessage || "Password does not meet requirements."
                  : errorType === "invalid_code"
                    ? "Invalid verification code."
                    : "An error occurred. Please try again."
            }
          />
        )}

        {state.type === "start" && (
          <form method="post" className="space-y-4">
            <input type="hidden" name="action" value="code" />
            <FormInput
              label="Confirm your Email"
              id="email"
              name="email"
              type="email"
              required
              defaultValue={form?.get("email")?.toString() || ""}
              autoFocus
              placeholder="name@example.com"
            />

            <FormButton type="submit">Send Reset Code</FormButton>

            <div className="text-center text-sm text-slate-400 mt-6">
              Remembered your password?{" "}
              <a
                href="authorize"
                className="text-indigo-400 hover:text-indigo-300 font-semibold transition-colors"
              >
                Sign in
              </a>
            </div>
          </form>
        )}

        {state.type === "code" && (
          <div className="space-y-4">
            <form method="post" className="space-y-4">
              <input type="hidden" name="action" value="verify" />
              <div className="text-center bg-indigo-500/5 border border-indigo-500/10 rounded-2xl p-6 mb-4">
                <span className="text-slate-400 text-sm">
                  We've sent a recovery code to your email.
                </span>
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

            <form method="post" className="text-center">
              <input type="hidden" name="action" value="code" />
              <input
                type="hidden"
                name="email"
                value={(state.email as string) || ""}
              />
              <div className="flex justify-between items-center text-xs mt-6 px-1">
                <a
                  href="authorize"
                  className="text-slate-400 hover:text-slate-200 transition-colors"
                >
                  Back to login
                </a>
                <button
                  type="submit"
                  className="text-indigo-400 hover:text-indigo-300 font-semibold transition-colors"
                >
                  Resend Code
                </button>
              </div>
            </form>
          </div>
        )}

        {state.type === "update" && (
          <form method="post" className="space-y-4">
            <input type="hidden" name="action" value="update" />

            <FormInput
              label="New Password"
              id="password"
              name="password"
              type="password"
              required
              autoFocus={passwordError}
              placeholder="********"
            />

            <FormInput
              label="Confirm New Password"
              id="repeat"
              name="repeat"
              type="password"
              required
              placeholder="********"
            />

            <FormButton type="submit">Update Password</FormButton>
          </form>
        )}
      </div>
    </CardWrapper>
  );
}
