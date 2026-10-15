/**
 * Reactive viewport and screen classification helper for SvelteKit 5.
 * Provides unified isMobile, isTablet, isDesktop states and DeviceType enum.
 */

export enum DeviceType {
	Mobile = 'mobile',
	Tablet = 'tablet',
	Desktop = 'desktop'
}

class ScreenStore {
	width = $state(typeof window !== 'undefined' ? window.innerWidth : 1200);

	get isMobile(): boolean {
		return this.width < 768;
	}

	get isTablet(): boolean {
		return this.width >= 768 && this.width < 1024;
	}

	get isDesktop(): boolean {
		return this.width >= 1024;
	}

	get device(): DeviceType {
		if (this.isMobile) return DeviceType.Mobile;
		if (this.isTablet) return DeviceType.Tablet;
		return DeviceType.Desktop;
	}

	constructor() {
		if (typeof window !== 'undefined') {
			window.addEventListener('resize', () => {
				this.width = window.innerWidth;
			});
		}
	}
}

export const screen = new ScreenStore();
