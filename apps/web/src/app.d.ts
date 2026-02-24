// See https://svelte.dev/docs/kit/types#app.d.ts

declare global {
  namespace App {
    interface Locals {
      user?: {
        user_id: string;
        email: string;
        tenant_id: string;
        scopes: string;
      };
    }
  }
}

export {};
