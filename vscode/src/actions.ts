export function buildRunCommand(command: string) {
	return {
		action_id: 'vscode',
		session_id: '',
		agent_id: 'vscode',
		type: 'run_command',
		timestamp: new Date().toISOString(),
		payload: { command, cwd: '/workspace', env: {} }
	};
}
