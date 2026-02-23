<script lang="ts">
  import { onMount } from 'svelte';

  // ─── Types ────────────────────────────────────────────────────────────────

  interface Endpoint {
    id: string;
    url: string;
    description?: string;
    eventTypes: string[];
    status: string;
    signingSecret?: string;
    createdAt: number;
    updatedAt: number;
  }

  interface Delivery {
    id: string;
    endpointId: string;
    eventId: string;
    status: string;
    attemptCount: number;
    responseStatus: number;
    nextRetryAt?: number;
    createdAt: number;
  }

  // ─── State ────────────────────────────────────────────────────────────────

  let endpoints: Endpoint[] = [];
  let loadingEndpoints = true;
  let endpointsError = '';

  // Create form
  let showCreate = false;
  let newUrl = '';
  let newDescription = '';
  let newEventTypes = 'payment.captured,payment.failed,checkout.completed';
  let creating = false;
  let createError = '';

  // Delivery panel
  let selectedEndpoint: Endpoint | null = null;
  let deliveries: Delivery[] = [];
  let loadingDeliveries = false;
  let deliveriesError = '';

  // Rotate secret
  let rotatedSecret = '';

  // ─── API helpers ──────────────────────────────────────────────────────────

  async function apiFetch(path: string, options: RequestInit = {}) {
    const res = await fetch('/api/admin/webhooks' + path, {
      ...options,
      headers: {
        'Content-Type': 'application/json',
        ...(options.headers ?? {}),
      },
    });
    if (!res.ok) {
      const body = await res.text();
      throw new Error(body || `HTTP ${res.status}`);
    }
    const text = await res.text();
    return text ? JSON.parse(text) : null;
  }

  // ─── Lifecycle ────────────────────────────────────────────────────────────

  onMount(loadEndpoints);

  async function loadEndpoints() {
    loadingEndpoints = true;
    endpointsError = '';
    try {
      const data = await apiFetch('/endpoints');
      endpoints = data?.endpoints ?? [];
    } catch (err: any) {
      endpointsError = err.message ?? 'Failed to load endpoints';
    } finally {
      loadingEndpoints = false;
    }
  }

  // ─── Create endpoint ──────────────────────────────────────────────────────

  async function handleCreate() {
    createError = '';
    if (!newUrl.trim()) return;
    creating = true;
    try {
      const endpoint = await apiFetch('/endpoints', {
        method: 'POST',
        body: JSON.stringify({
          url: newUrl.trim(),
          description: newDescription.trim() || undefined,
          eventTypes: newEventTypes.split(',').map((s) => s.trim()).filter(Boolean),
        }),
      });
      endpoints = [endpoint, ...endpoints];
      showCreate = false;
      newUrl = '';
      newDescription = '';
      newEventTypes = 'payment.captured,payment.failed,checkout.completed';
    } catch (err: any) {
      createError = err.message ?? 'Failed to create endpoint';
    } finally {
      creating = false;
    }
  }

  // ─── Rotate secret ────────────────────────────────────────────────────────

  async function rotateSecret(endpoint: Endpoint) {
    rotatedSecret = '';
    try {
      const updated = await apiFetch(`/endpoints/${endpoint.id}/rotate-secret`, { method: 'POST' });
      rotatedSecret = updated?.signingSecret ?? '';
      endpoints = endpoints.map((e) => (e.id === endpoint.id ? { ...e, ...updated } : e));
    } catch (err: any) {
      alert('Failed to rotate secret: ' + (err.message ?? 'unknown error'));
    }
  }

  // ─── Delete endpoint ──────────────────────────────────────────────────────

  async function deleteEndpoint(id: string) {
    if (!confirm('Delete this endpoint?')) return;
    try {
      await apiFetch(`/endpoints/${id}`, { method: 'DELETE' });
      endpoints = endpoints.filter((e) => e.id !== id);
      if (selectedEndpoint?.id === id) selectedEndpoint = null;
    } catch (err: any) {
      alert('Failed to delete endpoint: ' + (err.message ?? 'unknown error'));
    }
  }

  // ─── Deliveries ───────────────────────────────────────────────────────────

  async function openDeliveries(endpoint: Endpoint) {
    selectedEndpoint = endpoint;
    deliveries = [];
    deliveriesError = '';
    loadingDeliveries = true;
    try {
      const data = await apiFetch(`/endpoints/${endpoint.id}/deliveries`);
      deliveries = data?.deliveries ?? [];
    } catch (err: any) {
      deliveriesError = err.message ?? 'Failed to load deliveries';
    } finally {
      loadingDeliveries = false;
    }
  }

  async function retryDelivery(delivery: Delivery) {
    try {
      await apiFetch(`/endpoints/${delivery.endpointId}/deliveries/${delivery.id}/retry`, {
        method: 'POST',
      });
      await openDeliveries(selectedEndpoint!);
    } catch (err: any) {
      alert('Retry failed: ' + (err.message ?? 'unknown error'));
    }
  }

  function formatDate(ts: number) {
    return new Date(ts * 1000).toLocaleString();
  }
