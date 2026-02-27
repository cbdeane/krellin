import * as vscode from 'vscode';
import { SocketTransport } from './transport';
import { registerPanels } from './panels';
import { StateStore } from './state';
import { registerInput } from './input';
import { ensureDaemon } from './daemon';

export function activate(context: vscode.ExtensionContext) {
	const state = new StateStore();
	registerPanels(context, state);

	const transport = new SocketTransport();
	const sock = '/tmp/krellin.sock';
	const sessionId = '';
	const repoRoot = vscode.workspace.workspaceFolders?.[0]?.uri.fsPath ?? '';

	registerInput(context, transport);

	const disposable = vscode.commands.registerCommand('krellin.start', async () => {
		await ensureDaemon(sock);
		transport.connect(sock, sessionId, repoRoot, (line) => {
			state.appendTimeline(line);
			try {
				const ev = JSON.parse(line);
				if (ev.type === 'terminal.output' && ev.payload?.data) {
					state.appendTerminal(ev.payload.data as string);
				}
				if (ev.type === 'diff.ready' && ev.payload?.patch) {
					state.setDiff(ev.payload.patch as string);
				}
			} catch {
				// ignore parse errors
			}
		});
		vscode.window.showInformationMessage('Krellin connected');
	});
	context.subscriptions.push(disposable);
}

export function deactivate() {}
