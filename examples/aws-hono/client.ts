const response = await fetch("http://localhost:13557/rpc/tunnel");
const reader = response.body!.getReader();
const decoder = new TextDecoder();
while (true) {
  console.log("Waiting for data...");
  const { done, value } = await reader.read();

  if (done) break;

  const chunk = decoder.decode(value);
  buffer += chunk;

  const lines = buffer.split("\r\n");

  // Process all complete lines
  while (lines.length > 1) {
    const line = lines.shift();
    console.log("Processed line:", line);
  }

  // Keep any incomplete line in the buffer
  buffer = lines[0];
}

// Process any remaining line in the buffer
if (buffer) {
  console.log("Final line:", buffer);
}
