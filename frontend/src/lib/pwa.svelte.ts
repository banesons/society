// PWA Store and Service Worker lifecycle manager for CHS (Svelte 5 Runes)

interface BeforeInstallPromptEvent extends Event {
	readonly platforms: string[];
	readonly userChoice: Promise<{
		outcome: 'accepted' | 'dismissed';
		platform: string;
	}>;
	prompt(): Promise<void>;
}

class PwaStore {
	isInstalled = $state<boolean>(false);
	canInstall = $state<boolean>(false);
	isRegistered = $state<boolean>(false);
	hasUpdate = $state<boolean>(false);

	private deferredPrompt: BeforeInstallPromptEvent | null = null;

	init() {
		if (typeof window === 'undefined') return;

		// 1. Check if already running in standalone display mode
		const isStandalone =
			window.matchMedia('(display-mode: standalone)').matches ||
			// Safari iOS legacy check
			('standalone' in window.navigator && Boolean((window.navigator as unknown as { standalone: boolean }).standalone));

		this.isInstalled = isStandalone;

		// Watch display-mode changes
		window.matchMedia('(display-mode: standalone)').addEventListener('change', (e) => {
			this.isInstalled = e.matches;
			if (e.matches) {
				this.canInstall = false;
				this.deferredPrompt = null;
			}
		});

		// 2. Capture beforeinstallprompt event
		window.addEventListener('beforeinstallprompt', (e) => {
			e.preventDefault();
			this.deferredPrompt = e as BeforeInstallPromptEvent;
			this.canInstall = true;
		});

		window.addEventListener('appinstalled', () => {
			this.isInstalled = true;
			this.canInstall = false;
			this.deferredPrompt = null;
		});

		// 3. Register unified service worker
		this.registerServiceWorker();
	}

	async registerServiceWorker() {
		if (typeof window === 'undefined' || !('serviceWorker' in navigator)) return;

		try {
			const registration = await navigator.serviceWorker.register('/firebase-messaging-sw.js', {
				scope: '/'
			});
			this.isRegistered = true;

			// Check for background updates
			registration.addEventListener('updatefound', () => {
				const installingWorker = registration.installing;
				if (!installingWorker) return;

				installingWorker.addEventListener('statechange', () => {
					if (
						installingWorker.state === 'installed' &&
						navigator.serviceWorker.controller
					) {
						this.hasUpdate = true;
					}
				});
			});
		} catch (err) {
			console.warn('[PWA] Service worker registration deferred or failed:', err);
		}
	}

	async promptInstall(): Promise<boolean> {
		if (!this.deferredPrompt) {
			return false;
		}

		try {
			await this.deferredPrompt.prompt();
			const choice = await this.deferredPrompt.userChoice;
			if (choice.outcome === 'accepted') {
				this.isInstalled = true;
				this.canInstall = false;
				this.deferredPrompt = null;
				return true;
			}
		} catch (err) {
			console.warn('[PWA] Install prompt error:', err);
		}
		return false;
	}
}

export const pwa = new PwaStore();
