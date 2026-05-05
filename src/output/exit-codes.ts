export const EXIT_CODES = {
  SUCCESS: 0,
  NO_CHANGE: 0,
  DRY_RUN: 0,
  USER_CANCEL: 130,
  ERROR: 1
} as const;

export type ExitCode = (typeof EXIT_CODES)[keyof typeof EXIT_CODES];
