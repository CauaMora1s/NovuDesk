<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { can } from '$lib/permissions';
</script>

<svelte:head><title>Times — NovuDesk</title></svelte:head>

<div class="p-8 max-w-4xl mx-auto">
	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold">{$_('teams.title')}</h1>
		{#if can('teams:manage')}
			<button class="btn btn-primary btn-sm gap-2">
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
				</svg>
				{$_('teams.new')}
			</button>
		{/if}
	</div>

	<div class="card bg-base-100 shadow-card">
		<div class="card-body flex flex-col items-center justify-center py-16 text-base-content/40">
			<svg class="w-10 h-10 mb-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
					d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
			</svg>
			<p class="text-sm font-medium">{$_('teams.noTeams')}</p>
		</div>
	</div>
</div>
