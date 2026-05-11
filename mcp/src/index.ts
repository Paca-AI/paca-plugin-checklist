import {
	PluginAPIClient,
	type PluginMCPContext,
	type PluginMCPEntry,
	type Tool,
	errorResult,
	textResult,
} from "@paca-ai/plugin-sdk-mcp";

// ── Domain types ──────────────────────────────────────────────────────────────

interface ChecklistItem {
	id: string;
	checklist_id: string;
	title: string;
	is_checked: boolean;
	assignee_id: string | null;
	position: number;
	created_at: string;
	updated_at: string;
}

interface Checklist {
	id: string;
	task_id: string;
	title: string;
	position: number;
	items: ChecklistItem[];
	created_at: string;
	updated_at: string;
}

// ── Formatting helpers ────────────────────────────────────────────────────────

function formatItem(item: ChecklistItem): string {
	const checked = item.is_checked ? "[x]" : "[ ]";
	return `  ${checked} ${item.title} (ID: ${item.id})`;
}

function formatChecklist(cl: Checklist): string {
	const items =
		cl.items.length > 0
			? cl.items.map(formatItem).join("\n")
			: "  (no items)";
	return `Checklist: ${cl.title}\nID: ${cl.id}\nTask: ${cl.task_id}\n\nItems:\n${items}`;
}

// ── Tool definitions ──────────────────────────────────────────────────────────

const UUID_DESC =
	"The technical UUID of the %s (e.g., '550e8400-e29b-41d4-a716-446655440000').";

const tools: Tool[] = [
	{
		name: "checklist_list_checklists",
		description:
			"List all checklists (with their items) for a specific task in a project.",
		inputSchema: {
			type: "object",
			properties: {
				projectId: {
					type: "string",
					description: UUID_DESC.replace("%s", "project") + " Use list_projects to get the project ID.",
				},
				taskId: {
					type: "string",
					description: UUID_DESC.replace("%s", "task") + " Use list_tasks to get the task ID.",
				},
			},
			required: ["projectId", "taskId"],
		},
	},
	{
		name: "checklist_create_checklist",
		description: "Create a new checklist for a task.",
		inputSchema: {
			type: "object",
			properties: {
				projectId: {
					type: "string",
					description: UUID_DESC.replace("%s", "project") + " Use list_projects to get the project ID.",
				},
				taskId: {
					type: "string",
					description: UUID_DESC.replace("%s", "task") + " Use list_tasks to get the task ID.",
				},
				title: {
					type: "string",
					description: "The title of the new checklist.",
				},
			},
			required: ["projectId", "taskId", "title"],
		},
	},
	{
		name: "checklist_update_checklist",
		description: "Update the title of an existing checklist.",
		inputSchema: {
			type: "object",
			properties: {
				projectId: {
					type: "string",
					description: UUID_DESC.replace("%s", "project") + " Use list_projects to get the project ID.",
				},
				taskId: {
					type: "string",
					description: UUID_DESC.replace("%s", "task") + " Use list_tasks to get the task ID.",
				},
				checklistId: {
					type: "string",
					description: UUID_DESC.replace("%s", "checklist") + " Use checklist_list_checklists to get the checklist ID.",
				},
				title: {
					type: "string",
					description: "The new title for the checklist.",
				},
			},
			required: ["projectId", "taskId", "checklistId", "title"],
		},
	},
	{
		name: "checklist_delete_checklist",
		description: "Delete a checklist and all its items from a task.",
		inputSchema: {
			type: "object",
			properties: {
				projectId: {
					type: "string",
					description: UUID_DESC.replace("%s", "project") + " Use list_projects to get the project ID.",
				},
				taskId: {
					type: "string",
					description: UUID_DESC.replace("%s", "task") + " Use list_tasks to get the task ID.",
				},
				checklistId: {
					type: "string",
					description: UUID_DESC.replace("%s", "checklist") + " Use checklist_list_checklists to get the checklist ID.",
				},
			},
			required: ["projectId", "taskId", "checklistId"],
		},
	},
	{
		name: "checklist_create_item",
		description: "Add a new item to an existing checklist.",
		inputSchema: {
			type: "object",
			properties: {
				projectId: {
					type: "string",
					description: UUID_DESC.replace("%s", "project") + " Use list_projects to get the project ID.",
				},
				taskId: {
					type: "string",
					description: UUID_DESC.replace("%s", "task") + " Use list_tasks to get the task ID.",
				},
				checklistId: {
					type: "string",
					description: UUID_DESC.replace("%s", "checklist") + " Use checklist_list_checklists to get the checklist ID.",
				},
				title: {
					type: "string",
					description: "The title/text of the new checklist item.",
				},
			},
			required: ["projectId", "taskId", "checklistId", "title"],
		},
	},
	{
		name: "checklist_update_item",
		description: "Update a checklist item's title, checked state, or assignee.",
		inputSchema: {
			type: "object",
			properties: {
				projectId: {
					type: "string",
					description: UUID_DESC.replace("%s", "project") + " Use list_projects to get the project ID.",
				},
				taskId: {
					type: "string",
					description: UUID_DESC.replace("%s", "task") + " Use list_tasks to get the task ID.",
				},
				checklistId: {
					type: "string",
					description: UUID_DESC.replace("%s", "checklist") + " Use checklist_list_checklists to get the checklist ID.",
				},
				itemId: {
					type: "string",
					description: UUID_DESC.replace("%s", "item") + " Use checklist_list_checklists to see item IDs.",
				},
				title: {
					type: "string",
					description: "New title for the item (optional).",
				},
				isChecked: {
					type: "boolean",
					description: "Mark the item as checked (true) or unchecked (false) (optional).",
				},
				assigneeId: {
					type: "string",
					description: "UUID of the user to assign the item to. Pass null to unassign (optional).",
				},
			},
			required: ["projectId", "taskId", "checklistId", "itemId"],
		},
	},
	{
		name: "checklist_delete_item",
		description: "Delete a checklist item.",
		inputSchema: {
			type: "object",
			properties: {
				projectId: {
					type: "string",
					description: UUID_DESC.replace("%s", "project") + " Use list_projects to get the project ID.",
				},
				taskId: {
					type: "string",
					description: UUID_DESC.replace("%s", "task") + " Use list_tasks to get the task ID.",
				},
				checklistId: {
					type: "string",
					description: UUID_DESC.replace("%s", "checklist") + " Use checklist_list_checklists to get the checklist ID.",
				},
				itemId: {
					type: "string",
					description: UUID_DESC.replace("%s", "item") + " Use checklist_list_checklists to see item IDs.",
				},
			},
			required: ["projectId", "taskId", "checklistId", "itemId"],
		},
	},
];

