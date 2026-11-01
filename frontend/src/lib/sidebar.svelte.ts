/**
 * Reactive desktop sidebar state manager for SvelteKit 5.
 * Persists user collapse/expand preference across navigation sessions.
 */

class SidebarStore {
	isCollapsed = $state(
		typeof window !== 'undefined'
			? localStorage.getItem('chs_sidebar_collapsed') === 'true'
			: false
	);

	toggle() {
		this.isCollapsed = !this.isCollapsed;
		if (typeof window !== 'undefined') {
			try {
				localStorage.setItem('chs_sidebar_collapsed', String(this.isCollapsed));
			} catch {
				// Local storage disabled or unavailable
			}
		}
	}

	collapse() {
		this.isCollapsed = true;
		if (typeof window !== 'undefined') {
			try {
				localStorage.setItem('chs_sidebar_collapsed', 'true');
			} catch {
				// Local storage disabled or unavailable
			}
		}
	}

	expand() {
		this.isCollapsed = false;
		if (typeof window !== 'undefined') {
			try {
				localStorage.setItem('chs_sidebar_collapsed', 'false');
			} catch {
				// Local storage disabled or unavailable
			}
		}
	}
}

export const sidebar = new SidebarStore();
