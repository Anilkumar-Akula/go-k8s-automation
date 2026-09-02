import { useState } from "react";
import type { ActionResult } from "../types/dashboard";

interface Field {
  label: string;
  value: string;
}

interface ConfirmDialogProps {
  title: string;
  fields: Field[];
  execute: (reason: string) => Promise<ActionResult>;
  onClose: () => void;
}

type Phase = "confirm" | "loading" | "success" | "error";

export default function ConfirmDialog({ title, fields, execute, onClose }: ConfirmDialogProps) {
  const [reason, setReason] = useState("");
  const [phase, setPhase] = useState<Phase>("confirm");
  const [error, setError] = useState("");

  async function handleConfirm() {
    setPhase("loading");
    try {
      await execute(reason);
      setPhase("success");
    } catch (err) {
      setError(err instanceof Error ? err.message : String(err));
      setPhase("error");
    }
  }

  return (
    <div className="dialog-backdrop" onClick={onClose}>
      <div className="dialog" onClick={(e) => e.stopPropagation()}>
        <h3>{title}</h3>

        <dl className="dialog-fields">
          {fields.map((f) => (
            <div key={f.label}>
              <dt>{f.label}</dt>
              <dd>{f.value}</dd>
            </div>
          ))}
        </dl>

        {phase === "confirm" && (
          <>
            <label className="dialog-label">
              Reason
              <textarea
                value={reason}
                onChange={(e) => setReason(e.target.value)}
                placeholder="Why are you doing this?"
                rows={2}
              />
            </label>
            <div className="dialog-actions">
              <button className="btn-secondary" onClick={onClose}>
                Cancel
              </button>
              <button className="btn-primary" onClick={handleConfirm} disabled={reason.trim() === ""}>
                Confirm Action
              </button>
            </div>
          </>
        )}

        {phase === "loading" && <p className="dialog-status">Executing...</p>}

        {phase === "success" && (
          <>
            <p className="dialog-status dialog-status-success">Action completed</p>
            <div className="dialog-actions">
              <button className="btn-primary" onClick={onClose}>
                Close
              </button>
            </div>
          </>
        )}

        {phase === "error" && (
          <>
            <p className="dialog-status dialog-status-error">Action failed: {error}</p>
            <div className="dialog-actions">
              <button className="btn-secondary" onClick={onClose}>
                Close
              </button>
              <button className="btn-primary" onClick={handleConfirm}>
                Retry
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}
