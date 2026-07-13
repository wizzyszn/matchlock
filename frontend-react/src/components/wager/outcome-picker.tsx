import { Check } from "lucide-react";

import { TeamFlag } from "@/components/wager/team-flag";
import type { Match, Side } from "@/lib/api";
import { formatOdds, matchLabels } from "@/lib/match-display";
import { referenceOddsForSide, sideShortLabel } from "@/lib/wager-sides";
import { cn } from "@/lib/utils";

export type OutcomeDensity = "default" | "compact";

export type OutcomeOptionProps = {
  side: Side;
  match?: Match;
  selected: boolean;
  onSelect: () => void;
  showOdds?: boolean;
  density?: OutcomeDensity;
};

export function OutcomeOption({
  side,
  match,
  selected,
  onSelect,
  showOdds = true,
  density = "default",
}: OutcomeOptionProps) {
  const compact = density === "compact";
  const labels = match ? matchLabels(match) : null;
  const odds =
    match && showOdds ? referenceOddsForSide(side, match.odds) : null;
  const isDraw = side === "draw";
  const displayName = match
    ? side === "home"
      ? labels!.homeTeam
      : side === "away"
        ? labels!.awayTeam
        : "Draw"
    : side === "draw"
      ? "Draw"
      : side === "home"
        ? "Home"
        : "Away";

  return (
    <button
      type="button"
      role="radio"
      aria-checked={selected}
      onClick={onSelect}
      className={cn(
        "group flex w-full items-center text-left transition-all",
        "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-1 focus-visible:ring-ring",
        compact
          ? "min-h-12 gap-2.5 rounded-lg border px-3 py-2"
          : "min-h-15 gap-3.5 rounded-xl border px-4 py-3",
        selected
          ? "border-primary bg-primary/5 shadow-sm ring-1 ring-primary/20"
          : "border-border bg-card hover:border-primary/40 hover:bg-muted/30",
      )}
    >
      {/* 1 / X / 2 Label */}
      <span
        className={cn(
          "flex shrink-0 items-center justify-center rounded bg-muted/60 font-bold tracking-widest text-muted-foreground transition-colors group-hover:bg-muted group-hover:text-foreground",
          selected
            ? "bg-primary/20 text-primary group-hover:bg-primary/20 group-hover:text-primary"
            : "",
          compact ? "size-6 text-[10px]" : "size-7 text-[11px]",
        )}
      >
        {sideShortLabel(side)}
      </span>

      {/* Identity (Flag or Draw icon) */}
      {isDraw ? (
        <span
          className={cn(
            "flex shrink-0 items-center justify-center font-bold text-muted-foreground/40",
            compact ? "size-5 text-sm" : "size-6 text-base",
          )}
          aria-hidden
        >
          =
        </span>
      ) : match ? (
        <TeamFlag
          name={displayName}
          size={compact ? "sm" : "md"}
          className={cn(
            "shrink-0 shadow-sm transition-transform group-hover:scale-105",
            compact ? "p-0.5" : "",
          )}
        />
      ) : (
        <span
          className={cn(
            "flex shrink-0 items-center justify-center rounded-full border border-border bg-muted font-semibold text-muted-foreground",
            compact ? "size-5 text-[10px]" : "size-6 text-xs",
          )}
          aria-hidden
        />
      )}

      {/* Name */}
      <div className="min-w-0 flex-1 ml-1.5">
        <p
          className={cn(
            "truncate text-foreground transition-colors",
            selected ? "font-semibold" : "font-medium",
            compact ? "text-sm" : "text-base",
          )}
        >
          {displayName}
        </p>
      </div>

      {/* Odds & Check */}
      <div className="flex shrink-0 items-center gap-3">
        {showOdds && match ? (
          odds != null ? (
            <span
              className={cn(
                "tabular-nums transition-colors",
                selected
                  ? "font-bold text-primary text-base"
                  : "font-semibold text-muted-foreground",
                compact && !selected ? "text-sm" : "",
              )}
            >
              {formatOdds(odds)}
            </span>
          ) : (
            <span
              className={
                compact
                  ? "text-xs text-muted-foreground"
                  : "text-sm text-muted-foreground"
              }
            >
              —
            </span>
          )
        ) : null}

        {/* Subtle selected indicator */}
        <div
          className={cn(
            "flex size-5 items-center justify-center rounded-full transition-colors",
            selected ? "bg-primary text-primary-foreground" : "bg-transparent",
          )}
        >
          {selected && <Check className="size-3.5 stroke-3" aria-hidden />}
        </div>
      </div>
    </button>
  );
}

export type OutcomePickerProps = {
  match?: Match;
  sides: Side[];
  selected: Side;
  onSelect: (side: Side) => void;
  label?: string;
  hint?: string;
  showOdds?: boolean;
  density?: OutcomeDensity;
  className?: string;
};

export function OutcomePicker({
  match,
  sides,
  selected,
  onSelect,
  label = "Pick your outcome",
  hint,
  showOdds = true,
  density = "default",
  className,
}: OutcomePickerProps) {
  const compact = density === "compact";

  return (
    <div className={cn(compact ? "space-y-2" : "space-y-2", className)}>
      <div
        className={cn(
          "font-medium",
          compact ? "text-xs text-muted-foreground" : "text-sm",
        )}
      >
        {label}
        {hint ? (
          <p className="text-xs text-muted-foreground pt-1">{hint}</p>
        ) : null}
      </div>
      <div
        role="radiogroup"
        aria-label={label}
        className={cn(compact ? "space-y-2" : "space-y-3")}
      >
        {sides.map((side) => (
          <OutcomeOption
            key={side}
            side={side}
            match={match}
            selected={selected === side}
            onSelect={() => onSelect(side)}
            showOdds={showOdds}
            density={density}
          />
        ))}
      </div>
    </div>
  );
}
