<script lang="ts">
  import '../app.css';
  import type { LayoutData } from './$types';

  let { children, data }: { children: any; data: LayoutData } = $props();

  const { user } = data;

  async function signOut() {
    await fetch('/api/auth/logout', { method: 'POST' });
    window.location.href = '/';
  }
</script>

<div class="min-h-screen bg-white">
  <!-- Navbar -->
  <header class="sticky top-0 z-50 border-b border-gray-200 bg-white">
    <nav class="mx-auto flex max-w-7xl items-center justify-between px-6 py-4">
      <a href="/" class="text-2xl font-bold tracking-tight text-[#ff5a5f]">
        zist
      </a>

      <div class="hidden items-center gap-6 md:flex">
        <a href="/listings" class="text-sm font-medium text-gray-700 hover:text-gray-900">
          Explore
        </a>
        <a href="/bookings" class="text-sm font-medium text-gray-700 hover:text-gray-900">
          My trips
        </a>
      </div>

      <div class="flex items-center gap-3">
        {#if user}
          <span class="text-sm text-gray-600">{user.email}</span>
          <button
            onclick={signOut}
            class="rounded-full border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            Sign out
          </button>
        {:else}
          <a
            href="/api/auth/login"
            class="rounded-full border border-gray-300 px-4 py-2 text-sm font-medium text-gray-700 hover:bg-gray-50"
          >
            Sign in
          </a>
        {/if}
      </div>
    </nav>
  </header>

  <!-- Page content -->
  <main>
    {@render children()}
  </main>

  <!-- Footer -->
  <footer class="border-t border-gray-200 bg-gray-50 py-10">
    <div class="mx-auto max-w-7xl px-6 text-center text-sm text-gray-500">
      © {new Date().getFullYear()} Zist — stays across Central Asia
    </div>
  </footer>
</div>
