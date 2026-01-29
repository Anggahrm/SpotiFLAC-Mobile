import { writable } from 'svelte/store';
import { browser } from '$app/environment';

export interface HistoryItem {
	id: string;
	title: string;
	artist: string;
	album?: string;
	cover_url?: string;
	quality: string;
	file_name: string;
	downloaded_at: number;
}

const STORAGE_KEY = 'spotiflac-history';
const MAX_ITEMS = 100;

function loadHistory(): HistoryItem[] {
	if (!browser) return [];
	try {
		const stored = localStorage.getItem(STORAGE_KEY);
		return stored ? JSON.parse(stored) : [];
	} catch {
		return [];
	}
}

function saveHistory(items: HistoryItem[]) {
	if (!browser) return;
	try {
		localStorage.setItem(STORAGE_KEY, JSON.stringify(items));
	} catch {
		// Storage full or unavailable
	}
}

function createHistoryStore() {
	const { subscribe, update, set } = writable<HistoryItem[]>(loadHistory());

	return {
		subscribe,
		add: (item: Omit<HistoryItem, 'id' | 'downloaded_at'>) => {
			update((items) => {
				const newItem: HistoryItem = {
					...item,
					id: crypto.randomUUID(),
					downloaded_at: Date.now()
				};
				const updated = [newItem, ...items].slice(0, MAX_ITEMS);
				saveHistory(updated);
				return updated;
			});
		},
		remove: (id: string) => {
			update((items) => {
				const updated = items.filter((i) => i.id !== id);
				saveHistory(updated);
				return updated;
			});
		},
		clear: () => {
			set([]);
			saveHistory([]);
		}
	};
}

export const history = createHistoryStore();
