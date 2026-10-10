<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	import type { HTMLAttributes } from 'svelte/elements';

	let {
		variant = 'default',
		class: className = '',
		children,
		...restProps
	}: {
		variant?: 'default' | 'success' | 'warning' | 'destructive' | 'outline';
		class?: string;
		children?: Snippet;
	} & HTMLAttributes<HTMLSpanElement> = $props();

	const variants = {
		default: 'bg-indigo-500/10 text-indigo-400 border-indigo-500/30',
		success: 'bg-emerald-500/10 text-emerald-400 border-emerald-500/30',
		warning: 'bg-amber-500/10 text-amber-400 border-amber-500/30',
		destructive: 'bg-rose-500/10 text-rose-400 border-rose-500/30',
		outline: 'border-slate-700 text-slate-300'
	};
</script>

<span
	class={cn(
		'inline-flex items-center gap-1.5 rounded-full border px-2.5 py-0.5 text-xs font-medium tracking-wide transition-colors',
		variants[variant] || variants.default,
		className
	)}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</span>
