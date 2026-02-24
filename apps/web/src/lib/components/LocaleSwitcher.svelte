<script lang="ts">
  import { locale, setLocale, type Locale } from '$lib/i18n';

  const locales: { code: Locale; label: string }[] = [
    { code: 'ru', label: 'Рус' },
    { code: 'uz', label: "O'zb" },
    { code: 'en', label: 'Eng' },
    { code: 'kk', label: 'Қаз' },
  ];
</script>

<div class="flex gap-1 text-xs">
  {#each locales as l}
    <button
      class="px-2 py-1 rounded transition-colors {$locale === l.code
        ? 'bg-indigo-600 text-white'
        : 'bg-gray-100 text-gray-600 hover:bg-gray-200'}"
      onclick={() => setLocale(l.code)}
    >
      {l.label}
    </button>
  {/each}
</div>