// ── Entry ─────────────────────────────────────────────────────────────────────

const entry: PluginMCPEntry = {
	tools,

	async handleToolCall(
		name: string,
		args: Record<string, unknown>,
		context: PluginMCPContext,
	) {
		const api = new PluginAPIClient(context);

		try {
			switch (name) {
				case "checklist_list_checklists": {
					const { projectId, taskId } = args as {
						projectId: string;
						taskId: string;
					};
					const checklists = await api.pluginGet<Checklist[]>(
						`projects/${projectId}/tasks/${taskId}/checklists`,
					);
					if (checklists.length === 0) {
						return textResult("No checklists found for this task.");
					}
					const formatted = checklists.map(formatChecklist).join("\n\n---\n\n");
					return textResult(`Checklists:\n\n${formatted}`);
				}

				case "checklist_create_checklist": {
					const { projectId, taskId, title } = args as {
						projectId: string;
						taskId: string;
						title: string;
					};
					const checklist = await api.pluginPost<Checklist>(
						`projects/${projectId}/tasks/${taskId}/checklists`,
						{ title },
					);
					return textResult(
						`Checklist created successfully:\n\n${formatChecklist(checklist)}`,
					);
				}

				case "checklist_update_checklist": {
					const { projectId, taskId, checklistId, title } = args as {
						projectId: string;
						taskId: string;
						checklistId: string;
						title: string;
					};
					const checklist = await api.pluginPatch<Checklist>(
						`projects/${projectId}/tasks/${taskId}/checklists/${checklistId}`,
						{ title },
					);
					return textResult(
						`Checklist updated successfully:\n\n${formatChecklist(checklist)}`,
					);
				}

				case "checklist_delete_checklist": {
					const { projectId, taskId, checklistId } = args as {
						projectId: string;
						taskId: string;
						checklistId: string;
					};
					await api.pluginDelete(
						`projects/${projectId}/tasks/${taskId}/checklists/${checklistId}`,
					);
					return textResult(
						`Checklist ${checklistId} deleted successfully.`,
					);
				}

				case "checklist_create_item": {
					const { projectId, taskId, checklistId, title } = args as {
						projectId: string;
						taskId: string;
						checklistId: string;
						title: string;
					};
					const item = await api.pluginPost<ChecklistItem>(
						`projects/${projectId}/tasks/${taskId}/checklists/${checklistId}/items`,
						{ title },
					);
					return textResult(
						`Item created:\n${formatItem(item)}`,
					);
				}

				case "checklist_update_item": {
					const {
						projectId,
						taskId,
						checklistId,
						itemId,
						title,
						isChecked,
						assigneeId,
					} = args as {
						projectId: string;
						taskId: string;
						checklistId: string;
						itemId: string;
						title?: string;
						isChecked?: boolean;
						assigneeId?: string;
					};
					const body: Record<string, unknown> = {};
					if (title !== undefined) body.title = title;
					if (isChecked !== undefined) body.is_checked = isChecked;
					if (assigneeId !== undefined) body.assignee_id = assigneeId;
					const item = await api.pluginPatch<ChecklistItem>(
						`projects/${projectId}/tasks/${taskId}/checklists/${checklistId}/items/${itemId}`,
						body,
					);
					return textResult(
						`Item updated:\n${formatItem(item)}`,
					);
				}

				case "checklist_delete_item": {
					const { projectId, taskId, checklistId, itemId } = args as {
						projectId: string;
						taskId: string;
						checklistId: string;
						itemId: string;
					};
					await api.pluginDelete(
						`projects/${projectId}/tasks/${taskId}/checklists/${checklistId}/items/${itemId}`,
					);
					return textResult(`Item ${itemId} deleted successfully.`);
				}

				default:
					return errorResult(`Unknown tool: ${name}`);
			}
		} catch (err) {
			const message = err instanceof Error ? err.message : String(err);
			return errorResult(`Tool ${name} failed: ${message}`);
		}
	},
};

export default entry;
