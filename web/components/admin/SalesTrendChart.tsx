"use client";

import { useId, useMemo, useState, type PointerEvent as ReactPointerEvent } from "react";
import { formatRupiah } from "@/lib/format";

// Categorical palette for the 2-series (Online/POS) case — validated with
// the dataviz skill's validate_palette.js (adjacent + normal-vision floor,
// light mode only: this app is a fixed light brand, no dark theme).
// Blue/orange were chosen over the brand accent red so a data series is
// never confused with the site's CTA/promo color.
const COLOR_ONLINE = "#2a78d6";
const COLOR_POS = "#eb6834";
const COLOR_COMBINED = "#111111"; // brand ink — single series needs no legend
const COLOR_GRID = "#e5e5e5"; // --color-border
const COLOR_AXIS_TEXT = "#898781";

type Series = {
  key: string;
  label: string;
  color: string;
  values: number[];
};

const CHART_HEIGHT = 220;
const CHART_PADDING = { top: 16, right: 16, bottom: 28, left: 48 };

function niceMax(max: number): number {
  if (max <= 0) return 1;
  const magnitude = Math.pow(10, Math.floor(Math.log10(max)));
  const normalized = max / magnitude;
  const niceNormalized = normalized <= 1 ? 1 : normalized <= 2 ? 2 : normalized <= 5 ? 5 : 10;
  return niceNormalized * magnitude;
}

