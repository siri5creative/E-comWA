export type PeriodPreset = "today" | "7d" | "30d" | "month" | "custom";

export const PERIOD_PRESET_LABELS: Record<PeriodPreset, string> = {
  today: "Hari Ini",
  "7d": "7 Hari",
  "30d": "30 Hari",
  month: "Bulan Ini",
  custom: "Custom",
};

function fmt(d: Date): string {
  return d.toISOString().slice(0, 10);
}

// Resolves the period preset buttons (PRD 6.9: "harian, mingguan, bulanan,
// atau rentang tanggal custom") into concrete from/to day strings. Day
// boundaries are UTC, matching the backend (api/internal/handlers/reports.go).
export function resolvePeriod(
  preset: string | undefined,
  fromParam?: string,
  toParam?: string
): { from: string; to: string; preset: PeriodPreset } {
  if (preset === "custom" && fromParam && toParam) {
    return { from: fromParam, to: toParam, preset: "custom" };
  }

  const today = new Date();
  const todayStr = fmt(today);

  switch (preset) {
    case "today":
      return { from: todayStr, to: todayStr, preset: "today" };
    case "7d": {
      const from = new Date(today);
      from.setUTCDate(from.getUTCDate() - 6);
      return { from: fmt(from), to: todayStr, preset: "7d" };
    }
    case "month": {
      const from = new Date(Date.UTC(today.getUTCFullYear(), today.getUTCMonth(), 1));
      return { from: fmt(from), to: todayStr, preset: "month" };
    }
    case "30d":
    default: {
      const from = new Date(today);
      from.setUTCDate(from.getUTCDate() - 29);
      return { from: fmt(from), to: todayStr, preset: "30d" };
    }
  }
}
