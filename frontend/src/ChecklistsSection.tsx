import { PluginApiClient, PluginQueryClientProvider } from "@paca-ai/plugin-sdk-react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ListChecks, Plus } from "lucide-react";
import { useMemo } from "react";
import { ChecklistSection } from "./ChecklistSection";
import type { Checklist, ChecklistItem } from "./types";

// ── Constants ─────────────────────────────────────────────────────────────────

const PLUGIN_ID = "com.paca.checklist";

// ── Props ─────────────────────────────────────────────────────────────────────

interface ChecklistsSectionProps {
  projectId: string;
  taskId: string;
  canEdit?: boolean;
}

// ── Component ─────────────────────────────────────────────────────────────────

/**
 * ChecklistsSection — the entry component exposed by the checklist plugin.
 *
 * Receives props directly from the host's <ExtensionPoint> spread and builds
 * its own PluginApiClient using window.location.origin so it can run as an
 * independent micro-frontend.
 */
export default function ChecklistsSection(props: ChecklistsSectionProps) {
  return (
    <PluginQueryClientProvider>
      <ChecklistsSectionInner {...props} />
    </PluginQueryClientProvider>
  );
}

function ChecklistsSectionInner({
  projectId,
  taskId,
  canEdit = false,
}: ChecklistsSectionProps) {
  const api = useMemo(
    () =>
      new PluginApiClient({
        baseUrl: `${window.location.origin}/api/v1`,
        projectId,
        fetch: (url, init) =>
          window.fetch(url, { ...init, credentials: "include" }),
      }),
    [projectId],
  );

  const qc = useQueryClient();
  const queryKey = ["plugin", PLUGIN_ID, "checklists", projectId, taskId];

  // ── Query ──────────────────────────────────────────────────────────────────

  const { data: checklists = [], isLoading } = useQuery<Checklist[]>({
    queryKey,
    queryFn: () =>
      api.pluginGet<Checklist[]>(PLUGIN_ID, `/tasks/${taskId}/checklists`),
  });

  // ── Mutations ──────────────────────────────────────────────────────────────

  const invalidate = () => qc.invalidateQueries({ queryKey });

  const createChecklist = useMutation({
    mutationFn: () =>
      api.pluginPost<Checklist>(PLUGIN_ID, `/tasks/${taskId}/checklists`, {
        title: `Checklist ${checklists.length + 1}`,
      }),
    onSuccess: invalidate,
  });

  const deleteChecklist = useMutation({
    mutationFn: (checklistId: string) =>
      api.pluginDelete(PLUGIN_ID, `/tasks/${taskId}/checklists/${checklistId}`),
    onSuccess: invalidate,
  });

  const createItem = useMutation({
    mutationFn: ({
      checklistId,
      title,
    }: {
      checklistId: string;
      title: string;
    }) =>
      api.pluginPost<ChecklistItem>(
        PLUGIN_ID,
        `/tasks/${taskId}/checklists/${checklistId}/items`,
        { title },
      ),
    onSuccess: invalidate,
  });

  const updateItem = useMutation({
    mutationFn: ({
      checklistId,
      itemId,
      patch,
    }: {
      checklistId: string;
      itemId: string;
      patch: { is_checked?: boolean; title?: string };
    }) =>
      api.pluginPatch<ChecklistItem>(
        PLUGIN_ID,
        `/tasks/${taskId}/checklists/${checklistId}/items/${itemId}`,
        patch,
      ),
    onSuccess: invalidate,
  });

  const deleteItem = useMutation({
    mutationFn: ({
      checklistId,
      itemId,
    }: {
      checklistId: string;
      itemId: string;
    }) =>
      api.pluginDelete(
        PLUGIN_ID,
        `/tasks/${taskId}/checklists/${checklistId}/items/${itemId}`,
      ),
    onSuccess: invalidate,
  });

  // ── Render ─────────────────────────────────────────────────────────────────

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-[11px] font-semibold uppercase tracking-[0.08em] text-muted-foreground/70 flex items-center gap-2">
          <span>Checklists</span>
          <div className="flex-1 h-px bg-linear-to-r from-border/40 to-transparent" />
        </h3>
        {canEdit && (
          <button
            type="button"
            onClick={() => createChecklist.mutate()}
            className="flex items-center gap-1.5 rounded-lg bg-muted/40 text-muted-foreground/80 hover:bg-muted/60 hover:text-foreground px-2.5 py-1.5 text-[11px] font-semibold transition-all duration-150"
          >
            <Plus className="size-3" />
            Create checklist
          </button>
        )}
      </div>

      {isLoading ? null : checklists.length > 0 ? (
        <div className="space-y-3">
          {checklists.map((cl) => (
            <div
              key={cl.id}
              className="rounded-xl border border-border/25 bg-card/50 p-4"
            >
              <ChecklistSection
                checklist={cl}
                canEdit={canEdit}
                onRename={(id, title) =>
                  api.pluginPatch(PLUGIN_ID, `/tasks/${taskId}/checklists/${id}`, { title }).then(invalidate)
                }
                onDelete={(id) => deleteChecklist.mutate(id)}
                onCreateItem={(checklistId, title) =>
                  createItem.mutate({ checklistId, title })
                }
                onToggleItem={(checklistId, item) =>
                  updateItem.mutate({
                    checklistId,
                    itemId: item.id,
                    patch: { is_checked: !item.is_checked },
                  })
                }
                onDeleteItem={(checklistId, itemId) =>
                  deleteItem.mutate({ checklistId, itemId })
                }
              />
            </div>
          ))}
        </div>
      ) : (
        <div className="flex items-center gap-3 px-1 py-3 text-muted-foreground/45">
          <ListChecks className="size-4 opacity-70" />
          <p className="text-[13px] italic">No checklists yet</p>
        </div>
      )}
    </div>
  );
}
