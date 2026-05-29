<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { can } from '$lib/permissions';
	import { teamsApi, type Team, type TeamMember, type TeamDetail } from '$lib/api/teams';
	import { categoriesApi, type Category } from '$lib/api/categories';
	import { membersApi, type Member } from '$lib/api/members';
	import SearchSelect from '$lib/components/ui/SearchSelect.svelte';
	import type { SearchSelectOption } from '$lib/components/ui/SearchSelect.svelte';

	let teams: Team[] = [];
	let loading = true;
	let error = '';

	let selectedTeam: Team | null = null;
	let detail: TeamDetail | null = null;
	let detailLoading = false;

	let panelTab: 'members' | 'categories' = 'members';

	// Org data for selects
	let orgMembers: Member[] = [];
	let orgCategories: Category[] = [];

	// Modals
	let showNewTeamModal = false;
	let showAddMemberModal = false;
	let showNewCategoryModal = false;
	let showAddExistingCategoryModal = false;

	let newTeamForm = { name: '', description: '' };
	let newCategoryForm = { name: '', description: '' };
	let selectedMemberToAdd = '';
	let selectedCategoryToAdd = '';
	let saving = false;
	let saveError = '';

	$: loadTeams();

	async function loadTeams() {
		loading = true;
		error = '';
		try {
			teams = await teamsApi.list();
		} catch {
			error = $_('teams.loadError');
		} finally {
			loading = false;
		}
	}

	async function selectTeam(t: Team) {
		selectedTeam = t;
		detail = null;
		detailLoading = true;
		panelTab = 'members';
		try {
			detail = await teamsApi.get(t.id);
		} catch {
			// ignore
		} finally {
			detailLoading = false;
		}
	}

	// ── Create team ───────────────────────────────────────────
	async function createTeam() {
		saving = true;
		saveError = '';
		try {
			const t = await teamsApi.create({ name: newTeamForm.name, description: newTeamForm.description || undefined });
			teams = [...teams, t];
			showNewTeamModal = false;
			newTeamForm = { name: '', description: '' };
			await selectTeam(t);
		} catch {
			saveError = $_('teams.createError');
		} finally {
			saving = false;
		}
	}

	// ── Add member to team ────────────────────────────────────
	async function openAddMember() {
		orgMembers = await membersApi.list();
		selectedMemberToAdd = '';
		showAddMemberModal = true;
	}

	async function addMember() {
		if (!selectedTeam || !selectedMemberToAdd) return;
		saving = true;
		saveError = '';
		try {
			await teamsApi.addMember(selectedTeam.id, selectedMemberToAdd);
			detail = await teamsApi.get(selectedTeam.id);
			showAddMemberModal = false;
		} catch {
			saveError = $_('teams.addMemberError');
		} finally {
			saving = false;
		}
	}

	async function removeMember(userId: string) {
		if (!selectedTeam || !confirm($_('teams.confirmRemoveMember'))) return;
		try {
			await teamsApi.removeMember(selectedTeam.id, userId);
			if (detail) {
				detail = { ...detail, members: detail.members.filter((m) => m.user_id !== userId) };
			}
		} catch { /* ignore */ }
	}

	// ── Categories ────────────────────────────────────────────
	async function openNewCategory() {
		newCategoryForm = { name: '', description: '' };
		saveError = '';
		showNewCategoryModal = true;
	}

	async function createAndAddCategory() {
		if (!selectedTeam) return;
		saving = true;
		saveError = '';
		try {
			const cat = await categoriesApi.create({ name: newCategoryForm.name, description: newCategoryForm.description || undefined });
			await teamsApi.addCategory(selectedTeam.id, cat.id);
			detail = await teamsApi.get(selectedTeam.id);
			showNewCategoryModal = false;
		} catch {
			saveError = $_('teams.addCategoryError');
		} finally {
			saving = false;
		}
	}

	async function openAddExistingCategory() {
		orgCategories = await categoriesApi.list();
		selectedCategoryToAdd = '';
		saveError = '';
		showAddExistingCategoryModal = true;
	}

	async function addExistingCategory() {
		if (!selectedTeam || !selectedCategoryToAdd) return;
		saving = true;
		saveError = '';
		try {
			await teamsApi.addCategory(selectedTeam.id, selectedCategoryToAdd);
			detail = await teamsApi.get(selectedTeam.id);
			showAddExistingCategoryModal = false;
		} catch {
			saveError = $_('teams.linkCategoryError');
		} finally {
			saving = false;
		}
	}

	async function removeCategory(catId: string) {
		if (!selectedTeam || !confirm($_('teams.confirmRemoveCategory'))) return;
		try {
			await teamsApi.removeCategory(selectedTeam.id, catId);
			if (detail) {
				detail = { ...detail, categories: detail.categories.filter((c) => c.id !== catId) };
			}
		} catch { /* ignore */ }
	}

	$: memberOptions = orgMembers.map((m): SearchSelectOption => ({
		value: m.id,
		label: m.full_name,
		sublabel: m.email,
		avatar: true
	}));

	$: categoryOptions = orgCategories.map((c): SearchSelectOption => ({
		value: c.id,
		label: c.name,
		sublabel: c.description ?? undefined
	}));

	function initials(name: string) {
		return name.split(' ').slice(0, 2).map((n) => n[0]).join('').toUpperCase();
	}
