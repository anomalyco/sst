import { query } from "$app/server";

export const getArbitraryData = query(async () => {
  await new Promise((resolve) => setTimeout(resolve, 2000));
  return ["foo", "bar", "baz"];
});
