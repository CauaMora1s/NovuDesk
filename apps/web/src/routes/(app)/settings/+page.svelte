<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { can } from '$lib/permissions';
	import { membersApi, type Member, type Role } from '$lib/api/members';
	import { teamsApi, type Team } from '$lib/api/teams';

	let activeTab = 'organization';

	const tabs = [
		{ key: 'organization', label: 'settings.organization', permission: 'organization:view_settings' },
		{ key: 'members',      label: 'settings.members',      permission: 'users:view' },
		{ key: 'roles',        label: 'settings.roles',        permission: 'organization:view_settings' },
		{ key: 'sla',          label: 'settings.sla',          permission: 'sla:view' }
	] as const;

	$: visibleTabs = tabs.filter((t) => can(t.permission));

	// ── Members state ──────────────────────────────────────────
	let members: Member[] = [];
	let roles: Role[] = [];
	let teams: Team[] = [];
	let membersLoading = false;
	let membersError = '';

	let showCreateModal = false;
	let showEditModal = false;
	let editTarget: Member | null = null;

	let form = { full_name: '', email: '', password: '', role_id: '', team_id: '' };
	let editForm = { role_id: '' };
	let saving = false;
	let saveError = '';
	let showPassword = false;

	async function loadMembers() {
		membersLoading = true;
		membersError = '';
		try {
			[members, roles, teams] = await Promise.all([
				membersApi.list(),
				membersApi.listRoles(),
				teamsApi.list()
			]);
		} catch {
			membersError = $_('settings.loadError');
		} finally {
			membersLoading = false;
		}
	}

	$: if (activeTab === 'members') {
		loadMembers();
	}

	function openCreate() {
		form = { full_name: '', email: '', password: '', role_id: roles[0]?.id ?? '', team_id: '' };
		saveError = '';
		showCreateModal = true;
	}

	function openEdit(m: Member) {
		editTarget = m;
		editForm = { role_id: m.role_id };
		saveError = '';
		showEditModal = true;
	}

	async function createMember() {
		saving = true;
		saveError = '';
		try {
			const created = await membersApi.create({
				full_name: form.full_name,
				email: form.email,
				password: form.password,
				role_id: form.role_id,
				team_id: form.team_id || undefined
			});
			members = [...members, created];
			showCreateModal = false;
		} catch (err: unknown) {
			saveError = $_('settings.createMemberError');
		} finally {
			saving = false;
		}
	}

	async function updateRole() {
		if (!editTarget) return;
		saving = true;
		saveError = '';
		try {
			const updated = await membersApi.updateRole(editTarget.id, { role_id: editForm.role_id });
			members = members.map((m) => (m.id === updated.id ? updated : m));
			showEditModal = false;
		} catch {
			saveError = $_('settings.updateRoleError');
		} finally {
			saving = false;
		}
	}

	async function deactivate(m: Member) {
		if (!confirm($_('settings.confirmDeactivate', { values: { name: m.full_name } }))) return;
		try {
			await membersApi.deactivate(m.id);
			members = members.map((mb) => (mb.id === m.id ? { ...mb, is_active: false } : mb));
		} catch { /* ignore */ }
	}

	function initials(name: string) {
		return name.split(' ').slice(0, 2).map((n) => n[0]).join('').toUpperCase();
	}

	function roleBadgeClass(roleName: string) {
		const map: Record<string, string> = {
			owner: 'badge-error',
			admin: 'badge-warning',
			agent: 'badge-info',
			viewer: 'badge-ghost'
		};
		return `badge badge-sm ${map[roleName] ?? 'badge-ghost'}`;
	}
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
					<p class="text-sm text-base-content/50">{$_('settings.orgUnderConstruction')}</p>

				{:else if activeTab === 'members'}
					<!-- Header -->
					<div class="flex items-center justify-between mb-5">
						<h2 class="font-semibold">{$_('settings.members')}</h2>
						{#if can('users:invite')}
							<button class="btn btn-primary btn-sm gap-2" on:click={openCreate}>
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
								</svg>
								{$_('settings.addMemberButton')}
							</button>
						{/if}
					</div>

					{#if membersLoading}
						<div class="flex justify-center py-10">
							<span class="loading loading-spinner loading-md text-primary"></span>
						</div>
					{:else if membersError}
						<div class="alert alert-error text-sm">{membersError}</div>
					{:else if members.length === 0}
						<div class="text-center py-12 text-base-content/40">
							<p class="text-sm">{$_('settings.noMembers')}</p>
						</div>
					{:else}
						<div class="overflow-x-auto">
							<table class="table table-sm w-full">
								<thead>
									<tr class="text-xs text-base-content/50 border-b border-base-200">
										<th class="font-medium">{$_('settings.memberColumn')}</th>
										<th class="font-medium">{$_('settings.roleColumn')}</th>
										<th class="font-medium">{$_('settings.statusColumn')}</th>
										{#if can('users:manage_roles') || can('users:deactivate')}
											<th class="font-medium text-right">{$_('common.actions')}</th>
										{/if}
									</tr>
								</thead>
								<tbody>
									{#each members as m}
										<tr class="border-b border-base-200/50 hover:bg-base-200/30 transition-colors">
											<td>
												<div class="flex items-center gap-3">
													<div class="avatar placeholder">
														<div class="w-8 h-8 rounded-full bg-primary/10 text-primary text-xs font-semibold flex items-center justify-center">
															{initials(m.full_name)}
														</div>
													</div>
													<div class="min-w-0">
														<p class="text-sm font-medium truncate">{m.full_name}</p>
														<p class="text-xs text-base-content/50 truncate">{m.email}</p>
													</div>
												</div>
											</td>
											<td>
												<span class={roleBadgeClass(m.role_name)}>{m.role_name}</span>
											</td>
											<td>
												<span class="badge badge-sm {m.is_active ? 'badge-success' : 'badge-ghost'} gap-1">
													<span class="w-1.5 h-1.5 rounded-full {m.is_active ? 'bg-success' : 'bg-base-content/30'}"></span>
													{m.is_active ? $_('members.status.active') : $_('members.status.inactive')}
												</span>
											</td>
											{#if can('users:manage_roles') || can('users:deactivate')}
												<td class="text-right">
													<div class="flex justify-end gap-1">
														{#if can('users:manage_roles')}
															<button
																class="btn btn-ghost btn-xs"
																on:click={() => openEdit(m)}
															>{$_('settings.editButton')}</button>
														{/if}
														{#if can('users:deactivate') && m.is_active}
															<button
																class="btn btn-ghost btn-xs text-error"
																on:click={() => deactivate(m)}
															>{$_('settings.deactivateButton')}</button>
														{/if}
													</div>
												</td>
											{/if}
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					{/if}

				{:else if activeTab === 'roles'}
					<h2 class="font-semibold mb-4">{$_('settings.roles')}</h2>
					<p class="text-sm text-base-content/50">{$_('settings.rolesUnderConstruction')}</p>

				{:else if activeTab === 'sla'}
					<h2 class="font-semibold mb-4">{$_('settings.sla')}</h2>
					<p class="text-sm text-base-content/50">{$_('settings.slaUnderConstruction')}</p>
				{/if}

			</div>
		</div>
	</div>
</div>

<!-- ── Modal: Criar Membro ── -->
{#if showCreateModal}
	<dialog class="modal modal-open">
		<div class="modal-box max-w-md">
			<button
				class="btn btn-sm btn-circle btn-ghost absolute right-3 top-3"
				on:click={() => (showCreateModal = false)}
			>✕</button>

			<h3 class="font-bold text-lg mb-5">{$_('settings.modalAddMember')}</h3>

			{#if saveError}
				<div class="alert alert-error text-sm mb-4">{saveError}</div>
			{/if}

			<form on:submit|preventDefault={createMember} class="space-y-4">
				<div class="form-control">
					<label class="label pb-1" for="full_name">
						<span class="label-text font-medium">{$_('members.name')}</span>
					</label>
					<input
						id="full_name"
						type="text"
						bind:value={form.full_name}
						class="input input-bordered"
						placeholder={$_('settings.fullNamePlaceholder')}
						required
					/>
				</div>

				<div class="form-control">
					<label class="label pb-1" for="email_new">
						<span class="label-text font-medium">{$_('members.email')}</span>
					</label>
					<input
						id="email_new"
						type="email"
						bind:value={form.email}
						class="input input-bordered"
						placeholder={$_('settings.emailPlaceholder')}
						required
					/>
				</div>

				<div class="form-control">
					<label class="label pb-1" for="password_new">
						<span class="label-text font-medium">{$_('members.password')}</span>
					</label>
					<div class="relative">
						<input
							id="password_new"
							type={showPassword ? 'text' : 'password'}
							bind:value={form.password}
							class="input input-bordered w-full pr-10"
							placeholder={$_('settings.passwordPlaceholder')}
							required
							minlength="8"
						/>
						<button
							type="button"
							class="absolute right-3 top-1/2 -translate-y-1/2 text-base-content/40 hover:text-base-content"
							on:click={() => (showPassword = !showPassword)}
						>
							{#if showPassword}
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
										d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" />
								</svg>
							{:else}
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
										d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
										d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
								</svg>
							{/if}
						</button>
					</div>
				</div>

				<div class="form-control">
					<label class="label pb-1" for="role_select">
						<span class="label-text font-medium">{$_('members.role')}</span>
					</label>
					<select id="role_select" bind:value={form.role_id} class="select select-bordered" required>
						{#each roles as r}
							<option value={r.id}>{r.name}</option>
						{/each}
					</select>
				</div>

				<div class="form-control">
					<label class="label pb-1" for="team_select">
						<span class="label-text font-medium">{$_('settings.teamOptional')}</span>
					</label>
					<select id="team_select" bind:value={form.team_id} class="select select-bordered">
						<option value="">{$_('settings.noTeamOption')}</option>
						{#each teams as t}
							<option value={t.id}>{t.name}</option>
						{/each}
					</select>
					<label class="label pt-1">
						<span class="label-text-alt text-base-content/40">{$_('settings.noTeamHint')}</span>
					</label>
				</div>

				<div class="modal-action mt-6">
					<button type="button" class="btn btn-ghost" on:click={() => (showCreateModal = false)}>
						{$_('common.cancel')}
					</button>
					<button type="submit" class="btn btn-primary" disabled={saving}>
						{#if saving}
							<span class="loading loading-spinner loading-xs"></span>
						{/if}
						{$_('settings.createMemberButton')}
					</button>
				</div>
			</form>
		</div>
		<div class="modal-backdrop" on:click={() => (showCreateModal = false)}></div>
	</dialog>
{/if}

<!-- ── Modal: Editar Membro ── -->
{#if showEditModal && editTarget}
	<dialog class="modal modal-open">
		<div class="modal-box max-w-sm">
			<button
				class="btn btn-sm btn-circle btn-ghost absolute right-3 top-3"
				on:click={() => (showEditModal = false)}
			>✕</button>

			<h3 class="font-bold text-lg mb-1">{$_('settings.modalEditMember')}</h3>
			<p class="text-sm text-base-content/50 mb-5">{editTarget.full_name}</p>

			{#if saveError}
				<div class="alert alert-error text-sm mb-4">{saveError}</div>
			{/if}

			<form on:submit|preventDefault={updateRole} class="space-y-4">
				<div class="form-control">
					<label class="label pb-1" for="edit_role">
						<span class="label-text font-medium">{$_('members.role')}</span>
					</label>
					<select id="edit_role" bind:value={editForm.role_id} class="select select-bordered" required>
						{#each roles as r}
							<option value={r.id}>{r.name}</option>
						{/each}
					</select>
				</div>

				<div class="modal-action mt-6">
					<button type="button" class="btn btn-ghost" on:click={() => (showEditModal = false)}>
						{$_('common.cancel')}
					</button>
					<button type="submit" class="btn btn-primary" disabled={saving}>
						{#if saving}
							<span class="loading loading-spinner loading-xs"></span>
						{/if}
						{$_('common.save')}
					</button>
				</div>
			</form>
		</div>
		<div class="modal-backdrop" on:click={() => (showEditModal = false)}></div>
	</dialog>
{/if}