</script>

<svelte:head><title>{$_('teams.title')} — NovuDesk</title></svelte:head>

<div class="p-8 max-w-6xl mx-auto">
	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold">{$_('teams.title')}</h1>
		{#if can('teams:manage')}
			<button class="btn btn-primary btn-sm gap-2" on:click={() => (showNewTeamModal = true)}>
				<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
				</svg>
				{$_('teams.new')}
			</button>
		{/if}
	</div>

	{#if loading}
		<div class="flex justify-center py-20">
			<span class="loading loading-spinner loading-lg text-primary"></span>
		</div>
	{:else if error}
		<div class="alert alert-error">{error}</div>
	{:else}
		<div class="grid grid-cols-1 lg:grid-cols-3 gap-4 items-start">

			<!-- Team list -->
			<div class="card bg-base-100 shadow-card lg:col-span-1">
				<div class="card-body p-3">
					{#if teams.length === 0}
						<div class="flex flex-col items-center justify-center py-12 text-base-content/40">
							<svg class="w-8 h-8 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
									d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0z" />
							</svg>
							<p class="text-sm font-medium">{$_('teams.noTeams')}</p>
						</div>
					{:else}
						<ul class="space-y-0.5">
							{#each teams as t}
								<li>
									<button
										class="w-full flex items-center gap-3 px-3 py-2.5 rounded-lg text-left transition-colors hover:bg-base-200"
										class:bg-primary={selectedTeam?.id === t.id}
										class:text-primary-content={selectedTeam?.id === t.id}
										on:click={() => selectTeam(t)}
									>
										<div class="w-8 h-8 rounded-lg flex items-center justify-center text-xs font-bold shrink-0 {selectedTeam?.id === t.id ? 'bg-primary-content/20 text-primary-content' : 'bg-primary/10 text-primary'}"
										>
											{t.name.slice(0, 2).toUpperCase()}
										</div>
										<div class="min-w-0 flex-1">
											<p class="text-sm font-medium truncate">{t.name}</p>
											{#if t.description}
												<p class="text-xs opacity-60 truncate">{t.description}</p>
											{/if}
										</div>
									</button>
								</li>
							{/each}
						</ul>
					{/if}
				</div>
			</div>

			<!-- Detail panel -->
			<div class="card bg-base-100 shadow-card lg:col-span-2 min-h-64">
				<div class="card-body p-6">
					{#if !selectedTeam}
						<div class="flex flex-col items-center justify-center h-full py-16 text-base-content/40">
							<svg class="w-8 h-8 mb-2" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5"
									d="M15 19l-7-7 7-7" />
							</svg>
							<p class="text-sm">{$_('teams.selectTeamPrompt')}</p>
						</div>

					{:else if detailLoading}
						<div class="flex justify-center py-10">
							<span class="loading loading-spinner loading-md text-primary"></span>
						</div>

					{:else if detail}
						<!-- Team header -->
						<div class="mb-4">
							<h2 class="text-lg font-bold">{detail.team.name}</h2>
							{#if detail.team.description}
								<p class="text-sm text-base-content/60 mt-0.5">{detail.team.description}</p>
							{/if}
						</div>

						<!-- Tabs -->
						<div class="tabs tabs-boxed mb-4 w-fit">
							<button
								class="tab tab-sm"
								class:tab-active={panelTab === 'members'}
								on:click={() => (panelTab = 'members')}
							>
								{$_('teams.members')} <span class="ml-1 badge badge-xs">{detail.members.length}</span>
							</button>
							<button
								class="tab tab-sm"
								class:tab-active={panelTab === 'categories'}
								on:click={() => (panelTab = 'categories')}
							>
								{$_('teams.categories')} <span class="ml-1 badge badge-xs">{detail.categories.length}</span>
							</button>
						</div>

						{#if panelTab === 'members'}
							<div class="flex items-center justify-between mb-3">
								<p class="text-xs text-base-content/50 font-medium uppercase tracking-wide">
									{detail.members.length} {detail.members.length === 1 ? $_('teams.memberSingular') : $_('teams.memberPlural')}
								</p>
								{#if can('teams:manage')}
									<button class="btn btn-outline btn-xs gap-1" on:click={openAddMember}>
										<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
										</svg>
										{$_('teams.addButton')}
									</button>
								{/if}
							</div>

							{#if detail.members.length === 0}
								<p class="text-sm text-base-content/40 py-4 text-center">{$_('teams.noMembers')}</p>
							{:else}
								<ul class="space-y-2">
									{#each detail.members as m}
										<li class="flex items-center justify-between gap-3 py-2 border-b border-base-200/50 last:border-0">
											<div class="flex items-center gap-3">
												<div class="avatar placeholder">
													<div class="w-8 h-8 rounded-full bg-primary/10 text-primary text-xs font-semibold flex items-center justify-center">
														{initials(m.full_name)}
													</div>
												</div>
												<div>
													<p class="text-sm font-medium">{m.full_name}</p>
													<p class="text-xs text-base-content/50">{m.email}</p>
												</div>
											</div>
											{#if can('teams:manage')}
												<button
													class="btn btn-ghost btn-xs text-error"
													on:click={() => removeMember(m.user_id)}
												>{$_('teams.removeMember')}</button>
											{/if}
										</li>
									{/each}
								</ul>
							{/if}

						{:else}
							<!-- Categories panel -->
							<div class="flex items-center justify-between mb-3">
								<p class="text-xs text-base-content/50 font-medium uppercase tracking-wide">
									{detail.categories.length} {detail.categories.length === 1 ? $_('teams.categorySingular') : $_('teams.categoryPlural')}
								</p>
								{#if can('teams:manage')}
									<div class="flex gap-1">
										<button class="btn btn-outline btn-xs" on:click={openAddExistingCategory}>
											{$_('teams.linkButton')}
										</button>
										<button class="btn btn-outline btn-xs gap-1" on:click={openNewCategory}>
											<svg class="w-3 h-3" fill="none" viewBox="0 0 24 24" stroke="currentColor">
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
											</svg>
											{$_('teams.newButton')}
										</button>
									</div>
								{/if}
							</div>

							{#if detail.categories.length === 0}
								<p class="text-sm text-base-content/40 py-4 text-center">{$_('teams.noCategories')}</p>
							{:else}
								<ul class="space-y-2">
									{#each detail.categories as cat}
										<li class="flex items-center justify-between gap-3 py-2 border-b border-base-200/50 last:border-0">
											<div class="flex items-center gap-2">
												<span class="w-2 h-2 rounded-full bg-primary"></span>
												<div>
													<p class="text-sm font-medium">{cat.name}</p>
													{#if cat.description}
														<p class="text-xs text-base-content/50">{cat.description}</p>
													{/if}
												</div>
											</div>
											{#if can('teams:manage')}
												<button
													class="btn btn-ghost btn-xs text-error"
													on:click={() => removeCategory(cat.id)}
												>{$_('teams.removeMember')}</button>
											{/if}
										</li>
									{/each}
								</ul>
							{/if}
						{/if}
					{/if}
				</div>
			</div>
		</div>
	{/if}
</div>

<!-- ── Modal: Novo Time ── -->
{#if showNewTeamModal}
	<dialog class="modal modal-open">
		<div class="modal-box max-w-sm">
			<button class="btn btn-sm btn-circle btn-ghost absolute right-3 top-3" on:click={() => (showNewTeamModal = false)}>✕</button>
			<h3 class="font-bold text-lg mb-5">{$_('teams.modalNewTeam')}</h3>

			{#if saveError}
				<div class="alert alert-error text-sm mb-4">{saveError}</div>
			{/if}

			<form on:submit|preventDefault={createTeam} class="space-y-4">
				<div class="form-control">
					<label class="label pb-1" for="team_name">
						<span class="label-text font-medium">{$_('teams.teamNameLabel')}</span>
					</label>
					<input id="team_name" type="text" bind:value={newTeamForm.name}
						class="input input-bordered" placeholder={$_('teams.teamNamePlaceholder')} required />
				</div>
				<div class="form-control">
					<label class="label pb-1" for="team_desc">
						<span class="label-text font-medium">{$_('teams.descriptionLabel')} <span class="text-base-content/40 font-normal">{$_('common.optional')}</span></span>
					</label>
					<input id="team_desc" type="text" bind:value={newTeamForm.description}
						class="input input-bordered" placeholder={$_('teams.teamDescPlaceholder')} />
				</div>
				<div class="modal-action">
					<button type="button" class="btn btn-ghost" on:click={() => (showNewTeamModal = false)}>{$_('common.cancel')}</button>
					<button type="submit" class="btn btn-primary" disabled={saving}>
						{#if saving}<span class="loading loading-spinner loading-xs"></span>{/if}
						{$_('teams.modalNewTeam')}
					</button>
				</div>
			</form>
		</div>
		<div class="modal-backdrop" on:click={() => (showNewTeamModal = false)}></div>
	</dialog>
{/if}

<!-- ── Modal: Adicionar Membro ao Time ── -->
{#if showAddMemberModal}
	<dialog class="modal modal-open">
		<div class="modal-box max-w-sm">
			<button class="btn btn-sm btn-circle btn-ghost absolute right-3 top-3" on:click={() => (showAddMemberModal = false)}>✕</button>
			<h3 class="font-bold text-lg mb-5">{$_('teams.addMember')}</h3>

			{#if saveError}
				<div class="alert alert-error text-sm mb-4">{saveError}</div>
			{/if}

			<div class="form-control mb-4">
				<label class="label pb-1">
					<span class="label-text font-medium">{$_('teams.selectMember')}</span>
				</label>
				<SearchSelect
					options={memberOptions}
					bind:value={selectedMemberToAdd}
					placeholder={$_('teams.selectMemberPlaceholder')}
					searchPlaceholder="Pesquisar por nome ou e-mail..."
					size="md"
				/>
			</div>

			<div class="modal-action">
				<button class="btn btn-ghost" on:click={() => (showAddMemberModal = false)}>{$_('common.cancel')}</button>
				<button class="btn btn-primary" on:click={addMember} disabled={saving || !selectedMemberToAdd}>
					{#if saving}<span class="loading loading-spinner loading-xs"></span>{/if}
					{$_('teams.addButton')}
				</button>
			</div>
		</div>
		<div class="modal-backdrop" on:click={() => (showAddMemberModal = false)}></div>
	</dialog>
{/if}

<!-- ── Modal: Nova Categoria ── -->
{#if showNewCategoryModal}
	<dialog class="modal modal-open">
		<div class="modal-box max-w-sm">
			<button class="btn btn-sm btn-circle btn-ghost absolute right-3 top-3" on:click={() => (showNewCategoryModal = false)}>✕</button>
			<h3 class="font-bold text-lg mb-5">{$_('teams.newCategory')}</h3>

			{#if saveError}
				<div class="alert alert-error text-sm mb-4">{saveError}</div>
			{/if}

			<form on:submit|preventDefault={createAndAddCategory} class="space-y-4">
				<div class="form-control">
					<label class="label pb-1" for="cat_name">
						<span class="label-text font-medium">{$_('teams.categoryName')}</span>
					</label>
					<input id="cat_name" type="text" bind:value={newCategoryForm.name}
						class="input input-bordered" placeholder={$_('teams.categoryPlaceholder')} required />
				</div>
				<div class="form-control">
					<label class="label pb-1" for="cat_desc">
						<span class="label-text font-medium">{$_('teams.descriptionLabel')} <span class="text-base-content/40 font-normal">{$_('common.optional')}</span></span>
					</label>
					<input id="cat_desc" type="text" bind:value={newCategoryForm.description}
						class="input input-bordered" placeholder={$_('teams.descPlaceholder')} />
				</div>
				<div class="modal-action">
					<button type="button" class="btn btn-ghost" on:click={() => (showNewCategoryModal = false)}>{$_('common.cancel')}</button>
					<button type="submit" class="btn btn-primary" disabled={saving}>
						{#if saving}<span class="loading loading-spinner loading-xs"></span>{/if}
						{$_('teams.createAndLink')}
					</button>
				</div>
			</form>
		</div>
		<div class="modal-backdrop" on:click={() => (showNewCategoryModal = false)}></div>
	</dialog>
{/if}

<!-- ── Modal: Vincular Categoria Existente ── -->
{#if showAddExistingCategoryModal}
	<dialog class="modal modal-open">
		<div class="modal-box max-w-sm">
			<button class="btn btn-sm btn-circle btn-ghost absolute right-3 top-3" on:click={() => (showAddExistingCategoryModal = false)}>✕</button>
			<h3 class="font-bold text-lg mb-5">{$_('teams.newCategory')}</h3>

			{#if saveError}
				<div class="alert alert-error text-sm mb-4">{saveError}</div>
			{/if}

			<div class="form-control mb-4">
				<label class="label pb-1">
					<span class="label-text font-medium">{$_('teams.selectCategory')}</span>
				</label>
				<SearchSelect
					options={categoryOptions}
					bind:value={selectedCategoryToAdd}
					placeholder={$_('teams.selectCategoryPlaceholder')}
					searchPlaceholder="Pesquisar categoria..."
					size="md"
				/>
			</div>

			<div class="modal-action">
				<button class="btn btn-ghost" on:click={() => (showAddExistingCategoryModal = false)}>{$_('common.cancel')}</button>
				<button class="btn btn-primary" on:click={addExistingCategory} disabled={saving || !selectedCategoryToAdd}>
					{#if saving}<span class="loading loading-spinner loading-xs"></span>{/if}
					{$_('teams.linkButton')}
				</button>
			</div>
		</div>
		<div class="modal-backdrop" on:click={() => (showAddExistingCategoryModal = false)}></div>
	</dialog>
{/if}
