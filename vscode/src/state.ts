export class StateStore {
	terminal: string[] = [];
	diff: string[] = [];
	timeline: string[] = [];
	listeners: (() => void)[] = [];

	onChange(fn: () => void) {
		this.listeners.push(fn);
	}

	private notify() {
		this.listeners.forEach((fn) => fn());
	}

	appendTerminal(chunk: string) {
		this.terminal.push(chunk);
		if (this.terminal.length > 200) this.terminal.shift();
		this.notify();
	}

	setDiff(diffText: string) {
		this.diff = diffText.split('\n').slice(0, 200);
		this.notify();
	}

	appendTimeline(line: string) {
		this.timeline.push(line);
		if (this.timeline.length > 200) this.timeline.shift();
		this.notify();
	}
}
