<script lang="ts">
	import Sidebar from '$lib/components/layout/Sidebar.svelte';
</script>

<div class="flex h-screen overflow-hidden bg-base-200">
	<Sidebar />

	<main class="flex-1 overflow-y-auto">
		<slot />
	</main>
</div>
