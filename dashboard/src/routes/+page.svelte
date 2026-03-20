<script lang="ts">
  import { onMount } from 'svelte';
  
  let prices = { BTC: 0, ETH: 0, SOL: 0 };
  let loading = true;

  async function fetchPrices() {
    try {
      const symbols = ['BTC', 'ETH', 'SOL'];
      for (const sym of symbols) {
        // Fetching from localhost dispatcher in this demo
        const res = await fetch(`http://localhost:3001/prices/${sym}`);
        if (res.ok) {
          const data = await res.json();
          prices[sym] = data.price ? (data.price / 100000000).toFixed(2) : 0;
        }
      }
    } catch (e) {
      console.error("Error fetching prices", e);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    fetchPrices();
    const interval = setInterval(fetchPrices, 3000);
    return () => clearInterval(interval);
  });
</script>

<div class="min-h-screen bg-slate-950 flex flex-col items-center justify-center p-8 font-sans selection:bg-emerald-500/30">
  
  <div class="absolute inset-0 overflow-hidden pointer-events-none">
    <div class="absolute -top-40 -right-40 w-96 h-96 bg-emerald-500 rounded-full mix-blend-multiply filter blur-3xl opacity-20 animate-blob"></div>
    <div class="absolute -bottom-40 -left-40 w-96 h-96 bg-blue-500 rounded-full mix-blend-multiply filter blur-3xl opacity-20 animate-blob animation-delay-2000"></div>
  </div>

  <div class="z-10 w-full max-w-4xl backdrop-blur-xl bg-white/5 border border-white/10 rounded-3xl p-10 shadow-2xl">
    <div class="text-center mb-12">
      <h1 class="text-5xl font-extrabold text-transparent bg-clip-text bg-gradient-to-r from-emerald-400 to-cyan-400 tracking-tight">
        Web3 Oracle Feed
      </h1>
      <p class="text-slate-400 mt-4 text-lg">Real-time aggregated prices ingested from Bitfinex</p>
    </div>

    {#if loading}
      <div class="flex justify-center items-center h-48">
        <div class="animate-spin rounded-full h-12 w-12 border-t-2 border-b-2 border-emerald-400"></div>
      </div>
    {:else}
      <div class="grid grid-cols-1 md:grid-cols-3 gap-6">
        {#each Object.entries(prices) as [symbol, price]}
          <div class="group relative overflow-hidden rounded-2xl bg-white/5 border border-white/10 p-6 transition-all duration-300 hover:bg-white/10 hover:scale-105 hover:shadow-[0_0_30px_rgba(52,211,153,0.15)]">
            <div class="flex justify-between items-center mb-4">
              <span class="text-emerald-400 font-medium tracking-wider">{symbol}/USD</span>
              <div class="h-2 w-2 rounded-full bg-emerald-400 animate-pulse"></div>
            </div>
            <div class="text-4xl font-bold text-white tracking-tight">
              ${price === 0 ? '---' : price}
            </div>
            
            <div class="absolute bottom-0 left-0 h-1 w-full bg-gradient-to-r from-emerald-500 to-cyan-500 transform origin-left scale-x-0 transition-transform duration-300 group-hover:scale-x-100"></div>
          </div>
        {/each}
      </div>
    {/if}

    <div class="mt-12 pt-6 border-t border-white/10 flex justify-between items-center text-sm text-slate-500">
      <div class="flex items-center gap-2">
        <div class="h-2 w-2 rounded-full bg-emerald-500"></div>
        System Operational
      </div>
      <div>Update interval: 3s</div>
    </div>
  </div>
</div>

<style>
  :global(body) {
    margin: 0;
    color: white;
    background-color: #020617; /* tailwind slate-950 */
  }

  @keyframes blob {
    0% { transform: translate(0px, 0px) scale(1); }
    33% { transform: translate(30px, -50px) scale(1.1); }
    66% { transform: translate(-20px, 20px) scale(0.9); }
    100% { transform: translate(0px, 0px) scale(1); }
  }

  .animate-blob {
    animation: blob 7s infinite;
  }
  
  .animation-delay-2000 {
    animation-delay: 2s;
  }
</style>
