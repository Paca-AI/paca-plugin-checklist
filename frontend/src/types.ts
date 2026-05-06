// Domain types for the com.paca.checklist plugin frontend.
// These mirror the JSON produced by the backend plugin handlers.

export interface ChecklistItem {
  id: string;
  checklist_id: string;
  title: string;
  is_checked: boolean;
  assignee_id: string | null;
  position: number;
  created_at: string;
  updated_at: string;
}

export interface Checklist {
  id: string;
  task_id: string;
  title: string;
  position: number;
  items: ChecklistItem[];
  created_at: string;
  updated_at: string;
}
