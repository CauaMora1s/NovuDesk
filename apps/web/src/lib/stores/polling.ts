import { writable } from 'svelte/store';

export type PollingInterval = 10_000 | 30_000 | 60_000 | 0;

export const pollingInterval = writable<PollingInterval>(30_000);
