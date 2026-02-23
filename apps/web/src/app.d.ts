// See https://svelte.dev/docs/kit/types#app.d.ts

declare global {
  namespace App {
    interface Locals {
      user?: {
        sub: string;
        email: string;
        name?: string;
        tenant_id: string;
        roles: string[];
        scope: string;
      };
    }
  }
}

export {};
