import mysql from "mysql2/promise";
import { Resource } from "sst";

const connection = await mysql.createConnection({
  host: Resource.MyDatabase.host,
  port: Resource.MyDatabase.port,
  user: Resource.MyDatabase.username,
  password: Resource.MyDatabase.password,
  database: Resource.MyDatabase.database,
});

export async function handler() {
  const [rows] = await connection.query("SELECT NOW()");
  return {
    statusCode: 200,
    body: `Querying ${Resource.MyDatabase.host}\n\n` + rows[0].now,
  };
}
