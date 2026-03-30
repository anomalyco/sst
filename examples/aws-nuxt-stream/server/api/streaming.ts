export default defineEventHandler(async (event) => {  
  const eventStream = createEventStream(event);

  eventStream.push("Start\n\n");
  
  const interval = setInterval(async () => {
    await eventStream.push(`Random: ${Math.random().toFixed(5)} `);
  }, 1000);

  setTimeout(async () => {
    clearInterval(interval);
    await eventStream.close();
  }, 10000);

  eventStream.onClosed(() => {
    clearInterval(interval);
  });

  return eventStream.send();
})