export function SalesTrendChart({
  title,
  dates,
  series,
  valueFormat = "number",
}: {
  title: string;
  dates: string[];
  series: Series[];
  // A function prop can't cross the Server -> Client Component boundary, so
  // this takes a discriminator instead of a formatter callback.
  valueFormat?: "number" | "currency";
}) {
  const formatValue = valueFormat === "currency" ? formatRupiah : (n: number) => n.toLocaleString("id-ID");
  const gradientId = useId();
  const [hoverIndex, setHoverIndex] = useState<number | null>(null);
  const [showTable, setShowTable] = useState(false);

  const width = 640;
  const innerWidth = width - CHART_PADDING.left - CHART_PADDING.right;
  const innerHeight = CHART_HEIGHT - CHART_PADDING.top - CHART_PADDING.bottom;

  const maxValue = useMemo(() => {
    const allValues = series.flatMap((s) => s.values);
    return niceMax(Math.max(1, ...allValues));
  }, [series]);

  const xFor = (i: number) =>
    dates.length <= 1 ? innerWidth / 2 : (i / (dates.length - 1)) * innerWidth;
  const yFor = (v: number) => innerHeight - (v / maxValue) * innerHeight;

  const gridLines = [0, 0.25, 0.5, 0.75, 1].map((t) => ({
    y: innerHeight - t * innerHeight,
    value: Math.round(maxValue * t),
  }));

  function handlePointerMove(e: ReactPointerEvent<SVGRectElement>) {
    const rect = e.currentTarget.getBoundingClientRect();
    const x = e.clientX - rect.left;
    const ratio = Math.min(1, Math.max(0, x / innerWidth));
    const index = Math.round(ratio * (dates.length - 1));
    setHoverIndex(index);
  }

  const isMultiSeries = series.length > 1;

  return (
    <div className="border border-border p-4">
      <div className="mb-3 flex items-center justify-between">
        <h3 className="text-xs font-bold uppercase tracking-wide text-ink/60">
          {title}
        </h3>
        <button
          type="button"
          onClick={() => setShowTable((v) => !v)}
          className="text-xs font-bold uppercase text-ink/50 hover:text-accent"
        >
          {showTable ? "Lihat Grafik" : "Lihat Tabel"}
        </button>
      </div>

      {isMultiSeries && (
        <div className="mb-2 flex gap-4">
          {series.map((s) => (
            <div key={s.key} className="flex items-center gap-1.5 text-xs text-ink/70">
              <span
                className="inline-block h-0.5 w-4"
                style={{ backgroundColor: s.color }}
                aria-hidden
              />
              {s.label}
            </div>
          ))}
        </div>
      )}

      {showTable ? (
        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs">
            <thead>
              <tr className="border-b border-border text-ink/50">
                <th className="py-1 pr-3">Tanggal</th>
                {series.map((s) => (
                  <th key={s.key} className="py-1 pr-3">
                    {s.label}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {dates.map((date, i) => (
                <tr key={date} className="border-b border-border">
                  <td className="py-1 pr-3">{date}</td>
                  {series.map((s) => (
                    <td key={s.key} className="py-1 pr-3">
                      {formatValue(s.values[i] ?? 0)}
                    </td>
                  ))}
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <div className="relative">
          <svg
            viewBox={`0 0 ${width} ${CHART_HEIGHT}`}
            className="w-full"
            role="img"
            aria-label={title}
          >
            <defs>
              <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={COLOR_COMBINED} stopOpacity={0.08} />
                <stop offset="100%" stopColor={COLOR_COMBINED} stopOpacity={0} />
              </linearGradient>
            </defs>
            <g transform={`translate(${CHART_PADDING.left},${CHART_PADDING.top})`}>
              {gridLines.map(({ y, value }) => (
                <g key={y}>
                  <line
                    x1={0}
                    x2={innerWidth}
                    y1={y}
                    y2={y}
                    stroke={COLOR_GRID}
                    strokeWidth={1}
                  />
                  <text
                    x={-8}
                    y={y}
                    textAnchor="end"
                    dominantBaseline="middle"
                    fontSize={10}
                    fill={COLOR_AXIS_TEXT}
                  >
                    {value.toLocaleString("id-ID")}
                  </text>
                </g>
              ))}

              {series.map((s) => {
                const points = s.values.map((v, i) => `${xFor(i)},${yFor(v)}`).join(" ");
                const lastIndex = s.values.length - 1;
                return (
                  <g key={s.key}>
                    {!isMultiSeries && (
                      <polygon
                        points={`0,${innerHeight} ${points} ${xFor(lastIndex)},${innerHeight}`}
                        fill={`url(#${gradientId})`}
                      />
                    )}
                    <polyline
                      points={points}
                      fill="none"
                      stroke={s.color}
                      strokeWidth={2}
                      strokeLinejoin="round"
                      strokeLinecap="round"
                    />
                    {lastIndex >= 0 && (
                      <>
                        <circle
                          cx={xFor(lastIndex)}
                          cy={yFor(s.values[lastIndex])}
                          r={4}
                          fill={s.color}
                          stroke="#ffffff"
                          strokeWidth={2}
                        />
                        <text
                          x={xFor(lastIndex)}
                          y={yFor(s.values[lastIndex]) - 10}
                          textAnchor="end"
                          fontSize={10}
                          fontWeight={700}
                          fill={s.color}
                        >
                          {formatValue(s.values[lastIndex])}
                        </text>
                      </>
                    )}
                    {hoverIndex !== null && (
                      <circle
                        cx={xFor(hoverIndex)}
                        cy={yFor(s.values[hoverIndex] ?? 0)}
                        r={4}
                        fill={s.color}
                        stroke="#ffffff"
                        strokeWidth={2}
                      />
                    )}
                  </g>
                );
              })}

              {hoverIndex !== null && (
                <line
                  x1={xFor(hoverIndex)}
                  x2={xFor(hoverIndex)}
                  y1={0}
                  y2={innerHeight}
                  stroke={COLOR_AXIS_TEXT}
                  strokeWidth={1}
                  strokeDasharray="2,2"
                />
              )}

              <rect
                x={0}
                y={0}
                width={innerWidth}
                height={innerHeight}
                fill="transparent"
                onPointerMove={handlePointerMove}
                onPointerLeave={() => setHoverIndex(null)}
              />
            </g>
          </svg>

          {hoverIndex !== null && (
            <div
              className="pointer-events-none absolute top-2 z-10 border border-ink bg-paper px-3 py-2 text-xs shadow-sm"
              style={{
                left: `${(CHART_PADDING.left + xFor(hoverIndex)) / width * 100}%`,
                transform:
                  hoverIndex > dates.length / 2
                    ? "translateX(-100%)"
                    : "translateX(0)",
              }}
            >
              <p className="mb-1 font-bold">{dates[hoverIndex]}</p>
              {series.map((s) => (
                <p key={s.key} className="flex items-center gap-1.5 text-ink/70">
                  <span
                    className="inline-block h-0.5 w-3"
                    style={{ backgroundColor: s.color }}
                    aria-hidden
                  />
                  {s.label}:{" "}
                  <span className="font-bold text-ink">
                    {formatValue(s.values[hoverIndex] ?? 0)}
                  </span>
                </p>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export { COLOR_ONLINE, COLOR_POS, COLOR_COMBINED };
