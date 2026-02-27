import * as net from 'net';

export type EventHandler = (line: string) => void;

export class SocketTransport {
	private socket: net.Socket | null = null;

	connect(path: string, sessionId: string, repoRoot: string, onEvent: EventHandler) {
		this.socket = net.createConnection(path);
		this.socket.on('data', (data) => {
			const lines = data.toString('utf8').split('\n').filter(Boolean);
			lines.forEach((line) => {
				try {
					const msg = JSON.parse(line);
					if (msg.type === 'connected') return;
				} catch {
					// ignore parse errors
				}
				onEvent(line);
			});
		});
		this.socket.write(JSON.stringify({ type: 'connect', session_id: sessionId, repo_root: repoRoot }) + '\n');
	}

	send(action: object) {
		if (!this.socket) return;
		this.socket.write(JSON.stringify(action) + '\n');
	}

	close() {
		this.socket?.end();
		this.socket = null;
	}
}
