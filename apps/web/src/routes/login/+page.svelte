<script lang="ts">
  let email = $state('');
  let password = $state('');
  let error = $state('');
  let loading = $state(false);

  async function handleSubmit(e: SubmitEvent) {
    e.preventDefault();
    error = '';
    loading = true;

    try {
      const res = await fetch('/api/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email: email.trim(), password: password.trim() }),
      });

      if (res.ok) {
        window.location.href = '/';
      } else {
        const data = await res.json().catch(() => ({}));
        error = data.error ?? 'Login failed. Please try again.';
      }
    } catch {
      error = 'Network error. Please try again.';
    } finally {
      loading = false;
    }
  }
</script>

<svelte:head>
  <title>Sign in — Zist</title>
</svelte:head>

<div class="flex min-h-[calc(100vh-140px)] items-center justify-center px-6 py-12">
  <div class="w-full max-w-sm">
    <div class="mb-8 text-center">
      <a href="/" class="text-3xl font-bold tracking-tight text-[#ff5a5f]">zist</a>
      <p class="mt-2 text-sm text-gray-500">Sign in to your account</p>
    </div>

    <form onsubmit={handleSubmit} class="space-y-4">
      <div>
        <label for="email" class="block text-sm font-medium text-gray-700">Email</label>
        <input
          id="email"
          type="email"
          autocomplete="email"
          required
          bind:value={email}
          class="mt-1 block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-[#ff5a5f] focus:outline-none focus:ring-1 focus:ring-[#ff5a5f]"
          placeholder="you@example.com"
        />
      </div>

      <div>
        <label for="password" class="block text-sm font-medium text-gray-700">Password</label>
        <input
          id="password"
          type="password"
          autocomplete="current-password"
          required
          bind:value={password}
          class="mt-1 block w-full rounded-lg border border-gray-300 px-3 py-2 text-sm shadow-sm focus:border-[#ff5a5f] focus:outline-none focus:ring-1 focus:ring-[#ff5a5f]"
          placeholder="••••••••"
        />
      </div>

      {#if error}
        <p class="text-sm text-red-600">{error}</p>
      {/if}

      <button
        type="submit"
        disabled={loading}
        class="w-full rounded-lg bg-[#ff5a5f] px-4 py-2.5 text-sm font-semibold text-white shadow-sm hover:bg-[#e04e53] disabled:opacity-60"
      >
        {loading ? 'Signing in…' : 'Sign in'}
      </button>
    </form>
  </div>
</div>
