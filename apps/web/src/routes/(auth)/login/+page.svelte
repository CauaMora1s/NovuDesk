<script lang="ts">
	import { _ } from 'svelte-i18n';
	import { goto } from '$app/navigation';
	import { api } from '$lib/api/client';
	import { authStore } from '$lib/stores/auth';

	let email = '';
	let password = '';
	let orgSlug = '';
	let loading = false;
	let error = '';

	async function handleLogin() {
		loading = true;
		error = '';

		try {
			const res = await api.post<{ access_token: string; expires_in: number }>('/api/v1/auth/login', {
				email,
				password,
				org_slug: orgSlug
			});

			// Decode claims from JWT to populate auth store
			const [, payloadB64] = res.access_token.split('.');
			const payload = JSON.parse(atob(payloadB64.replace(/-/g, '+').replace(/_/g, '/')));

			authStore.setSession({
				user: { id: payload.uid, email, fullName: '', avatarUrl: null, locale: 'pt' },
				orgId: payload.oid,
				orgSlug,
				role: payload.role,
				permissions: payload.perms ?? [],
				teamIds: payload.team_ids ?? [],
				accessToken: res.access_token
			});

			goto('/dashboard');
		} catch (err: unknown) {
			error = $_('auth.loginError');
		} finally {
			loading = false;
		}
	}
</script>

<svelte:head>
	<title>Entrar — NovuDesk</title>
</svelte:head>

<div class="card bg-base-100 shadow-card">
	<div class="card-body p-8">
		<h1 class="text-2xl font-bold mb-1">{$_('auth.login')}</h1>
		<p class="text-base-content/60 text-sm mb-6">Acesse seu painel de atendimento</p>

		{#if error}
			<div class="alert alert-error mb-4 text-sm">
				<svg class="w-4 h-4 shrink-0" fill="none" viewBox="0 0 24 24" stroke="currentColor">
					<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2"
						d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
				</svg>
				{error}
			</div>
		{/if}

		<form on:submit|preventDefault={handleLogin} class="space-y-4">
			<div class="form-control">
				<label class="label pb-1" for="org">
					<span class="label-text font-medium">{$_('auth.orgSlug')}</span>
				</label>
				<input
					id="org"
					type="text"
					bind:value={orgSlug}
					placeholder="minha-empresa"
					class="input input-bordered w-full"
					required
					autocomplete="organization"
				/>
			</div>

			<div class="form-control">
				<label class="label pb-1" for="email">
					<span class="label-text font-medium">{$_('auth.email')}</span>
				</label>
				<input
					id="email"
					type="email"
					bind:value={email}
					placeholder="voce@empresa.com"
					class="input input-bordered w-full"
					required
					autocomplete="email"
				/>
			</div>

			<div class="form-control">
				<label class="label pb-1" for="password">
					<span class="label-text font-medium">{$_('auth.password')}</span>
				</label>
				<input
					id="password"
					type="password"
					bind:value={password}
					class="input input-bordered w-full"
					required
					autocomplete="current-password"
				/>
				<label class="label pt-1">
					<a href="/forgot-password" class="label-text-alt link link-hover text-primary">
						{$_('auth.forgotPassword')}
					</a>
				</label>
			</div>

			<button type="submit" class="btn btn-primary w-full" disabled={loading}>
				{#if loading}
					<span class="loading loading-spinner loading-sm"></span>
				{/if}
				{$_('auth.login')}
			</button>
		</form>

		<p class="text-center text-sm text-base-content/60 mt-6">
			{$_('auth.noAccount')}
			<a href="/register" class="link link-primary font-medium">{$_('auth.register')}</a>
		</p>
	</div>
</div>
