import { Resource } from "sst";

export default {
  async fetch(req: Request) {
    await Resource.MyDatabase.prepare(
      "CREATE TABLE IF NOT EXISTS todo (id INTEGER PRIMARY KEY, content TEXT)"
    ).run();
    const result = await Resource.MyDatabase.prepare(
      "SELECT id FROM todo ORDER BY id DESC LIMIT 1"
    ).first();
    const id = ((result?.id as number) ?? 0) + 1;
    await Resource.MyDatabase.prepare("INSERT INTO todo (id) VALUES (?1)")
      .bind(id)
      .run();
    return new Response(id.toString());
  },
};
