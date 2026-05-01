import { Resource } from "sst";
import { createServer } from "http";

const server = createServer((req, res) => {
  res.writeHead(200, { "Content-Type": "application/json" });
  res.end(JSON.stringify({ bucket: Resource.MyBucket.name }));
});

server.listen(8080, () => {
  console.log(`Listening on :8080, bucket=${Resource.MyBucket.name}`);
});
