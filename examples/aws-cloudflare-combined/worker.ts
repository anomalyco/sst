export default {
  async fetch(req: Request) {
    return new Response("Hello from Cloudflare Worker!");
  },
};
