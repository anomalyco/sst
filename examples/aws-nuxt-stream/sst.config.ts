/// <reference path="./.sst/platform/config.d.ts" />

/**
 * ## AWS Nuxt streaming
 *
 * An example of how to use streaming with Nuxt.js. Uses `createEventStream` to stream data from a server API.
 *
 * ```ts title="server/api/streaming.ts"
 * export default defineEventHandler(async (event) => {
 *   const eventStream = createEventStream(event);
 *   eventStream.push("Start\n\n");
 *
 *   // Send a message every second
 *   const interval = setInterval(async () => {
 *     await eventStream.push(`Random: ${Math.random().toFixed(5)} `);
 *   }, 1000);
 * }
 * ```
 *
 * The client uses the Fetch API to consume the stream.
 *
 * ```vue title="app.vue"
 * <script setup lang="ts">
 * const output = ref('')
 * 
 * async function stream() {
 *   output.value = ''
 *   const response = await fetch('/api/streaming')
 *   const reader = response.body?.getReader()
 *   const decoder = new TextDecoder()
 *   let done = false
 * 
 *   while (!done && reader) {
 *     const { value, done: readerDone } = await reader.read()
 *     done = readerDone
 *     if (value) {
 *       output.value += decoder.decode(value, { stream: true })
 *     }
 *   }
 * }
 * </script>
 * 
 * <template>
 *   <div>
 *     <pre>{{ output }}</pre>
 *     <button @click="stream">Call API</button>
 *     <button @click="output = ''">Clear Output</button>
 *   </div>
 * </template>
 * ```
 * 
 * Make sure to have your nuxt.config.ts set up to handle the streaming API correctly.
 * 
 * ```ts title="nuxt.config.ts" {4-6}
 * export default defineNuxtConfig({
 *  nitro: {
 *   preset: 'aws-lambda',
 *   awsLambda: {
 *     streaming: true
 *   }
 *  }
 * });
 * ```
 * 
 * You should see random numbers streamed to the page every second for 10 seconds.
 * 
 * :::note
 * Safari handles streaming differently than other browsers.
 * :::
 *
 * Safari uses a [different heuristic](https://bugs.webkit.org/show_bug.cgi?id=252413) to
 * determine when to stream data. You need to render _enough_ initial HTML to trigger streaming.
 * This is typically only a problem for demo apps.
 */
export default $config({
  app(input) {
    return {
      name: "aws-nuxt-stream",
      removal: input?.stage === "production" ? "retain" : "remove",
      home: "aws",
    };
  },
  async run() {
    new sst.aws.Nuxt("MyWeb");
  },
});
