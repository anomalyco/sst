export default {
  async queue(batch, env) {
    for (const message of batch.messages) {
      console.log("Processing message:", message.body);
    }
  },
};
