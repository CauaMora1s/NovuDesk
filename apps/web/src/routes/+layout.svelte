<script lang="ts">
	import '../app.css';
	import { onMount } from 'svelte';

	onMount(() => {
		// Apply saved theme preference
		const saved = localStorage.getItem('nd_theme') ?? 'light';
		document.documentElement.setAttribute('data-theme', saved);
	});
</script>

<slot />
