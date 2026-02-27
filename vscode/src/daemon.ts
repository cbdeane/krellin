import { spawn } from 'child_process';
import * as net from 'net';

export async function ensureDaemon(sock: string): Promise<void> {
	return new Promise((resolve) => {
		const probe = net.createConnection(sock);
		probe.once('connect', () => {
			probe.end();
			resolve();
		});
		probe.once('error', () => {
			spawn('krellind', ['-sock', sock], { detached: true, stdio: 'ignore' }).unref();
			setTimeout(resolve, 300);
		});
	});
}
