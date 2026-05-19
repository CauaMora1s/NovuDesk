import { browser } from '$app/environment';
import { waitLocale } from 'svelte-i18n';
import { setupI18n } from '$lib/i18n';

export const prerender = false;
export const ssr = false;

export async function load() {
	setupI18n();
	// Wait for the locale JSON to finish loading before any component renders.
	// Without this, $_ throws "Cannot format a message without first setting
	// the initial locale" on hard refresh / direct navigation.
	if (browser) {
		await waitLocale();
	}
}
