<script lang="ts">
  import { getArbitraryData } from "./home.remote";
</script>

<svelte:boundary>
  {#snippet pending()}
    <p>Loading...</p>
  {/snippet}

  {#snippet failed(err, reset)}
    <p>Error: {err}</p>
    <button
      onclick={() => {
        reset();
      }}>Reset</button
    >
  {/snippet}

  <ul>
    {#each await getArbitraryData() as item}
      <li>{item}</li>
    {/each}
  </ul>
</svelte:boundary>
