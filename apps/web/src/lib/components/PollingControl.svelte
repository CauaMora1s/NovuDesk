<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { pollingInterval, type PollingInterval } from '$lib/stores/polling';

	const options: Array<{ value: PollingInterval; labelKey: string }> = [
		{ value: 10_000, labelKey: 'polling.10s' },
		{ value: 30_000, labelKey: 'polling.30s' },
		{ value: 60_000, labelKey: 'polling.1min' },
		{ value: 0,      labelKey: 'polling.off' }
	];
</script>

<div class="flex items-center gap-2 text-xs text-base-content/50">
	<span>{$_('polling.label')}</span>
	<div class="join">
		{#each options as opt}
			<button
				class="join-item btn btn-xs"
				class:btn-primary={$pollingInterval === opt.value}
				class:bg-base-100={$pollingInterval !== opt.value}
				class:border-base-200={$pollingInterval !== opt.value}
				on:click={() => pollingInterval.set(opt.value)}
			>
				{$_(opt.labelKey)}
			</button>
		{/each}
	</div>
</div>
