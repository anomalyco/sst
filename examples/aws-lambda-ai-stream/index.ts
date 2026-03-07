import { streamText } from 'ai';

export const handler = awslambda.streamifyResponse(async (_event, responseStream) => {
  const result = streamText({
    model: 'amazon/nova-micro',
    prompt: 'Write a poem about clouds that is twenty paragraphs long.',
  });

  responseStream.setContentType('text/plain');
  for await (const chunk of result.textStream) {
    responseStream.write(chunk);
    process.stdout.write(chunk);
  }
  responseStream.end();
});
