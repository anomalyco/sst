import express from "express";

const PORT = 3000;

const app = express();

app.get("/", (req, res) => {
  res.json({ status: "ok" });
});

app.get("/app", (req, res) => {
  res.send("<h1>Web App</h1><p>Hello from the Web service</p>");
});

app.get("/app/health", (req, res) => {
  res.json({ status: "ok" });
});

app.get("/app/greeting", (req, res) => {
  res.send("<h1>Web App</h1><p>Hello from the Web service</p>");
});

app.get("/app/*", (req, res) => {
  res.send(`<h1>Web App</h1><p>Path: ${req.path}</p>`);
});

app.listen(PORT, () => {
  console.log(`Web service is running on http://localhost:${PORT}`);
});
