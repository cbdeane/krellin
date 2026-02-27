import * as vscode from 'vscode';
import { StateStore } from './state';

function renderPre(lines: string[]) {
	return `<html><body><pre>${lines.join('')}</pre></body></html>`;
}

export function registerPanels(context: vscode.ExtensionContext, state: StateStore) {
	let terminalView: vscode.WebviewView | null = null;
	let diffView: vscode.WebviewView | null = null;
	let timelineView: vscode.WebviewView | null = null;

	state.onChange(() => {
		if (terminalView) terminalView.webview.html = renderPre(state.terminal);
		if (diffView) diffView.webview.html = `<html><body><pre>${state.diff.join('\n')}</pre></body></html>`;
		if (timelineView) timelineView.webview.html = `<html><body><pre>${state.timeline.join('\n')}</pre></body></html>`;
	});

	context.subscriptions.push(
		vscode.window.registerWebviewViewProvider('krellin.chat', {
			resolveWebviewView(view) {
				view.webview.html = `<html><body><h3>Krellin Chat</h3><p>Coming soon</p></body></html>`;
			}
		})
	);

	context.subscriptions.push(
		vscode.window.registerWebviewViewProvider('krellin.terminal', {
			resolveWebviewView(view) {
				terminalView = view;
				view.webview.html = renderPre(state.terminal);
			}
		})
	);

	context.subscriptions.push(
		vscode.window.registerWebviewViewProvider('krellin.diff', {
			resolveWebviewView(view) {
				diffView = view;
				view.webview.html = `<html><body><pre>${state.diff.join('\n')}</pre></body></html>`;
			}
		})
	);

	context.subscriptions.push(
		vscode.window.registerWebviewViewProvider('krellin.timeline', {
			resolveWebviewView(view) {
				timelineView = view;
				view.webview.html = `<html><body><pre>${state.timeline.join('\n')}</pre></body></html>`;
			}
		})
	);
}
