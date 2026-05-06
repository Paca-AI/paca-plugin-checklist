import { Check, Trash2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import type { Checklist, ChecklistItem } from "./types";

function cn(...classes: (string | boolean | undefined | null)[]): string {
  return classes.filter(Boolean).join(" ");
}

interface ChecklistSectionProps {
  checklist: Checklist;
  canEdit: boolean;
  onRename: (id: string, title: string) => void;
  onDelete: (id: string) => void;
  onCreateItem: (checklistId: string, title: string) => void;
  onToggleItem: (checklistId: string, item: ChecklistItem) => void;
  onDeleteItem: (checklistId: string, itemId: string) => void;
}

export function ChecklistSection({
  checklist,
  canEdit,
  onRename,
  onDelete,
  onCreateItem,
  onToggleItem,
  onDeleteItem,
}: ChecklistSectionProps) {
  const [editingTitle, setEditingTitle] = useState(false);
  const [titleDraft, setTitleDraft] = useState(checklist.title);
  const [newItem, setNewItem] = useState("");
  const titleInputRef = useRef<HTMLInputElement>(null);

  const completed = checklist.items.filter((i) => i.is_checked).length;
  const pct =
    checklist.items.length > 0
      ? Math.round((completed / checklist.items.length) * 100)
      : 0;

  useEffect(() => {
    if (!editingTitle) {
      setTitleDraft(checklist.title);
    }
  }, [checklist.title, editingTitle]);

  const submitTitle = () => {
    setEditingTitle(false);
    const trimmed = titleDraft.trim();
    if (trimmed && trimmed !== checklist.title) {
      onRename(checklist.id, trimmed);
      return;
    }
    setTitleDraft(checklist.title);
  };

  const addItem = () => {
    const text = newItem.trim();
    if (!text) return;
    onCreateItem(checklist.id, text);
    setNewItem("");
  };

  return (
    <div className="space-y-3">
      {/* Header */}
      <div className="flex items-center gap-3">
        {editingTitle ? (
          <input
            ref={titleInputRef}
            value={titleDraft}
            onChange={(e) => setTitleDraft(e.target.value)}
            onBlur={submitTitle}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                e.currentTarget.blur();
              }
              if (e.key === "Escape") {
                setEditingTitle(false);
                setTitleDraft(checklist.title);
              }
            }}
            // biome-ignore lint/a11y/noAutofocus: intentional for inline title editing
            autoFocus
            className="text-[13px] font-semibold text-foreground flex-1 min-w-0 bg-transparent outline-none rounded-md -mx-2 px-2 py-1"
            aria-label="Checklist title"
          />
        ) : (
          <button
            type="button"
            onClick={() => {
              if (!canEdit) return;
              setTitleDraft(checklist.title);
              setEditingTitle(true);
              setTimeout(() => titleInputRef.current?.focus(), 0);
            }}
            className={cn(
              "text-[13px] font-semibold text-foreground flex-1 min-w-0 text-left",
              canEdit &&
                "cursor-text rounded-md -mx-2 px-2 py-1 transition-colors hover:bg-muted/30",
            )}
          >
            {checklist.title}
          </button>
        )}
        <span
          className={cn(
            "text-[11px] font-bold tabular-nums rounded-full px-2 py-0.5",
            pct === 100
              ? "bg-emerald-500/15 text-emerald-600"
              : "bg-muted/50 text-muted-foreground/80",
          )}
        >
          {completed}/{checklist.items.length}
        </span>
        {canEdit && (
          <button
            type="button"
            onClick={() => onDelete(checklist.id)}
            className="flex items-center justify-center size-6 rounded-md text-muted-foreground/40 hover:text-destructive hover:bg-destructive/10 transition-colors"
            aria-label="Delete checklist"
          >
            <Trash2 className="size-3.5" />
          </button>
        )}
      </div>

      {/* Progress bar */}
      <div className="h-1.5 rounded-full bg-border/25 overflow-hidden">
        <div
          className={cn(
            "h-full rounded-full transition-all duration-500 ease-out",
            pct === 100
              ? "bg-emerald-500 shadow-sm shadow-emerald-500/30"
              : "bg-primary/60",
          )}
          style={{ width: `${pct}%` }}
        />
      </div>

      {/* Items */}
      <div className="space-y-0.5">
        {checklist.items.map((item) => (
          <div
            key={item.id}
            className="group flex items-center gap-3 rounded-lg px-2 py-2 hover:bg-muted/30 transition-colors duration-150"
          >
            <button
              type="button"
              onClick={() => onToggleItem(checklist.id, item)}
              disabled={!canEdit}
              className={cn(
                "flex size-4.5 shrink-0 items-center justify-center rounded-[5px] border-2 transition-all duration-200",
                item.is_checked
                  ? "border-emerald-500 bg-emerald-500 text-white shadow-sm shadow-emerald-500/20"
                  : "border-border/40 text-transparent hover:border-border/70 hover:bg-muted/40",
              )}
            >
              <Check className="size-2.5" strokeWidth={3} />
            </button>
            <span
              className={cn(
                "flex-1 text-[13px] transition-all duration-200",
                item.is_checked
                  ? "line-through text-muted-foreground/60"
                  : "text-foreground",
              )}
            >
              {item.title}
            </span>
            {canEdit && (
              <button
                type="button"
                onClick={() => onDeleteItem(checklist.id, item.id)}
                className="flex items-center justify-center size-6 rounded-md opacity-0 group-hover:opacity-100 text-muted-foreground/40 hover:text-muted-foreground/70 hover:bg-muted/50 transition-all duration-150"
                aria-label="Delete item"
              >
                <Trash2 className="size-3" />
              </button>
            )}
          </div>
        ))}

        {/* Add item input */}
        {canEdit && (
          <div className="flex items-center gap-3 px-2 pt-1">
            <div className="size-4.5 shrink-0 rounded-[5px] border-2 border-dashed border-border/25" />
            <input
              value={newItem}
              onChange={(e) => setNewItem(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "Enter") addItem();
              }}
              placeholder="Add an item…"
              className="flex-1 bg-transparent text-[13px] outline-none placeholder:text-muted-foreground/45 py-1.5 focus:placeholder:text-muted-foreground/70 transition-colors"
            />
            {newItem && (
              <button
                type="button"
                onClick={addItem}
                className="text-[11px] text-primary/80 font-semibold hover:text-primary transition-colors duration-150"
              >
                Add
              </button>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
