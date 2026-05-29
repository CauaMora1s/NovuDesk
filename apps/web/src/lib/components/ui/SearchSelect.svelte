<script lang="ts">
	import { tick } from 'svelte';
	import { _ } from 'svelte-i18n';

	export interface SearchSelectOption {
		value: string;
		label: string;
		sublabel?: string;
		avatar?: boolean;
	}

	export let options: SearchSelectOption[] = [];
	export let value = '';
	export let placeholder: string | undefined = undefined;
	export let searchPlaceholder: string | undefined = undefined;
	export let noResultsText: string | undefined = undefined;
	export let emptyLabel: string | undefined = undefined;
	export let disabled = false;
	export let size: 'sm' | 'md' = 'sm';
	export let onChange: ((val: string) => void) | undefined = undefined;

	let open = false;
	let query = '';
	let inputEl: HTMLInputElement;
	let containerEl: HTMLElement;
	let listEl: HTMLUListElement;
	let highlightedIndex = -1;

	$: selected = options.find((o) => o.value === value) ?? null;

	$: filtered = query.trim()
		? options.filter(
				(o) =>
					o.label.toLowerCase().includes(query.toLowerCase()) ||
					(o.sublabel?.toLowerCase().includes(query.toLowerCase()) ?? false)
			)
		: options;

	$: if (filtered) highlightedIndex = -1;

	async function openMenu() {
		if (disabled) return;
		open = true;
		query = '';
		await tick();
		inputEl?.focus();
	}

	function closeMenu() {
		open = false;
		query = '';
		highlightedIndex = -1;
	}

	async function toggleMenu() {
		if (disabled) return;
		if (open) {
			closeMenu();
		} else {
			await openMenu();
		}
	}

	function select(opt: SearchSelectOption | null) {
		value = opt?.value ?? '';
		onChange?.(value);
		closeMenu();
	}

	function onTriggerKeyDown(e: KeyboardEvent) {
		if (e.key === 'Enter' || e.key === ' ' || e.key === 'ArrowDown') {
			e.preventDefault();
			openMenu();
		} else if (e.key === 'Escape') {
			closeMenu();
		}
	}

	function onSearchKeyDown(e: KeyboardEvent) {
		const hasEmpty = emptyLabel !== undefined;
		const listLength = filtered.length;

		if (e.key === 'Escape') { e.stopPropagation(); closeMenu(); return; }
		if (e.key === 'ArrowDown') {
			e.preventDefault();
			highlightedIndex = Math.min(highlightedIndex + 1, listLength - 1);
			scrollHighlighted();
		} else if (e.key === 'ArrowUp') {
			e.preventDefault();
			highlightedIndex = Math.max(highlightedIndex - 1, hasEmpty ? -1 : 0);
			scrollHighlighted();
		} else if (e.key === 'Enter') {
			e.preventDefault();
			if (highlightedIndex === -1 && hasEmpty) select(null);
			else if (highlightedIndex >= 0) select(filtered[highlightedIndex]);
		}
	}

	function scrollHighlighted() {
		const el = listEl?.querySelector(`[data-idx="${highlightedIndex}"]`) as HTMLElement | null;
		el?.scrollIntoView({ block: 'nearest' });
	}

	// Window-level click handler: fires for any click that reaches window.
	// Internal clicks are stopped by on:click|stopPropagation on the dropdown wrapper,
	// so only external clicks (outside containerEl) reach here.
	function onWindowClick(e: MouseEvent) {
		if (!open) return;
		if (containerEl && containerEl.contains(e.target as Node)) return;
		closeMenu();
	}

	function initials(name: string) {
		return name.split(' ').slice(0, 2).map((n) => n[0]).join('').toUpperCase();
	}

	$: h = size === 'sm' ? 'h-8 text-sm px-3' : 'h-10 text-sm px-4';
</script>

<svelte:window on:click={onWindowClick} />

