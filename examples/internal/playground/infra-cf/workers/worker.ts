import { Resource } from "sst";

export default {
  async fetch(req: Request) {
    return new Response(
      JSON.stringify({ time: new Date().toISOString() }, null, 2)
    );
  },
};
