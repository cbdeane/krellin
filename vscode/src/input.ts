import * as vscode from 'vscode';
import { SocketTransport } from './transport';
import { buildRunCommand } from './actions';

export function registerInput(context: vscode.ExtensionContext, transport: SocketTransport) {
	context.subscriptions.push(
		vscode.commands.registerCommand('krellin.runCommand', async () => {
			const cmd = await vscode.window.showInputBox({ prompt: 'Run command in capsule' });
			if (!cmd) return;
			transport.send(buildRunCommand(cmd));
		})
	);
}
