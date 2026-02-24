/**
 * Playwright global setup — starts the mock gateway before the webServer.
 *
 * Returns a teardown function that Playwright calls after all tests complete.
 */
import type { FullConfig } from '@playwright/test';
import { startMockGateway } from './mock-gateway.js';

export default async function globalSetup(_config: FullConfig) {
  const gateway = await startMockGateway(9999);
  console.log(`[mock-gateway] listening on http://localhost:${gateway.port}`);

  return async () => {
    await gateway.close();
    console.log('[mock-gateway] stopped');
  };
}
