<script lang="ts">
  import type { PageData } from './$types';

  interface Flag {
    id: string;
    name: string;
    enabled: boolean;
    rollout: number;
    tenantId?: string;
    updatedAt: number;
  }

  let { data }: { data: PageData } = $props();

  let flags = $state<Flag[]>(data.flags);
  let newName = $state('');
  let newEnabled = $state(false);
  let newRollout = $state(100);
  let saving = $state(false);
  let error = $state('');

  async function saveFlag(flag?: Flag) {
    saving = true;
    error = '';
    const body = flag
      ? { name: flag.name, enabled: flag.enabled, rollout: flag.rollout }
      : { name: newName.trim(), enabled: newEnabled, rollout: newRollout };

    if (!body.name) {
      error = 'Name is required';
      saving = false;
      return;
    }

    const res = await fetch('/api/admin/flags', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(body),
    });

    if (!res.ok) {
      const j = await res.json();
      error = j.error ?? 'Failed to save flag';
    } else {
      const saved: Flag = await res.json();
      const idx = flags.findIndex((f) => f.name === saved.name);
      if (idx >= 0) {
        flags[idx] = saved;
      } else {
        flags = [...flags, saved];
        newName = '';
        newEnabled = false;
        newRollout = 100;
      }
    }
    saving = false;
  }
</script>

<svelte:head>
  <title>Feature Flags · Admin · Zist</title>
</svelte:head>

<div class="max-w-3xl mx-auto px-4 py-8">
  <div class="flex items-center gap-3 mb-6">
    <a href="/admin/webhooks" class="text-indigo-600 hover:underline text-sm">← Webhooks</a>
    <h1 class="text-2xl font-bold">Feature Flags</h1>
  </div>

  {#if error}
    <div class="bg-red-50 border border-red-200 text-red-700 text-sm rounded-lg p-3 mb-4">{error}</div>
  {/if}

  <!-- Create new flag -->
  <div class="bg-white border rounded-xl p-5 mb-6">
    <h2 class="font-semibold text-sm mb-3 text-gray-700">New flag</h2>
    <div class="flex gap-3 items-end flex-wrap">
      <div class="flex-1 min-w-40">
        <label class="text-xs text-gray-500 mb-1 block">Name</label>
        <input bind:value={newName} placeholder="e.g. instant_book_v2"
          class="w-full border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500" />
      </div>
      <div class="w-24">
        <label class="text-xs text-gray-500 mb-1 block">Rollout %</label>
        <input type="number" min="0" max="100" bind:value={newRollout}
          class="w-full border rounded-lg px-3 py-2 text-sm" />
      </div>
      <label class="flex items-center gap-2 text-sm cursor-pointer">
        <input type="checkbox" bind:checked={newEnabled} class="w-4 h-4" />
        Enabled
      </label>
      <button onclick={() => saveFlag()}
        disabled={saving}
        class="bg-indigo-600 text-white px-4 py-2 rounded-lg text-sm font-medium hover:bg-indigo-700 disabled:opacity-50">
        Add
      </button>
    </div>
  </div>

  <!-- Flags table -->
  <div class="bg-white border rounded-xl overflow-hidden">
    {#if flags.length === 0}
      <p class="text-gray-400 text-sm p-6 text-center">No flags configured.</p>
    {:else}
      <table class="w-full text-sm">
        <thead class="bg-gray-50 border-b">
          <tr>
            <th class="text-left px-5 py-3 font-medium text-gray-600">Name</th>
            <th class="text-left px-4 py-3 font-medium text-gray-600">Rollout</th>
            <th class="text-left px-4 py-3 font-medium text-gray-600">Enabled</th>
            <th class="px-4 py-3"></th>
          </tr>
        </thead>
        <tbody>
          {#each flags as flag (flag.id)}
            <tr class="border-b last:border-0 hover:bg-gray-50">
              <td class="px-5 py-3 font-mono text-xs text-gray-800">{flag.name}</td>
              <td class="px-4 py-3">
                <input type="number" min="0" max="100" bind:value={flag.rollout}
                  class="w-20 border rounded px-2 py-1 text-xs" />
              </td>
              <td class="px-4 py-3">
                <input type="checkbox" bind:checked={flag.enabled} class="w-4 h-4 cursor-pointer" />
              </td>
              <td class="px-4 py-3 text-right">
                <button onclick={() => saveFlag(flag)}
                  disabled={saving}
                  class="text-xs text-indigo-600 hover:underline disabled:opacity-50">
                  Save
                </button>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    {/if}
  </div>
</div>
