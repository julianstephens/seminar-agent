// TypeScript mirror of the backend DTOs defined in internal/http/dto.go and
// internal/domain/*.go. Keep in sync when backend types change.

// ── Shared ────────────────────────────────────────────────────────────────────

export interface ApiError {
  error: string;
  details?: unknown;
}

// ── Seminar ───────────────────────────────────────────────────────────────────

export interface Seminar {
  id: string;
  title: string;
  author?: string;
  edition_notes?: string;
  thesis_current: string;
  default_mode: string;
  default_recon_minutes: number;
  created_at: string;
  updated_at: string;
}

export interface CreateSeminarInput {
  title: string;
  author?: string;
  edition_notes?: string;
  thesis_current: string;
  default_mode?: string;
  default_recon_minutes?: number;
}

export interface UpdateSeminarInput {
  title?: string;
  author?: string;
  edition_notes?: string;
  default_mode?: string;
  default_recon_minutes?: number;
}

// ── Session ───────────────────────────────────────────────────────────────────

export type SessionStatus = "in_progress" | "complete" | "abandoned";
export type SessionPhase =
  | "reconstruction"
  | "opposition"
  | "reversal"
  | "residue_required"
  | "done";

export interface Session {
  id: string;
  seminar_id: string;
  section_label: string;
  mode: string;
  excerpt_text?: string;
  excerpt_hash?: string;
  status: SessionStatus;
  phase: SessionPhase;
  recon_minutes: number;
  phase_started_at: string;
  phase_ends_at: string;
  started_at: string;
  ended_at?: string;
  residue_text?: string;
}

export interface Turn {
  id: string;
  session_id: string;
  phase: SessionPhase;
  speaker: string;
  text: string;
  flags: string[];
  created_at: string;
}

export interface SessionDetail extends Session {
  turns: Turn[];
}

export interface CreateSessionInput {
  section_label: string;
  mode?: string;
  excerpt_text?: string;
  recon_minutes?: number;
}

// ── Tutorial ──────────────────────────────────────────────────────────────────

export interface Tutorial {
  id: string;
  title: string;
  subject: string;
  description?: string;
  difficulty: "beginner" | "intermediate" | "advanced";
  created_at: string;
  updated_at: string;
}

export interface CreateTutorialInput {
  title: string;
  subject: string;
  description?: string;
  difficulty?: "beginner" | "intermediate" | "advanced";
}

export interface UpdateTutorialInput {
  title?: string;
  subject?: string;
  description?: string;
  difficulty?: "beginner" | "intermediate" | "advanced";
}

// ── TutorialSession ───────────────────────────────────────────────────────────

export type TutorialSessionStatus = "in_progress" | "complete" | "abandoned";

export type TutorialSessionKind = "diagnostic" | "extended";

export interface TutorialSession {
  id: string;
  tutorial_id: string;
  status: TutorialSessionStatus;
  kind?: TutorialSessionKind;
  notes?: string;
  started_at: string;
  ended_at?: string;
}

export interface CreateTutorialSessionInput {
  kind?: TutorialSessionKind;
}

export type TutorialTurnSpeaker = "user" | "agent" | "system";

export interface TutorialTurn {
  id: string;
  session_id: string;
  speaker: TutorialTurnSpeaker;
  text: string;
  created_at: string;
}

export interface SubmitTutorialTurnResponse {
  user_turn: TutorialTurn;
  agent_turn?: TutorialTurn;
}

export interface AgentResponseChunkPayload {
  session_id: string;
  turn_id: string;
  chunk: string;
  is_final: boolean;
}

export interface TutorialSessionDetail extends TutorialSession {
  artifacts: Artifact[];
  turns: TutorialTurn[];
}

// ── Artifact ──────────────────────────────────────────────────────────────────

export type ArtifactKind = "summary" | "notes" | "problem_set" | "diagnostic";

export interface Artifact {
  id: string;
  session_id: string;
  kind: ArtifactKind;
  title: string;
  content: string;
  created_at: string;
}

export interface CreateArtifactInput {
  kind: ArtifactKind;
  title: string;
  content: string;
}
