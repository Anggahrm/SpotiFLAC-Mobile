import { writable, derived } from 'svelte/store';
import { getProgress, type ProgressData } from '$lib/api';

export interface DownloadItem {
	id: string;
	title: string;
	artist: string;
	cover_url?: string;
	quality: string;
	status: 'pending' | 'downloading' | 'completed' | 'error';
	progress: number;
	message?: string;
	file_name?: string;
	started_at: number;
}

function createDownloadsStore() {
	const { subscribe, update, set } = writable<Map<string, DownloadItem>>(new Map());

	let pollingInterval: ReturnType<typeof setInterval> | null = null;

	function startPolling() {
		if (pollingInterval) return;

		pollingInterval = setInterval(async () => {
			try {
				const progress = await getProgress();
				update((downloads) => {
					for (const [id, item] of downloads) {
						if (progress[id]) {
							const p = progress[id];
							item.progress = p.progress;
							item.status = p.status;
							item.message = p.message;
							if (p.file_path) {
								item.file_name = p.file_path.split('/').pop();
							}
						}
					}
					return new Map(downloads);
				});
			} catch {
				// Ignore polling errors
			}
		}, 1000);
	}

	function stopPolling() {
		if (pollingInterval) {
			clearInterval(pollingInterval);
			pollingInterval = null;
		}
	}

	return {
		subscribe,
		add: (item: Omit<DownloadItem, 'status' | 'progress' | 'started_at'>) => {
			update((downloads) => {
				downloads.set(item.id, {
					...item,
					status: 'pending',
					progress: 0,
					started_at: Date.now()
				});
				return new Map(downloads);
			});
			startPolling();
		},
		updateItem: (id: string, updates: Partial<DownloadItem>) => {
			update((downloads) => {
				const item = downloads.get(id);
				if (item) {
					Object.assign(item, updates);
				}
				return new Map(downloads);
			});
		},
		remove: (id: string) => {
			update((downloads) => {
				downloads.delete(id);
				if (downloads.size === 0) {
					stopPolling();
				}
				return new Map(downloads);
			});
		},
		clear: () => {
			set(new Map());
			stopPolling();
		}
	};
}

export const downloads = createDownloadsStore();

export const activeDownloads = derived(downloads, ($downloads) =>
	Array.from($downloads.values()).filter(
		(d) => d.status === 'pending' || d.status === 'downloading'
	)
);

export const hasActiveDownloads = derived(
	activeDownloads,
	($active) => $active.length > 0
);
