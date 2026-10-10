<script lang="ts">
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';
	import type { HTMLButtonAttributes } from 'svelte/elements';

	let {
		variant = 'primary',
		size = 'md',
		class: className = '',
		children,
		...restProps
	}: {
		variant?: 'primary' | 'secondary' | 'outline' | 'ghost';
		size?: 'sm' | 'md' | 'lg';
		class?: string;
		children?: Snippet;
	} & HTMLButtonAttributes = $props();

	const variants = {
		primary: 'bg-indigo-600 text-white hover:bg-indigo-500 shadow-sm active:scale-98',
		secondary: 'bg-slate-800 text-slate-100 hover:bg-slate-700 active:scale-98',
		outline: 'border border-slate-700 text-slate-200 hover:bg-slate-800/60 active:scale-98',
		ghost: 'text-slate-400 hover:text-slate-100 hover:bg-slate-800/40'
	};

	const sizes = {
		sm: 'px-2.5 py-1.5 text-xs',
		md: 'px-4 py-2 text-sm',
		lg: 'px-5 py-2.5 text-base'
	};
</script>

<button
	class={cn(
		'inline-flex items-center justify-center gap-2 rounded-lg font-medium transition-all focus-visible:ring-2 focus-visible:ring-indigo-500 focus-visible:outline-none disabled:pointer-events-none disabled:opacity-50',
		variants[variant],
		sizes[size],
		className
	)}
	{...restProps}
>
	{#if children}
		{@render children()}
	{/if}
</button>
