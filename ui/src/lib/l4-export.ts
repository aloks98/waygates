import type { CreateL4ProxyRequest } from '@/types/l4-proxy';

// Shape of an exported L4 proxy — identical to the create request. The backend
// GET /l4-proxies/export emits this (dropping ids/timestamps), so an exported
// file round-trips through the create/import path.
export type L4Export = CreateL4ProxyRequest;
