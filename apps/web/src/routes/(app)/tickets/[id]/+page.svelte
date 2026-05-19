<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { page } from '$app/stores';
	import { ticketsApi, type Ticket, type TicketStatus } from '$lib/api/tickets';
	import { can } from '$lib/permissions';

	let ticket: Ticket | null = null;
	let loading = true;
	let error = '';
	let commentBody = '';
	let isInternal = false;
	let submittingComment = false;

	$: ticketId = $page.params.id;

	// Re-fetches whenever ticketId changes (including initial load)
	$: if (ticketId) {
		loading = true;
		error = '';
		ticket = null;
		ticketsApi
			.get(ticketId)
			.then((t) => {
				ticket = t;
			})
			.catch(() => {
				error = 'Ticket não encontrado.';
			})
			.finally(() => {
				loading = false;
			});
	}

	async function changeStatus(status: TicketStatus) {
		if (!ticket) return;
		try {
			ticket = await ticketsApi.update(ticket.id, { status });
		} catch { /* handle silently */ }
	}

	function formatDate(iso: string) {
		return new Intl.DateTimeFormat(undefined, {
			year: 'numeric', month: 'short', day: 'numeric',
			hour: '2-digit', minute: '2-digit'
		}).format(new Date(iso));
	}

	const statusColors: Record<TicketStatus, string> = {
		open:     'status-badge status-open',
		pending:  'status-badge status-pending',
		on_hold:  'status-badge status-on-hold',
		resolved: 'status-badge status-resolved',
		closed:   'status-badge status-closed'
	};
</script>

<svelte:head>
	<title>{ticket ? `#${ticket.number} ${ticket.title}` : 'Ticket'} — NovuDesk</title>
</svelte:head>

<div class="p-8 max-w-5xl mx-auto">
	<!-- Back -->
	<div class="mb-5">
		<a href="/tickets" class="inline-flex items-center gap-1.5 text-sm text-base-content/50 hover:text-base-content transition-colors">
			<svg class="w-4 h-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
				<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 19l-7-7 7-7" />
			</svg>
			Todos os tickets
		</a>
	</div>

	{#if loading}
		<div class="flex items-center justify-center py-20">
			<span class="loading loading-spinner loading-lg text-primary"></span>
		</div>
	{:else if error || !ticket}
		<div class="alert alert-error">{error || 'Ticket não encontrado.'}</div>
	{:else}
		<div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
			<!-- Main column -->
			<div class="lg:col-span-2 space-y-4">
				<!-- Title card -->
				<div class="card bg-base-100 shadow-card">
					<div class="card-body p-6">
						<div class="flex items-start justify-between gap-4">
							<div class="flex-1 min-w-0">
								<div class="flex items-center gap-2 mb-2">
									<span class="text-xs text-base-content/40 font-mono">#{ticket.number}</span>
									<span class="{statusColors[ticket.status]}">
										{$_(`tickets.status.${ticket.status}`)}
									</span>
									{#if ticket.sla_breached}
										<span class="badge badge-sm badge-error">SLA violado</span>
									{/if}
								</div>
								<h1 class="text-xl font-bold">{ticket.title}</h1>
							</div>

							{#if can('tickets:change_status')}
								<div class="dropdown dropdown-end shrink-0">
									<button class="btn btn-sm btn-outline gap-1">
										Alterar status
										<svg class="w-3.5 h-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" />
										</svg>
									</button>
									<ul class="dropdown-content menu menu-sm bg-base-100 rounded-box shadow-lg border border-base-200 z-10 w-40 mt-1">
										{#each ['open','pending','on_hold','resolved','closed'] as s}
											<li>
												<button
													on:click={() => changeStatus(s as TicketStatus)}
													class:font-medium={ticket.status === s}
												>
													{$_(`tickets.status.${s}`)}
												</button>
											</li>
										{/each}
									</ul>
								</div>
							{/if}
						</div>

						{#if ticket.description}
							<div class="prose prose-sm max-w-none mt-4 text-base-content/80">
								<p>{ticket.description}</p>
							</div>
						{/if}

						<div class="text-xs text-base-content/40 mt-4">
							Criado em {formatDate(ticket.created_at)}
						</div>
					</div>
				</div>

				<!-- Reply box -->
				{#if can('comments:create_public') || can('comments:create_internal')}
					<div class="card bg-base-100 shadow-card">
						<div class="card-body p-5">
							<div class="tabs tabs-bordered mb-3">
								{#if can('comments:create_public')}
									<button
										class="tab tab-sm"
										class:tab-active={!isInternal}
										on:click={() => (isInternal = false)}
									>
										{$_('tickets.comments.public')}
									</button>
								{/if}
								{#if can('comments:create_internal')}
									<button
										class="tab tab-sm"
										class:tab-active={isInternal}
										on:click={() => (isInternal = true)}
									>
										{$_('tickets.comments.internal')}
									</button>
								{/if}
							</div>

							{#if isInternal}
								<div class="alert alert-warning py-2 text-xs mb-2">
									Nota interna — visível apenas para agentes
								</div>
							{/if}

							<textarea
								bind:value={commentBody}
								placeholder={isInternal
									? $_('tickets.comments.internalPlaceholder')
									: $_('tickets.comments.placeholder')}
								class="textarea textarea-bordered w-full min-h-24 resize-none text-sm"
								class:border-warning={isInternal}
							></textarea>

							<div class="flex justify-end mt-3">
								<button
									class="btn btn-primary btn-sm"
									disabled={submittingComment || !commentBody.trim()}
								>
									{#if submittingComment}
										<span class="loading loading-spinner loading-xs"></span>
									{/if}
									{$_('tickets.comments.send')}
								</button>
							</div>
						</div>
					</div>
				{/if}
			</div>

			<!-- Sidebar column -->
			<div class="space-y-4">
				<div class="card bg-base-100 shadow-card">
					<div class="card-body p-5">
						<h3 class="text-xs font-semibold text-base-content/50 uppercase tracking-wide mb-3">Detalhes</h3>

						<dl class="space-y-3 text-sm">
							<div>
								<dt class="text-xs text-base-content/40 mb-0.5">{$_('tickets.priority')}</dt>
								<dd class="status-badge priority-{ticket.priority}">
									{$_(`tickets.priority.${ticket.priority}`)}
								</dd>
							</div>

							{#if ticket.tags && ticket.tags.length > 0}
								<div>
									<dt class="text-xs text-base-content/40 mb-1">Tags</dt>
									<dd class="flex flex-wrap gap-1">
										{#each ticket.tags as tag}
											<span class="badge badge-ghost badge-sm">{tag}</span>
										{/each}
									</dd>
								</div>
							{/if}

							{#if ticket.sla_resolution_due_at}
								<div>
									<dt class="text-xs text-base-content/40 mb-0.5">Prazo SLA</dt>
									<dd class="text-sm" class:text-error={ticket.sla_breached}>
										{formatDate(ticket.sla_resolution_due_at)}
									</dd>
								</div>
							{/if}
						</dl>
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>
