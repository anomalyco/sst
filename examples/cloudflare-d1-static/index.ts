import { Resource } from "sst";

export default {
  async fetch(request: Request) {
    if (new URL(request.url).pathname === "/favicon.ico")
      return new Response(null, { status: 404 });

    await Resource.MyDatabase.prepare("CREATE TABLE IF NOT EXISTS todo (id INTEGER PRIMARY KEY AUTOINCREMENT)").run();

    await Resource.MyDatabase.prepare("INSERT INTO todo DEFAULT VALUES").run();

    const { count } = await Resource.MyDatabase.prepare("SELECT COUNT(*) as count FROM todo").first<{ count: number }>();

    return new Response(`Total todos: ${count}`);
  },
};
