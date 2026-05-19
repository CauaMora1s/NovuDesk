<script lang="ts">
	import { isAuthenticated } from '$lib/stores/auth';
	import { goto } from '$app/navigation';
	import { browser } from '$app/environment';
	import Sidebar from '$lib/components/layout/Sidebar.svelte';

	$: if (browser && !$isAuthenticated) {
		goto('/login');
	}
</script>

{#if $isAuthenticated}
	<div class="flex h-screen overflow-hidden bg-base-200">
		<Sidebar />

		<main class="flex-1 overflow-y-auto">
			<slot />
		</main>
	</div>
{/if}
