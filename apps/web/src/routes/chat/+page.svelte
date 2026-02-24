<script lang="ts">
  import { onMount, onDestroy } from 'svelte';

  interface Message {
    id: string;
    sender: string;
    text: string;
    ts: number;
    self: boolean;
  }

  let topic = $state('');
  let inputTopic = $state('');
  let messages = $state<Message[]>([]);
  let draft = $state('');
  let connected = $state(false);
  let ws: WebSocket | null = null;
  let error = $state('');
  let messagesEl: HTMLElement;

  function connect() {
    if (!inputTopic.trim()) return;
    topic = inputTopic.trim();
    error = '';

    const proto = window.location.protocol === 'https:' ? 'wss' : 'ws';
    const url = `${proto}://${window.location.host}/api/chat?topic=${encodeURIComponent(topic)}`;

    ws = new WebSocket(url);

    ws.onopen = () => {
      connected = true;
    };

    ws.onmessage = (ev) => {
      try {
        const msg = JSON.parse(ev.data);
        messages = [...messages, {
          id: msg.id ?? crypto.randomUUID(),
          sender: msg.sender ?? 'unknown',
          text: msg.text ?? ev.data,
          ts: msg.ts ?? Date.now(),
          self: false,
        }];
        scrollToBottom();
      } catch {
        // raw text fallback
        messages = [...messages, {
          id: crypto.randomUUID(),
          sender: 'system',
          text: ev.data,
          ts: Date.now(),
          self: false,
        }];
      }
    };

    ws.onerror = () => {
      error = 'Connection error. Is mgChat (HookLine) configured?';
      connected = false;
    };

    ws.onclose = () => {
      connected = false;
    };
  }

  function disconnect() {
    ws?.close();
    ws = null;
    connected = false;
    topic = '';
    messages = [];
  }

  function sendMessage() {
    if (!draft.trim() || !ws || ws.readyState !== WebSocket.OPEN) return;
    const msg = { text: draft.trim(), ts: Date.now() };
    ws.send(JSON.stringify(msg));
    messages = [...messages, {
      id: crypto.randomUUID(),
      sender: 'you',
      text: draft.trim(),
      ts: Date.now(),
      self: true,
    }];
    draft = '';
    scrollToBottom();
  }

  function scrollToBottom() {
    setTimeout(() => messagesEl?.scrollTo({ top: messagesEl.scrollHeight, behavior: 'smooth' }), 50);
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage();
    }
  }

  onDestroy(() => ws?.close());
</script>

<svelte:head>
  <title>Chat · Zist</title>
</svelte:head>

<div class="max-w-2xl mx-auto px-4 py-8 flex flex-col h-[80vh]">
  <h1 class="text-2xl font-bold mb-4">Chat</h1>

  {#if !topic}
    <!-- Connect form -->
    <div class="bg-white border rounded-xl p-6 flex flex-col gap-3">
      <p class="text-sm text-gray-500">Enter the booking ID or chat topic to join a conversation.</p>
      <input
        bind:value={inputTopic}
        placeholder="e.g. booking:abc-123"
        class="border rounded-lg px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500"
      />
      {#if error}
        <p class="text-red-500 text-sm">{error}</p>
      {/if}
      <button
        onclick={connect}
        disabled={!inputTopic.trim()}
        class="bg-indigo-600 text-white py-2 rounded-lg text-sm font-medium hover:bg-indigo-700 disabled:opacity-50"
      >
        Join chat
      </button>
    </div>
  {:else}
    <!-- Chat UI -->
    <div class="flex items-center justify-between mb-3">
      <div class="flex items-center gap-2">
        <span class="text-sm font-medium text-gray-700">Topic: {topic}</span>
        <span class="w-2 h-2 rounded-full {connected ? 'bg-green-500' : 'bg-gray-300'}"></span>
        <span class="text-xs text-gray-400">{connected ? 'Connected' : 'Disconnected'}</span>
      </div>
      <button onclick={disconnect} class="text-xs text-red-500 hover:underline">Leave</button>
    </div>

    <!-- Messages -->
    <div
      bind:this={messagesEl}
      class="flex-1 overflow-y-auto bg-gray-50 rounded-xl p-4 space-y-2 border"
    >
      {#each messages as msg (msg.id)}
        <div class="flex {msg.self ? 'justify-end' : 'justify-start'}">
          <div class="max-w-xs {msg.self ? 'bg-indigo-600 text-white' : 'bg-white border text-gray-800'} rounded-2xl px-4 py-2 text-sm shadow-sm">
            {#if !msg.self}
              <p class="text-xs text-gray-400 mb-0.5">{msg.sender}</p>
            {/if}
            <p>{msg.text}</p>
          </div>
        </div>
      {/each}
      {#if messages.length === 0}
        <p class="text-gray-400 text-sm text-center mt-8">No messages yet. Say hello!</p>
      {/if}
    </div>

    <!-- Input -->
    <div class="flex gap-2 mt-3">
      <textarea
        bind:value={draft}
        onkeydown={handleKeydown}
        rows={1}
        placeholder="Type a message…"
        class="flex-1 border rounded-xl px-3 py-2 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-indigo-500"
      ></textarea>
      <button
        onclick={sendMessage}
        disabled={!draft.trim() || !connected}
        class="bg-indigo-600 text-white px-4 rounded-xl text-sm font-medium hover:bg-indigo-700 disabled:opacity-50"
      >
        Send
      </button>
    </div>
  {/if}
</div>
