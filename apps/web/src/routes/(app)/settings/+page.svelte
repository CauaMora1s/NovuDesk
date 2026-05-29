<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { can } from '$lib/permissions';
	import { membersApi, type Member, type PermissionOverride, type EffectivePermissions } from '$lib/api/members';
	import { rolesApi, type RoleWithPermissions, type Permission } from '$lib/api/roles';
	import { teamsApi, type Team } from '$lib/api/teams';
	import { slaApi, type CategorySLAStat, type SLAUnit, formatAvgDuration, formatSLAValue } from '$lib/api/sla';
	import PermissionMatrix from '$lib/components/ui/PermissionMatrix.svelte';

	let activeTab = 'organization';

	const tabs = [
		{ key: 'organization', label: 'settings.organization', permission: 'organization:view_settings' },
		{ key: 'members',      label: 'settings.members',      permission: 'users:view' },
		{ key: 'roles',        label: 'settings.roles',        permission: 'organization:view_settings' },
		{ key: 'sla',          label: 'settings.sla',          permission: 'sla:view' }
	] as const;

	$: visibleTabs = tabs.filter((t) => can(t.permission));

	// ── Search state ───────────────────────────────────────────
	let memberSearch = '';
	let slaSearch = '';

	$: filteredMembers = memberSearch.trim()
		? members.filter(
				(m) =>
					m.full_name.toLowerCase().includes(memberSearch.toLowerCase()) ||
					m.email.toLowerCase().includes(memberSearch.toLowerCase())
			)
		: members;

	$: filteredSlaStats = slaSearch.trim()
		? slaStats.filter((s) => s.category_name.toLowerCase().includes(slaSearch.toLowerCase()))
		: slaStats;

	// ── SLA state ──────────────────────────────────────────────
	let slaStats: CategorySLAStat[] = [];
	let slaLoading = false;
	let slaError = '';
	let editingCategoryId: string | null = null;
	let editValue = 1;
	let editUnit: SLAUnit = 'days';
	let slaSaving = false;

	async function loadSLAStats() {
		slaLoading = true;
		slaError = '';
		try {
			slaStats = await slaApi.listWithStats();
		} catch {
			slaError = $_('sla.loadError');
		} finally {
			slaLoading = false;
		}
	}

	function startEditSLA(stat: CategorySLAStat) {
		editingCategoryId = stat.category_id;
		editValue = stat.resolution_value ?? 1;
		editUnit = (stat.resolution_unit as SLAUnit) ?? 'days';
	}

	function cancelEditSLA() {
		editingCategoryId = null;
	}

	async function saveSLA(categoryId: string) {
		slaSaving = true;
		try {
			await slaApi.upsert(categoryId, { resolution_value: editValue, resolution_unit: editUnit });
			editingCategoryId = null;
			await loadSLAStats();
		} catch {
			slaError = $_('sla.saveError');
		} finally {
			slaSaving = false;
		}
	}

	async function deleteSLA(stat: CategorySLAStat) {
		if (!stat.sla_id) return;
		if (!confirm(`Remover SLA da categoria "${stat.category_name}"?`)) return;
		try {
			await slaApi.delete(stat.sla_id);
			await loadSLAStats();
		} catch {
			slaError = $_('sla.deleteError');
		}
	}

	$: if (activeTab === 'sla') { slaSearch = ''; loadSLAStats(); }

	// ── Members state ──────────────────────────────────────────
	let members: Member[] = [];
	let memberRoles: { id: string; name: string; is_system_role: boolean }[] = [];
	let teams: Team[] = [];
	let membersLoading = false;
	let membersError = '';

	let showCreateModal = false;
	let showEditModal = false;
	let editTarget: Member | null = null;
	let editTab = 'data';

	let form = { full_name: '', email: '', password: '', role_id: '', team_id: '' };
	let editFormProfile = { full_name: '', email: '' };
	let editFormPassword = { new_password: '' };
	let editFormRole = { role_id: '' };
	let editFormOverrides: Record<string, boolean | null> = {};

	let saving = false;
	let saveError = '';
	let saveSuccess = '';
	let showPassword = false;

	// Member permissions (loaded when editing)
	let effectivePerms: EffectivePermissions | null = null;
	let allPermissionsForMember: Permission[] = [];
	let permsLoading = false;

	async function loadMembers() {
		membersLoading = true;
		membersError = '';
		try {
			[members, memberRoles, teams] = await Promise.all([
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
		memberSearch = '';
		loadMembers();
	}

	function openCreate() {
		form = { full_name: '', email: '', password: '', role_id: memberRoles[0]?.id ?? '', team_id: '' };
		saveError = '';
		saveSuccess = '';
		showCreateModal = true;
	}

	async function openEdit(m: Member) {
		editTarget = m;
		editTab = 'data';
		editFormProfile = { full_name: m.full_name, email: m.email };
		editFormPassword = { new_password: '' };
		editFormRole = { role_id: m.role_id };
		editFormOverrides = {};
		saveError = '';
		saveSuccess = '';
		showEditModal = true;

		// Pre-load permissions data for the permissions tab
		if (can('users:manage_roles')) {
			permsLoading = true;
			try {
				[effectivePerms, allPermissionsForMember] = await Promise.all([
					membersApi.getPermissions(m.id),
					rolesApi.listAllPermissions()
				]);
				// Build override map from existing overrides
				editFormOverrides = {};
				if (effectivePerms?.overrides) {
					for (const o of effectivePerms.overrides) {
						editFormOverrides[o.permission_key] = o.is_granted;
					}
				}
			} catch {
				// Non-fatal; permissions tab will show a load error
			} finally {
				permsLoading = false;
			}
		}
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
		} catch {
			saveError = $_('settings.createMemberError');
		} finally {
			saving = false;
		}
	}

	async function saveProfile() {
		if (!editTarget) return;
		saving = true;
		saveError = '';
		saveSuccess = '';
		try {
			const updated = await membersApi.updateProfile(editTarget.id, editFormProfile);
			members = members.map((m) => (m.id === updated.id ? updated : m));
			editTarget = updated;
			saveSuccess = $_('memberEdit.saveSuccess');
		} catch {
			saveError = $_('settings.updateMemberError');
		} finally {
			saving = false;
		}
	}

	async function savePassword() {
		if (!editTarget) return;
		saving = true;
		saveError = '';
		saveSuccess = '';
		try {
			await membersApi.updatePassword(editTarget.id, editFormPassword);
			editFormPassword = { new_password: '' };
			saveSuccess = $_('memberEdit.passwordSuccess');
		} catch {
			saveError = $_('settings.updateMemberError');
		} finally {
			saving = false;
		}
	}

	async function saveRole() {
		if (!editTarget) return;
		saving = true;
		saveError = '';
		saveSuccess = '';
		try {
			const updated = await membersApi.updateRole(editTarget.id, { role_id: editFormRole.role_id });
			members = members.map((m) => (m.id === updated.id ? updated : m));
			editTarget = updated;
			saveSuccess = $_('memberEdit.saveSuccess');
		} catch {
			saveError = $_('settings.updateRoleError');
		} finally {
			saving = false;
		}
	}

	async function savePermissions() {
		if (!editTarget) return;
		saving = true;
		saveError = '';
		saveSuccess = '';
		try {
			const overridesList: PermissionOverride[] = Object.entries(editFormOverrides)
				.filter(([, val]) => val !== null)
				.map(([key, val]) => ({ permission_key: key, is_granted: val as boolean }));
			await membersApi.setPermissions(editTarget.id, overridesList);
			saveSuccess = $_('memberEdit.permissionsSuccess');
		} catch {
			saveError = $_('settings.updateMemberError');
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

	async function activate(m: Member) {
		if (!confirm($_('settings.confirmActivate', { values: { name: m.full_name } }))) return;
		try {
			const updated = await membersApi.activate(m.id);
			members = members.map((mb) => (mb.id === m.id ? updated : mb));
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

	// ── Roles state ─────────────────────────────────────────────
	let roles: RoleWithPermissions[] = [];
	let allPermissions: Permission[] = [];
	let rolesLoading = false;
	let rolesError = '';

	let showRoleModal = false;
	let roleModalMode: 'create' | 'view' | 'edit' = 'create';
	let roleTarget: RoleWithPermissions | null = null;
	let roleForm = { name: '', permission_keys: [] as string[] };
	let roleSaving = false;
	let roleSaveError = '';
	let expandedSystemRole: string | null = null;

	async function loadRoles() {
		rolesLoading = true;
		rolesError = '';
		try {
			[roles, allPermissions] = await Promise.all([
				rolesApi.list(),
				rolesApi.listAllPermissions()
			]);
		} catch {
			rolesError = $_('roles.loadError');
		} finally {
			rolesLoading = false;
		}
	}

	$: if (activeTab === 'roles') {
		loadRoles();
	}

	$: systemRoles = roles.filter((r) => r.is_system_role);
	$: orgRoles = roles.filter((r) => !r.is_system_role);

	function openCreateRole() {
		roleTarget = null;
		roleModalMode = 'create';
		roleForm = { name: '', permission_keys: [] };
		roleSaveError = '';
		showRoleModal = true;
	}

	function openViewRole(r: RoleWithPermissions) {
		roleTarget = r;
		roleModalMode = r.is_system_role ? 'view' : 'edit';
		roleForm = {
			name: r.name,
			permission_keys: r.permissions.map((p) => p.key)
		};
		roleSaveError = '';
		showRoleModal = true;
	}

	async function saveRole_() {
		roleSaving = true;
		roleSaveError = '';
		try {
			if (roleModalMode === 'create') {
				const created = await rolesApi.create({ name: roleForm.name, permission_keys: roleForm.permission_keys });
				roles = [...roles, created];
			} else if (roleModalMode === 'edit' && roleTarget) {
				const updated = await rolesApi.update(roleTarget.id, { name: roleForm.name, permission_keys: roleForm.permission_keys });
				roles = roles.map((r) => (r.id === updated.id ? updated : r));
			}
			showRoleModal = false;
		} catch {
			roleSaveError = roleModalMode === 'create' ? $_('roles.createError') : $_('roles.updateError');
		} finally {
			roleSaving = false;
		}
	}

	async function deleteRole(r: RoleWithPermissions) {
		if (!confirm($_('roles.deleteConfirm', { values: { name: r.name } }))) return;
		try {
			await rolesApi.delete(r.id);
			roles = roles.filter((ro) => ro.id !== r.id);
		} catch { /* ignore */ }
	}

	function toggleSystemRoleExpand(id: string) {
		expandedSystemRole = expandedSystemRole === id ? null : id;
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
					<!-- Members header -->
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
					{:else if filteredMembers.length === 0}
						<div class="text-center py-8 text-base-content/40">
							<p class="text-sm">Nenhum membro encontrado para "<strong>{memberSearch}</strong>".</p>
						</div>
					{:else}
						<!-- Member search -->
					<div class="mb-4">
						<label class="flex items-center gap-2 px-3 py-2 border border-base-300 rounded-lg bg-base-50 focus-within:border-primary transition-colors">
							<svg class="w-4 h-4 text-base-content/40 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-4.35-4.35M17 11A6 6 0 1 1 5 11a6 6 0 0 1 12 0z" />
							</svg>
							<input
								type="text"
								bind:value={memberSearch}
								placeholder="Pesquisar por nome ou e-mail..."
								class="grow text-sm bg-transparent outline-none placeholder:text-base-content/40"
							/>
							{#if memberSearch}
								<button type="button" class="text-base-content/40 hover:text-base-content" on:click={() => (memberSearch = '')}>
									<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
										<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
									</svg>
								</button>
							{/if}
						</label>
					</div>

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
									{#each filteredMembers as m}
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
														{#if can('users:deactivate') && !m.is_active}
															<button
																class="btn btn-ghost btn-xs text-success"
																on:click={() => activate(m)}
															>{$_('settings.activateButton')}</button>
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
					<!-- Roles header -->
					<div class="flex items-center justify-between mb-5">
						<h2 class="font-semibold">{$_('roles.title')}</h2>
						{#if can('organization:manage_settings')}
							<button class="btn btn-primary btn-sm gap-2" on:click={openCreateRole}>
								<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
								</svg>
								{$_('roles.newRole')}
							</button>
						{/if}
					</div>

					{#if rolesLoading}
						<div class="flex justify-center py-10">
							<span class="loading loading-spinner loading-md text-primary"></span>
						</div>
					{:else if rolesError}
						<div class="alert alert-error text-sm">{rolesError}</div>
					{:else}
						<!-- System roles section -->
						<div class="mb-6">
							<p class="text-xs font-semibold text-base-content/40 uppercase tracking-wider mb-3">
								{$_('roles.systemRoles')}
							</p>
							<div class="space-y-2">
								{#each systemRoles as r}
									<div class="border border-base-200 rounded-xl overflow-hidden">
										<div class="flex items-center justify-between px-4 py-3">
											<div class="flex items-center gap-3">
												<span class={roleBadgeClass(r.name)}>{r.name}</span>
												<span class="badge badge-s badge-outline text-base-content/40">
													{$_('roles.systemBadge')}
												</span>
												<span class="text-xs text-base-content/40">
													{r.permissions.length} {$_('roles.permissions').toLowerCase()}
												</span>
											</div>
											<button
												class="btn btn-ghost btn-xs gap-1"
												on:click={() => toggleSystemRoleExpand(r.id)}
											>
												{expandedSystemRole === r.id ? $_('roles.hidePermissions') : $_('roles.viewPermissions')}
												<svg class="w-3 h-3 transition-transform {expandedSystemRole === r.id ? 'rotate-180' : ''}" fill="none" viewBox="0 0 24 24" stroke="currentColor">
													<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
												</svg>
											</button>
										</div>
										{#if expandedSystemRole === r.id}
											<div class="border-t border-base-200 px-4 py-3 bg-base-50">
												<PermissionMatrix
													permissions={allPermissions}
													selected={r.permissions.map((p) => p.key)}
													readonly={true}
												/>
											</div>
										{/if}
									</div>
								{/each}
							</div>
						</div>

						<!-- Org custom roles section -->
						<div>
							<p class="text-xs font-semibold text-base-content/40 uppercase tracking-wider mb-3">
								{$_('roles.orgRoles')}
							</p>
							{#if orgRoles.length === 0}
								<div class="text-center py-8 text-base-content/40">
									<p class="text-sm">{$_('roles.noOrgRoles')}</p>
								</div>
							{:else}
								<div class="space-y-2">
									{#each orgRoles as r}
										<div class="flex items-center justify-between px-4 py-3 border border-base-200 rounded-xl hover:bg-base-200/30 transition-colors">
											<div class="flex items-center gap-3">
												<span class="badge badge-sm badge-ghost">{r.name}</span>
												<span class="text-xs text-base-content/40">
													{r.permissions.length} {$_('roles.permissions').toLowerCase()}
												</span>
											</div>
											<div class="flex gap-1">
												<button
													class="btn btn-ghost btn-xs"
													on:click={() => openViewRole(r)}
												>{$_('common.edit')}</button>
												{#if can('organization:manage_settings')}
													<button
														class="btn btn-ghost btn-xs text-error"
														on:click={() => deleteRole(r)}
													>{$_('roles.deleteRole')}</button>
												{/if}
											</div>
										</div>
									{/each}
								</div>
							{/if}
						</div>
					{/if}

				{:else if activeTab === 'sla'}
					<div class="flex items-center justify-between mb-4">
						<div>
							<h2 class="font-semibold">{$_('sla.title')}</h2>
							<p class="text-sm text-base-content/50 mt-0.5">{$_('sla.subtitle')}</p>
						</div>
					</div>

					{#if slaError}
						<div class="alert alert-error mb-4 text-sm">{slaError}</div>
					{/if}

					{#if slaLoading}
						<div class="flex items-center justify-center py-12">
							<span class="loading loading-spinner loading-md text-primary"></span>
						</div>
					{:else if slaStats.length === 0}
						<p class="text-sm text-base-content/50 py-8 text-center">Nenhuma categoria encontrada. Crie categorias em Times para configurar SLAs.</p>
					{:else}
						<!-- SLA search -->
						<div class="mb-4">
							<label class="flex items-center gap-2 px-3 py-2 border border-base-300 rounded-lg bg-base-50 focus-within:border-primary transition-colors">
								<svg class="w-4 h-4 text-base-content/40 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
									<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-4.35-4.35M17 11A6 6 0 1 1 5 11a6 6 0 0 1 12 0z" />
								</svg>
								<input
									type="text"
									bind:value={slaSearch}
									placeholder="Pesquisar categoria..."
									class="grow text-sm bg-transparent outline-none placeholder:text-base-content/40"
								/>
								{#if slaSearch}
									<button type="button" class="text-base-content/40 hover:text-base-content" on:click={() => (slaSearch = '')}>
										<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
										</svg>
									</button>
								{/if}
							</label>
						</div>

						{#if filteredSlaStats.length === 0}
							<p class="text-sm text-base-content/40 py-6 text-center">Nenhuma categoria encontrada para "<strong>{slaSearch}</strong>".</p>
						{:else}
						<div class="overflow-x-auto">
							<table class="table table-sm">
								<thead>
									<tr class="text-xs text-base-content/50 uppercase tracking-wide">
										<th>{$_('sla.categoryColumn')}</th>
										<th class="w-64">{$_('sla.resolutionColumn')}</th>
										<th class="w-40">{$_('sla.avgColumn')}</th>
										<th class="w-32 text-right">{$_('sla.actionsColumn')}</th>
									</tr>
								</thead>
								<tbody>
									{#each filteredSlaStats as stat}
										<tr class="hover:bg-base-50">
											<td class="font-medium text-sm">{stat.category_name}</td>
											<td>
												{#if editingCategoryId === stat.category_id}
													<div class="flex items-center gap-2">
														<input
															type="number"
															bind:value={editValue}
															min="1"
															max="9999"
															class="input input-bordered input-sm w-20"
														/>
														<select bind:value={editUnit} class="select select-bordered select-sm">
															<option value="hours">{$_('sla.unit.hours')}</option>
															<option value="days">{$_('sla.unit.days')}</option>
															<option value="weeks">{$_('sla.unit.weeks')}</option>
														</select>
														<button
															class="btn btn-primary btn-xs"
															disabled={slaSaving}
															on:click={() => saveSLA(stat.category_id)}
														>
															{#if slaSaving}
																<span class="loading loading-spinner loading-xs"></span>
															{:else}
																{$_('common.save')}
															{/if}
														</button>
														<button class="btn btn-ghost btn-xs" on:click={cancelEditSLA}>
															{$_('common.cancel')}
														</button>
													</div>
												{:else if stat.sla_id}
													<span class="inline-flex items-center gap-1.5 text-sm font-medium text-primary">
														<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
															<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
														</svg>
														{formatSLAValue(stat.resolution_value, stat.resolution_unit)}
													</span>
												{:else}
													<span class="text-sm text-base-content/40">{$_('sla.noPolicy')}</span>
												{/if}
											</td>
											<td class="text-sm text-base-content/60">
												{formatAvgDuration(stat.avg_resolution_hours)}
											</td>
											<td class="text-right">
												{#if editingCategoryId !== stat.category_id}
													<div class="flex items-center justify-end gap-1">
														{#if can('sla:manage')}
															<button
																class="btn btn-ghost btn-xs"
																on:click={() => startEditSLA(stat)}
															>
																{stat.sla_id ? $_('sla.editPolicy') : $_('sla.setPolicy')}
															</button>
															{#if stat.sla_id}
																<button
																	class="btn btn-ghost btn-xs text-error"
																	on:click={() => deleteSLA(stat)}
																>
																	{$_('sla.removePolicy')}
																</button>
															{/if}
														{/if}
													</div>
												{/if}
											</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
						{/if}
					{/if}
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
						{#each memberRoles as r}
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

<!-- ── Modal: Editar Membro (com tabs) ── -->
{#if showEditModal && editTarget}
	<dialog class="modal modal-open">
		<div class="modal-box max-w-lg">
			<button
				class="btn btn-sm btn-circle btn-ghost absolute right-3 top-3"
				on:click={() => (showEditModal = false)}
			>✕</button>

			<h3 class="font-bold text-lg mb-1">{$_('settings.modalEditMember')}</h3>
			<p class="text-sm text-base-content/50 mb-4">{editTarget.full_name}</p>

			<!-- Inner tabs -->
			<div class="tabs tabs-bordered mb-5">
				{#if can('users:manage_roles')}
					<button
						class="tab tab-sm"
						class:tab-active={editTab === 'data'}
						on:click={() => { editTab = 'data'; saveError = ''; saveSuccess = ''; }}
					>{$_('memberEdit.tabData')}</button>
					<button
						class="tab tab-sm"
						class:tab-active={editTab === 'security'}
						on:click={() => { editTab = 'security'; saveError = ''; saveSuccess = ''; }}
					>{$_('memberEdit.tabSecurity')}</button>
					<button
						class="tab tab-sm"
						class:tab-active={editTab === 'role'}
						on:click={() => { editTab = 'role'; saveError = ''; saveSuccess = ''; }}
					>{$_('memberEdit.tabRole')}</button>
					<button
						class="tab tab-sm"
						class:tab-active={editTab === 'permissions'}
						on:click={() => { editTab = 'permissions'; saveError = ''; saveSuccess = ''; }}
					>{$_('memberEdit.tabPermissions')}</button>
				{:else}
					<button class="tab tab-sm tab-active">{$_('memberEdit.tabRole')}</button>
				{/if}
			</div>

			{#if saveError}
				<div class="alert alert-error text-sm mb-4">{saveError}</div>
			{/if}
			{#if saveSuccess}
				<div class="alert alert-success text-sm mb-4">{saveSuccess}</div>
			{/if}

			<!-- Tab: Dados -->
			{#if editTab === 'data' && can('users:manage_roles')}
				<form on:submit|preventDefault={saveProfile} class="space-y-4">
					<div class="form-control">
						<label class="label pb-1" for="edit_full_name">
							<span class="label-text font-medium">{$_('members.name')}</span>
						</label>
						<input
							id="edit_full_name"
							type="text"
							bind:value={editFormProfile.full_name}
							class="input input-bordered"
							required
						/>
					</div>
					<div class="form-control">
						<label class="label pb-1" for="edit_email">
							<span class="label-text font-medium">{$_('members.email')}</span>
						</label>
						<input
							id="edit_email"
							type="email"
							bind:value={editFormProfile.email}
							class="input input-bordered"
							required
						/>
					</div>
					<div class="modal-action mt-6">
						<button type="button" class="btn btn-ghost" on:click={() => (showEditModal = false)}>
							{$_('common.cancel')}
						</button>
						<button type="submit" class="btn btn-primary" disabled={saving}>
							{#if saving}<span class="loading loading-spinner loading-xs"></span>{/if}
							{$_('common.save')}
						</button>
					</div>
				</form>

			<!-- Tab: Segurança -->
			{:else if editTab === 'security' && can('users:manage_roles')}
				<form on:submit|preventDefault={savePassword} class="space-y-4">
					<div class="form-control">
						<label class="label pb-1" for="edit_password">
							<span class="label-text font-medium">{$_('memberEdit.newPassword')}</span>
						</label>
						<input
							id="edit_password"
							type="password"
							bind:value={editFormPassword.new_password}
							class="input input-bordered"
							placeholder={$_('memberEdit.newPasswordPlaceholder')}
							required
							minlength="8"
						/>
					</div>
					<div class="modal-action mt-6">
						<button type="button" class="btn btn-ghost" on:click={() => (showEditModal = false)}>
							{$_('common.cancel')}
						</button>
						<button type="submit" class="btn btn-primary" disabled={saving}>
							{#if saving}<span class="loading loading-spinner loading-xs"></span>{/if}
							{$_('common.save')}
						</button>
					</div>
				</form>

			<!-- Tab: Função -->
			{:else if editTab === 'role'}
				<form on:submit|preventDefault={saveRole} class="space-y-4">
					<div class="form-control">
						<label class="label pb-1" for="edit_role">
							<span class="label-text font-medium">{$_('members.role')}</span>
						</label>
						<select id="edit_role" bind:value={editFormRole.role_id} class="select select-bordered" required>
							{#each memberRoles as r}
								<option value={r.id}>{r.name}</option>
							{/each}
						</select>
					</div>
					<div class="modal-action mt-6">
						<button type="button" class="btn btn-ghost" on:click={() => (showEditModal = false)}>
							{$_('common.cancel')}
						</button>
						{#if can('users:manage_roles')}
							<button type="submit" class="btn btn-primary" disabled={saving}>
								{#if saving}<span class="loading loading-spinner loading-xs"></span>{/if}
								{$_('common.save')}
							</button>
						{/if}
					</div>
				</form>

			<!-- Tab: Permissões -->
			{:else if editTab === 'permissions' && can('users:manage_roles')}
				<div>
					<p class="text-sm text-base-content/50 mb-4">{$_('memberEdit.permissionOverridesHint')}</p>

					{#if permsLoading}
						<div class="flex justify-center py-6">
							<span class="loading loading-spinner loading-sm text-primary"></span>
						</div>
					{:else if effectivePerms}
						<div class="max-h-80 overflow-y-auto pr-1">
							<PermissionMatrix
								permissions={allPermissionsForMember}
								overrideMode={true}
								bind:overrides={editFormOverrides}
								readonly={false}
							/>
						</div>

						<div class="mt-4 p-3 rounded-lg bg-base-200/50 text-xs text-base-content/50">
							<p class="font-medium mb-1">{$_('memberEdit.rolePermissions')}: {effectivePerms.role_permissions.length}</p>
							<p>{$_('memberEdit.effectivePermissions')}: {effectivePerms.effective.length}</p>
						</div>

						<div class="modal-action mt-4">
							<button type="button" class="btn btn-ghost" on:click={() => (showEditModal = false)}>
								{$_('common.cancel')}
							</button>
							<button class="btn btn-primary" disabled={saving} on:click={savePermissions}>
								{#if saving}<span class="loading loading-spinner loading-xs"></span>{/if}
								{$_('common.save')}
							</button>
						</div>
					{:else}
						<p class="text-sm text-error">{$_('memberEdit.loadPermissionsError')}</p>
					{/if}
				</div>
			{/if}
		</div>
		<div class="modal-backdrop" on:click={() => (showEditModal = false)}></div>
	</dialog>
{/if}

<!-- ── Modal: Criar / Editar Função ── -->
{#if showRoleModal}
	<dialog class="modal modal-open">
		<div class="modal-box max-w-2xl">
			<button
				class="btn btn-sm btn-circle btn-ghost absolute right-3 top-3"
				on:click={() => (showRoleModal = false)}
			>✕</button>

			<h3 class="font-bold text-lg mb-1">
				{roleModalMode === 'create' ? $_('roles.newRole') : roleModalMode === 'edit' ? $_('roles.editRole') : roleTarget?.name}
			</h3>
			{#if roleTarget?.is_system_role}
				<p class="text-xs text-base-content/40 mb-4">{$_('roles.systemRoleReadOnly')}</p>
			{:else if roleModalMode !== 'create'}
				<p class="text-xs text-base-content/40 mb-4">&nbsp;</p>
			{/if}

			{#if roleSaveError}
				<div class="alert alert-error text-sm mb-4">{roleSaveError}</div>
			{/if}

			<form on:submit|preventDefault={saveRole_} class="space-y-4">
				<!-- Role name field -->
				{#if !roleTarget?.is_system_role}
					<div class="form-control">
						<label class="label pb-1" for="role_name">
							<span class="label-text font-medium">{$_('roles.roleName')}</span>
						</label>
						<input
							id="role_name"
							type="text"
							bind:value={roleForm.name}
							class="input input-bordered"
							placeholder={$_('roles.roleNamePlaceholder')}
							disabled={roleTarget?.is_system_role}
							required
						/>
					</div>
				{/if}

				<!-- Permission matrix -->
				<div>
					<p class="label-text font-medium mb-2">{$_('roles.permissions')}</p>
					<div class="max-h-80 overflow-y-auto border border-base-200 rounded-xl p-3">
						<PermissionMatrix
							permissions={allPermissions}
							bind:selected={roleForm.permission_keys}
							readonly={!!roleTarget?.is_system_role}
						/>
					</div>
				</div>

				{#if !roleTarget?.is_system_role}
					<div class="modal-action mt-6">
						<button type="button" class="btn btn-ghost" on:click={() => (showRoleModal = false)}>
							{$_('common.cancel')}
						</button>
						<button type="submit" class="btn btn-primary" disabled={roleSaving}>
							{#if roleSaving}<span class="loading loading-spinner loading-xs"></span>{/if}
							{$_('common.save')}
						</button>
					</div>
				{:else}
					<div class="modal-action mt-6">
						<button type="button" class="btn btn-ghost" on:click={() => (showRoleModal = false)}>
							{$_('common.close')}
						</button>
					</div>
				{/if}
			</form>
		</div>
		<div class="modal-backdrop" on:click={() => (showRoleModal = false)}></div>
	</dialog>
{/if}
