<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { can } from '$lib/permissions';

	let activeTab = 'organization';

	const tabs = [
		{ key: 'organization', label: 'settings.organization', permission: 'organization:view_settings' },
		{ key: 'members',      label: 'settings.members',      permission: 'users:view' },
		{ key: 'roles',        label: 'settings.roles',        permission: 'organization:view_settings' },
		{ key: 'sla',          label: 'settings.sla',          permission: 'sla:view' }
	] as const;

	$: visibleTabs = tabs.filter((t) => can(t.permission));
</script>

<svelte:head><title>Configurações — NovuDesk</title></svelte:head>

<div class="p-8 max-w-5xl mx-auto">
	<h1 class="text-2xl font-bold mb-6">{$_('settings.title')}</h1>

	<div class="flex gap-6">
		<!-- Vertical tabs -->
		<nav class="w-44 shrink-0 space-y-0.5">
			{#each visibleTabs as tab}
				<button
					class="sidebar-link w-full text-left"
					class:active={activeTab === tab.key}
					on:click={() => (activeTab = tab.key)}
				>
					{$_(tab.label)}
				</button>
			{/each}
		</nav>

		<!-- Content -->
		<div class="flex-1 card bg-base-100 shadow-card">
			<div class="card-body p-6">
				{#if activeTab === 'organization'}
					<h2 class="font-semibold mb-4">{$_('settings.organization')}</h2>
					<p class="text-sm text-base-content/50">Configurações da organização em construção.</p>
				{:else if activeTab === 'members'}
					<h2 class="font-semibold mb-4">{$_('settings.members')}</h2>
					<p class="text-sm text-base-content/50">Gestão de membros em construção.</p>
				{:else if activeTab === 'roles'}
					<h2 class="font-semibold mb-4">{$_('settings.roles')}</h2>
					<p class="text-sm text-base-content/50">Gestão de funções em construção.</p>
				{:else if activeTab === 'sla'}
					<h2 class="font-semibold mb-4">{$_('settings.sla')}</h2>
					<p class="text-sm text-base-content/50">Configurações de SLA em construção.</p>
				{/if}
			</div>
		</div>
	</div>
</div>
