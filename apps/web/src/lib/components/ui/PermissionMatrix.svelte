<script lang="ts">
	import { _ } from 'svelte-i18n';
	import type { Permission } from '$lib/api/roles';

	export let permissions: Permission[] = [];
	export let selected: string[] = [];
	export let readonly = false;

	// For member overrides: null = default (use role), true = grant, false = deny
	export let overrideMode = false;
	export let overrides: Record<string, boolean | null> = {};

	const groups: { key: string; i18nKey: string }[] = [
		{ key: 'tickets', i18nKey: 'permissionGroups.tickets' },
		{ key: 'comments', i18nKey: 'permissionGroups.comments' },
		{ key: 'teams', i18nKey: 'permissionGroups.teams' },
		{ key: 'users', i18nKey: 'permissionGroups.users' },
		{ key: 'organization', i18nKey: 'permissionGroups.organization' },
		{ key: 'sla', i18nKey: 'permissionGroups.sla' },
		{ key: 'other', i18nKey: 'permissionGroups.other' }
	];

	function groupFor(key: string): string {
		const prefix = key.split(':')[0];
		return groups.find((g) => g.key === prefix) ? prefix : 'other';
	}

	$: grouped = groups.map((g) => ({
		...g,
		perms: permissions.filter((p) => groupFor(p.key) === g.key)
	})).filter((g) => g.perms.length > 0);

	function togglePermission(key: string) {
		if (readonly || overrideMode) return;
		if (selected.includes(key)) {
			selected = selected.filter((k) => k !== key);
		} else {
			selected = [...selected, key];
		}
	}

	function cycleOverride(key: string) {
		if (readonly) return;
		const current = overrides[key] ?? null;
		if (current === null) {
			overrides = { ...overrides, [key]: true };
		} else if (current === true) {
			overrides = { ...overrides, [key]: false };
		} else {
			const { [key]: _, ...rest } = overrides;
			overrides = rest;
		}
	}

	function overrideBadgeClass(val: boolean | null): string {
		if (val === true) return 'badge badge-xs badge-success';
		if (val === false) return 'badge badge-xs badge-error';
		return 'badge badge-xs badge-ghost';
	}

	function overrideBadgeLabel(val: boolean | null): string {
		if (val === true) return $_('memberEdit.overrideGrant');
		if (val === false) return $_('memberEdit.overrideDeny');
		return $_('memberEdit.overrideNone');
	}
</script>

<div class="space-y-4">
	{#each grouped as group}
		<div>
			<p class="text-xs font-semibold text-base-content/50 uppercase tracking-wider mb-2">
				{$_(group.i18nKey)}
			</p>
			<div class="space-y-1">
				{#each group.perms as perm}
					{#if overrideMode}
						<div
							class="flex items-center justify-between gap-3 px-3 py-2 rounded-lg hover:bg-base-200/50 transition-colors"
							class:cursor-pointer={!readonly}
							on:click={() => cycleOverride(perm.key)}
							on:keydown={(e) => e.key === 'Enter' && cycleOverride(perm.key)}
							role="button"
							tabindex={readonly ? -1 : 0}
						>
							<div class="min-w-0">
								<p class="text-sm font-mono text-base-content/70 truncate">{perm.key}</p>
								<p class="text-xs text-base-content/40 truncate">{perm.description}</p>
							</div>
							<span class={overrideBadgeClass(overrides[perm.key] ?? null)}>
								{overrideBadgeLabel(overrides[perm.key] ?? null)}
							</span>
						</div>
					{:else}
						<label
							class="flex items-center gap-3 px-3 py-2 rounded-lg hover:bg-base-200/50 transition-colors cursor-pointer"
							class:opacity-60={readonly}
							class:cursor-not-allowed={readonly}
						>
							<input
								type="checkbox"
								class="checkbox checkbox-primary checkbox-sm shrink-0"
								checked={selected.includes(perm.key)}
								disabled={readonly}
								on:change={() => togglePermission(perm.key)}
							/>
							<div class="min-w-0">
								<p class="text-sm font-mono text-base-content/70 truncate">{perm.key}</p>
								<p class="text-xs text-base-content/40 truncate">{perm.description}</p>
							</div>
						</label>
					{/if}
				{/each}
			</div>
		</div>
	{/each}

	{#if permissions.length === 0}
		<p class="text-sm text-base-content/40 text-center py-4">{$_('roles.noPermissions')}</p>
	{/if}
</div>
