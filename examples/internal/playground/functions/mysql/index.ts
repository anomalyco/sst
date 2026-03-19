import { Resource } from "sst";
import mysql from "mysql2/promise";

const connection = await mysql.createConnection({
  user: Resource.MyMysql.username,
  password: Resource.MyMysql.password,
  database: Resource.MyMysql.database,
  host: Resource.MyMysql.host,
  port: Resource.MyMysql.port,
});

export async function handler() {
  const res = await connection.execute("SELECT NOW() AS now");
  return {
    statusCode: 200,
    body: JSON.stringify(res[0]),
  };
}