</script>

<div class="space-y-6 p-6">
  <!-- Header -->
  <div class="flex items-center justify-between">
    <div>
      <h1 class="text-2xl font-semibold">Webhook Endpoints</h1>
      <p class="mt-1 text-sm text-gray-500">Manage event delivery destinations for this workspace.</p>
    </div>
    <button
      class="rounded bg-indigo-600 px-4 py-2 text-sm text-white hover:bg-indigo-700 disabled:opacity-50"
      on:click={() => (showCreate = !showCreate)}
    >
      + New Endpoint
    </button>
  </div>

  <!-- New endpoint form -->
  {#if showCreate}
    <div class="rounded-lg border bg-white p-4 shadow-sm">
      <h2 class="mb-3 font-semibold">Create Endpoint</h2>
      <div class="space-y-3">
        <div>
          <label class="mb-1 block text-sm font-medium text-gray-700">URL *</label>
          <input
            type="url"
            bind:value={newUrl}
            placeholder="https://your-app.com/webhooks/mashgate"
            class="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
          />
        </div>
        <div>
          <label class="mb-1 block text-sm font-medium text-gray-700">Description</label>
          <input
            type="text"
            bind:value={newDescription}
            placeholder="Optional description"
            class="w-full rounded border px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
          />
        </div>
        <div>
          <label class="mb-1 block text-sm font-medium text-gray-700">
            Event Types (comma-separated)
          </label>
          <input
            type="text"
            bind:value={newEventTypes}
            class="w-full rounded border px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-indigo-500"
          />
        </div>
        {#if createError}
          <p class="text-sm text-red-600">{createError}</p>
        {/if}
        <div class="flex gap-2">
          <button
            class="rounded bg-indigo-600 px-4 py-2 text-sm text-white hover:bg-indigo-700 disabled:opacity-50"
            disabled={creating || !newUrl.trim()}
            on:click={handleCreate}
          >
            {creating ? 'Creating…' : 'Create'}
          </button>
          <button
            class="rounded border px-4 py-2 text-sm hover:bg-gray-50"
            on:click={() => (showCreate = false)}
          >
            Cancel
          </button>
        </div>
      </div>
    </div>
  {/if}

  <!-- Rotated secret banner -->
  {#if rotatedSecret}
    <div class="rounded border border-yellow-300 bg-yellow-50 p-3 text-sm">
      <p class="font-semibold text-yellow-800">New signing secret (shown once):</p>
      <code class="mt-1 block break-all font-mono text-yellow-900">{rotatedSecret}</code>
      <button class="mt-2 text-xs text-yellow-700 underline" on:click={() => (rotatedSecret = '')}>
        Dismiss
      </button>
    </div>
  {/if}

  <!-- Endpoints table -->
  {#if loadingEndpoints}
    <div class="space-y-2">
      {#each [1, 2, 3] as _}
        <div class="h-12 animate-pulse rounded bg-gray-100"></div>
      {/each}
    </div>
  {:else if endpointsError}
    <p class="text-sm text-red-600">{endpointsError}</p>
  {:else if endpoints.length === 0}
    <p class="text-sm text-gray-400">No endpoints yet. Create one above.</p>
  {:else}
    <div class="overflow-x-auto rounded-lg border bg-white shadow-sm">
      <table class="min-w-full divide-y divide-gray-200 text-sm">
        <thead class="bg-gray-50">
          <tr>
            <th class="px-4 py-3 text-left font-medium text-gray-500">URL</th>
            <th class="px-4 py-3 text-left font-medium text-gray-500">Event Types</th>
            <th class="px-4 py-3 text-left font-medium text-gray-500">Status</th>
            <th class="px-4 py-3 text-left font-medium text-gray-500">Created</th>
            <th class="px-4 py-3 text-left font-medium text-gray-500">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          {#each endpoints as endpoint}
            <tr class="hover:bg-gray-50">
              <td class="px-4 py-3">
                <p class="max-w-xs truncate font-mono">{endpoint.url}</p>
                {#if endpoint.description}
                  <p class="text-xs text-gray-400">{endpoint.description}</p>
                {/if}
              </td>
              <td class="px-4 py-3">
                <div class="flex flex-wrap gap-1">
                  {#each endpoint.eventTypes as et}
                    <span class="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-xs">{et}</span>
                  {/each}
                </div>
              </td>
              <td class="px-4 py-3">
                <span
                  class="rounded px-2 py-0.5 text-xs font-medium {endpoint.status === 'active'
                    ? 'bg-green-100 text-green-700'
                    : 'bg-gray-100 text-gray-500'}"
                >
                  {endpoint.status}
                </span>
              </td>
              <td class="px-4 py-3 text-gray-500">{formatDate(endpoint.createdAt)}</td>
              <td class="px-4 py-3">
                <div class="flex gap-2">
                  <button
                    class="text-indigo-600 hover:underline"
                    on:click={() => openDeliveries(endpoint)}
                  >
                    Deliveries
                  </button>
                  <button
                    class="text-gray-500 hover:underline"
                    on:click={() => rotateSecret(endpoint)}
                  >
                    Rotate
                  </button>
                  <button
                    class="text-red-500 hover:underline"
                    on:click={() => deleteEndpoint(endpoint.id)}
                  >
                    Delete
                  </button>
                </div>
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}

  <!-- Delivery history panel -->
  {#if selectedEndpoint}
    <div class="rounded-lg border bg-white p-4 shadow-sm">
      <div class="mb-3 flex items-center justify-between">
        <h2 class="font-semibold">
          Deliveries — <span class="font-mono text-sm">{selectedEndpoint.url}</span>
        </h2>
        <button class="text-sm text-gray-500 hover:underline" on:click={() => (selectedEndpoint = null)}>
          Close
        </button>
      </div>

      {#if loadingDeliveries}
        <div class="space-y-2">
          {#each [1, 2] as _}
            <div class="h-10 animate-pulse rounded bg-gray-100"></div>
          {/each}
        </div>
      {:else if deliveriesError}
        <p class="text-sm text-red-600">{deliveriesError}</p>
      {:else if deliveries.length === 0}
        <p class="text-sm text-gray-400">No deliveries yet.</p>
      {:else}
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 text-sm">
            <thead class="bg-gray-50">
              <tr>
                <th class="px-3 py-2 text-left font-medium text-gray-500">Event ID</th>
                <th class="px-3 py-2 text-left font-medium text-gray-500">Status</th>
                <th class="px-3 py-2 text-left font-medium text-gray-500">HTTP</th>
                <th class="px-3 py-2 text-left font-medium text-gray-500">Attempts</th>
                <th class="px-3 py-2 text-left font-medium text-gray-500">Created</th>
                <th class="px-3 py-2 text-left font-medium text-gray-500">Actions</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100">
              {#each deliveries as delivery}
                <tr>
                  <td class="px-3 py-2 font-mono text-xs">{delivery.eventId}</td>
                  <td class="px-3 py-2">
                    <span
                      class="rounded px-1.5 py-0.5 text-xs font-medium {delivery.status ===
                      'succeeded'
                        ? 'bg-green-100 text-green-700'
                        : delivery.status === 'failed'
                          ? 'bg-red-100 text-red-600'
                          : 'bg-yellow-100 text-yellow-700'}"
                    >
                      {delivery.status}
                    </span>
                  </td>
                  <td class="px-3 py-2">{delivery.responseStatus || '-'}</td>
                  <td class="px-3 py-2">{delivery.attemptCount}</td>
                  <td class="px-3 py-2 text-gray-500">{formatDate(delivery.createdAt)}</td>
                  <td class="px-3 py-2">
                    {#if delivery.status === 'failed'}
                      <button
                        class="text-indigo-600 hover:underline"
                        on:click={() => retryDelivery(delivery)}
                      >
                        Retry
                      </button>
                    {/if}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </div>
  {/if}
</div>
