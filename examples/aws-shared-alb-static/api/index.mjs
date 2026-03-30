import express from "express";

const PORT = 3000;

const app = express();

app.get("/", (req, res) => {
  res.json({ status: "ok" });
});

app.get("/api", (req, res) => {
  res.json({ service: "api", message: "Hello from the API service" });
});

app.get("/api/health", (req, res) => {
  res.json({ status: "ok" });
});

app.get("/api/greeting", (req, res) => {
  res.json({ service: "api", message: "Hello from the API service" });
});

app.get("/api/*", (req, res) => {
  res.json({ service: "api", path: req.path });
});

app.listen(PORT, () => {
  console.log(`API service is running on http://localhost:${PORT}`);
});
