export type User = {
  id: number;
  name: string;
  email: string;
};

export const DEPLOY_URL =
  process.env.NODE_ENV === "development"
    ? "http://localhost:3000"
    : "https://tanstack.playground.sst.sh";
