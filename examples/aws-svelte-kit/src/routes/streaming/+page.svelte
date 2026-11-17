<script>
  /** @type {import('./$types').PageData} */
  export let data;
</script>

<style>
  section {
    max-width: 600px;
    margin: 2rem auto;
    font-family: system-ui, sans-serif;
  }
  .card {
    border: 1px solid #ddd;
    border-radius: 8px;
    padding: 1rem;
    margin-bottom: 1rem;
  }
  .card h3 {
    margin: 0 0 0.5rem;
  }
  .loading {
    color: #888;
    animation: pulse 1.5s infinite;
  }
  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50% { opacity: 0.4; }
  }
  .done {
    color: #16a34a;
  }
  code {
    background: #f3f4f6;
    padding: 0.15rem 0.4rem;
    border-radius: 4px;
    font-size: 0.85em;
  }
</style>

<section>
  <h1>SvelteKit Streaming Demo</h1>
  <p>This page demonstrates Lambda response streaming. The shell and instant data arrive first, then streamed promises resolve progressively.</p>

  <div class="card">
    <h3>Instant Data</h3>
    <p class="done">{data.instant.message}</p>
    <p><code>timestamp: {data.instant.timestamp}</code></p>
  </div>

  <div class="card">
    <h3>Streamed — 2s delay</h3>
    {#await data.streamed.slow}
      <p class="loading">Waiting for data...</p>
    {:then result}
      <p class="done">{result.message}</p>
      <p><code>timestamp: {result.timestamp}</code></p>
    {/await}
  </div>

  <div class="card">
    <h3>Streamed — 4s delay</h3>
    {#await data.streamed.slower}
      <p class="loading">Waiting for data...</p>
    {:then result}
      <p class="done">{result.message}</p>
      <p><code>timestamp: {result.timestamp}</code></p>
    {/await}
  </div>

  <p><a href="/">← Back to home</a></p>
</section>
