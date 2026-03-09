import type { TutorialSessionKind } from "@/lib/types";

export interface Command {
  name: string;
  description: string;
  /** If set, the command is only available for this session kind. */
  sessionKind?: TutorialSessionKind;
}

export const COMMANDS: Command[] = [
  {
    name: "/problem-set",
    description: "Generate a targeted problem set for practice",
    sessionKind: "extended",
  },
  {
    name: "/review-problem-set",
    description: "Evaluate the previous week's problem set",
    sessionKind: "extended",
  },
  {
    name: "/diagnose",
    description: "Run a canonical diagnose on selected artifacts",
  },
];
