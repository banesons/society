<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		class: className = '',
		children,
		...restProps
	}: {
		class?: string;
		children?: Snippet;
	} & HTMLAttributes<HTMLDivElement> = $props();
</script>

<div
	class={cn(
		'rounded-xl border border-slate-800 bg-slate-900/80 p-6 text-slate-100 shadow-xl backdrop-blur-sm',
		className
	)}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</div>