<div class="relative w-full" bind:this={containerEl}>
	<!-- ── Trigger ── -->
	<button
		type="button"
		class="w-full flex items-center justify-between gap-2 border border-base-300 rounded-lg bg-base-100
		       transition-colors hover:border-base-content/30 focus:outline-none focus:border-primary
		       {h} {disabled ? 'opacity-50 cursor-not-allowed' : 'cursor-pointer'}"
		{disabled}
		on:click={toggleMenu}
		on:keydown={onTriggerKeyDown}
	>
		<span class="truncate flex-1 text-left {selected ? 'text-base-content' : 'text-base-content/40'}">
			{#if selected?.avatar}
				<span class="inline-flex items-center gap-1.5">
					<span class="w-4 h-4 rounded-full bg-primary/10 text-primary text-[9px] font-bold
					             inline-flex items-center justify-center shrink-0">
						{initials(selected.label)}
					</span>
					{selected.label}
				</span>
			{:else}
				{selected?.label ?? (placeholder ?? $_('common.select'))}
			{/if}
		</span>
		<svg
			class="w-3.5 h-3.5 shrink-0 text-base-content/40 transition-transform {open ? 'rotate-180' : ''}"
			fill="none" viewBox="0 0 24 24" stroke="currentColor"
		>
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
		</svg>
	</button>

	<!-- ── Dropdown ── -->
	{#if open}
		<!-- stopPropagation prevents internal clicks from reaching svelte:window and closing the menu -->
		<div
			class="absolute z-50 top-full mt-1 left-0 right-0 min-w-[220px] bg-base-100 border border-base-300
			       rounded-box shadow-xl overflow-hidden"
			on:click|stopPropagation
		>
			<!-- Search input -->
			<div class="p-2 border-b border-base-200">
				<label class="flex items-center gap-2 px-2 py-1 border border-base-300 rounded-lg bg-base-50
				              focus-within:border-primary transition-colors">
					<svg class="w-3.5 h-3.5 text-base-content/40 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
							d="M21 21l-4.35-4.35M17 11A6 6 0 1 1 5 11a6 6 0 0 1 12 0z" />
					</svg>
					<input
						bind:this={inputEl}
						bind:value={query}
						type="text"
						class="grow text-sm bg-transparent outline-none placeholder:text-base-content/40"
						placeholder={searchPlaceholder ?? $_('common.search')}
						on:keydown={onSearchKeyDown}
					/>
					{#if query}
						<button
							type="button"
							class="text-base-content/40 hover:text-base-content"
							on:click|stopPropagation={() => (query = '')}
						>
							<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
							</svg>
						</button>
					{/if}
				</label>
			</div>

			<!-- Options list -->
			<ul bind:this={listEl} class="max-h-52 overflow-y-auto py-1">
				{#if emptyLabel !== undefined}
					<li>
						<button
							type="button"
							data-idx="-1"
							class="w-full text-left px-3 py-2 text-sm transition-colors
							       {!value ? 'bg-primary/5 text-primary font-medium' : 'text-base-content/50 hover:bg-base-200'}"
							on:click={() => select(null)}
						>
							{emptyLabel || '—'}
						</button>
					</li>
				{/if}

				{#each filtered as opt, i}
					<li>
						<button
							type="button"
							data-idx={i}
							class="w-full text-left px-3 py-2 flex items-center gap-2.5 transition-colors
							       {opt.value === value ? 'bg-primary/10 text-primary' : 'text-base-content hover:bg-base-200'}
							       {highlightedIndex === i ? 'bg-base-200' : ''}"
							on:click={() => select(opt)}
						>
							{#if opt.avatar}
								<span class="w-6 h-6 rounded-full bg-primary/10 text-primary text-xs font-semibold
								             flex items-center justify-center shrink-0">
									{initials(opt.label)}
								</span>
							{/if}
							<span class="min-w-0 flex-1">
								<span class="block text-sm font-medium truncate">{opt.label}</span>
								{#if opt.sublabel}
									<span class="block text-xs text-base-content/50 truncate">{opt.sublabel}</span>
								{/if}
							</span>
							{#if opt.value === value}
								<svg class="w-3.5 h-3.5 text-primary shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7" />
								</svg>
							{/if}
						</button>
					</li>
				{:else}
					<li class="px-3 py-5 text-center text-sm text-base-content/40">{noResultsText ?? $_('common.noResults')}</li>
				{/each}
			</ul>
		</div>
	{/if}
</div>
