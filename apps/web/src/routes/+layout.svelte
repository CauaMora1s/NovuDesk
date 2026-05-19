<script lang="ts">
	import '../app.css';
	import { setupI18n } from '$lib/i18n';
	import { onMount } from 'svelte';

	setupI18n();

	onMount(() => {
		// Apply saved theme preference
		const saved = localStorage.getItem('nd_theme') ?? 'light';
		document.documentElement.setAttribute('data-theme', saved);
	});
</script>

<slot />
