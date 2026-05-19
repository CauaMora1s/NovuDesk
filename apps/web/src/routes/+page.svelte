<script lang="ts">
	import { goto } from '$app/navigation';
	import { browser } from '$app/environment';
	import { isAuthenticated } from '$lib/stores/auth';

	$: if (browser) {
		if ($isAuthenticated) {
			goto('/dashboard');
		} else {
			goto('/login');
		}
	}
</script>
