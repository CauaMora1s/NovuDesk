<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { goto } from '$app/navigation';
	import { ticketsApi, type TicketPriority } from '$lib/api/tickets';
	import { teamsApi, type Team } from '$lib/api/teams';
	import type { Category } from '$lib/api/categories';
	import FileUpload from '$lib/components/FileUpload.svelte';

	let title = '';
	let description = '';
	let priority: TicketPriority = 'normal';
	let categoryId = '';
	let loading = false;
	let error = '';

	let fileUploadRef: FileUpload;

	// Teams with their categories for the grouped dropdown
	type TeamWithCats = { team: Team; categories: Category[] };
	let teamGroups: TeamWithCats[] = [];

	$: loadCategories();

	async function loadCategories() {
		try {
			const teams = await teamsApi.list();
			const groups = await Promise.all(
				teams.map(async (t) => ({
					team: t,
					categories: await teamsApi.listCategories(t.id)
				}))
			);
			// Only show teams that actually have categories
			teamGroups = groups.filter((g) => g.categories.length > 0);
		} catch {
			// non-critical — dropdown just stays empty
		}
	}

	const priorities: Array<{ value: TicketPriority; label: string }> = [
		{ value: 'low',    label: 'tickets.priority.low' },
		{ value: 'normal', label: 'tickets.priority.normal' },
		{ value: 'high',   label: 'tickets.priority.high' },
		{ value: 'urgent', label: 'tickets.priority.urgent' }
	];

	async function submit() {
		if (!title.trim()) return;
		loading = true;
		error = '';
		try {
			const ticket = await ticketsApi.create({
				title,
				description,
				priority,
				category_id: categoryId || undefined
			});

			if (fileUploadRef) {
				await fileUploadRef.uploadAll(ticket.id);
			}

			goto(`/tickets/${ticket.id}`);
		} catch {
			error = $_('tickets.createError');
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head><title>Novo Ticket — NovuDesk</title></svelte:head>

<div class="p-8 max-w-2xl mx-auto">
	<div class="flex items-center gap-3 mb-6">
		<a href="/tickets" class="btn btn-ghost btn-sm btn-square">
			<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
			</svg>
		</a>
		<div>
			<h1 class="text-xl font-bold">{$_('tickets.new')}</h1>
		</div>
	</div>

	<div class="card bg-base-100 shadow-card">
		<div class="card-body p-6">
			{#if error}
				<div class="alert alert-error mb-4 text-sm">{error}</div>
			{/if}

			<form on:submit|preventDefault={submit} class="space-y-5">
				<div class="form-control">
					<label class="label pb-1" for="title">
						<span class="label-text font-medium">{$_('tickets.form.title')} *</span>
					</label>
					<input
						id="title"
						type="text"
						bind:value={title}
						placeholder={$_('tickets.form.titlePlaceholder')}
						class="input input-bordered w-full"
						required
					/>
				</div>

				<div class="form-control">
					<label class="label pb-1" for="description">
						<span class="label-text font-medium">{$_('tickets.form.description')}</span>
					</label>
					<textarea
						id="description"
						bind:value={description}
						placeholder={$_('tickets.form.descriptionPlaceholder')}
						class="textarea textarea-bordered w-full min-h-32 resize-y"
						rows="5"
					></textarea>
				</div>

				<div class="form-control">
					<label class="label pb-1" for="priority">
						<span class="label-text font-medium">{$_('tickets.priorityLabel')}</span>
					</label>
					<select id="priority" bind:value={priority} class="select select-bordered w-full">
						{#each priorities as p}
							<option value={p.value}>{$_(p.label)}</option>
						{/each}
					</select>
				</div>

				{#if teamGroups.length > 0}
					<div class="form-control">
						<label class="label pb-1" for="category">
							<span class="label-text font-medium">{$_('tickets.categoryLabel')} *</span>
						</label>
						<select id="category" bind:value={categoryId} class="select select-bordered w-full" required>
							<option value="">{$_('tickets.noCategoryOption')}</option>
							{#each teamGroups as g}
								<optgroup label={g.team.name}>
									{#each g.categories as cat}
										<option value={cat.id}>{cat.name}</option>
									{/each}
								</optgroup>
							{/each}
						</select>
					</div>
				{/if}

				<div class="form-control">
					<label class="label pb-1">
						<span class="label-text font-medium">
							{$_('tickets.attachments.title')}
							<span class="text-base-content/40 font-normal">{$_('common.optional')}</span>
						</span>
					</label>
					<!-- ticketId is empty at creation time; uploadAll() is called after ticket creation -->
					<FileUpload bind:this={fileUploadRef} ticketId="" />
				</div>

				<div class="flex items-center gap-3 pt-2">
					<button type="submit" class="btn btn-primary" disabled={loading || !title.trim() || (teamGroups.length > 0 && !categoryId)}>
						{#if loading}
							<span class="loading loading-spinner loading-sm"></span>
						{/if}
						{$_('common.create')}
					</button>
					<a href="/tickets" class="btn btn-ghost">{$_('common.cancel')}</a>
				</div>
			</form>
		</div>
	</div>
</div>
