import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { useNavigate } from "react-router-dom";
import { apiFetch } from "@/lib/api";
import { formatSats } from "@/lib/format";
import { Input } from "@/components/ui/input";
import { Badge } from "@/components/ui/badge";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Search } from "lucide-react";
import { BatchProgramDialog } from "@/components/batch-program-dialog";

interface CardSummary {
  cardId: number;
  uid: string;
  note: string;
  balanceSats: number;
  lnurlwEnable: string;
  groupTag: string;
  txLimitSats: number;
  dayLimitSats: number;
}

type StatusFilter = "active" | "disabled" | "all";

export function CardsPage() {
  const [search, setSearch] = useState("");
  const [statusFilter, setStatusFilter] = useState<StatusFilter>("active");
  const navigate = useNavigate();

  const { data, isLoading } = useQuery({
    queryKey: ["cards"],
    queryFn: () => apiFetch<{ cards: CardSummary[] }>("/cards"),
  });

  if (isLoading || !data) {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold">Cards</h1>
        <div className="h-64 animate-pulse rounded-lg bg-muted" />
      </div>
    );
  }

  const filtered = data.cards.filter((c) => {
    if (statusFilter === "active" && c.lnurlwEnable !== "Y") return false;
    if (statusFilter === "disabled" && c.lnurlwEnable !== "N") return false;
    if (!search) return true;
    const q = search.toLowerCase();
    return (
      c.note.toLowerCase().includes(q) ||
      c.uid.toLowerCase().includes(q) ||
      c.groupTag.toLowerCase().includes(q) ||
      String(c.cardId).includes(q)
    );
  });

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between md:pr-20">
        <h1 className="text-2xl font-bold">Cards</h1>
        <div className="flex items-center gap-3">
          <span className="text-sm text-muted-foreground">
            {filtered.length} card{filtered.length !== 1 ? "s" : ""}
          </span>
          <BatchProgramDialog />
        </div>
      </div>

      {data.cards.length > 0 && (
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
          <div className="flex gap-1.5">
            {(["active", "disabled", "all"] as const).map((s) => (
              <button
                key={s}
                onClick={() => setStatusFilter(s)}
                className={`rounded-full px-3 py-1 text-xs font-medium transition-colors ${
                  statusFilter === s
                    ? "bg-primary text-primary-foreground"
                    : "bg-muted text-muted-foreground hover:bg-muted/80"
                }`}
              >
                {s.charAt(0).toUpperCase() + s.slice(1)}
              </button>
            ))}
          </div>
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              placeholder="Search by note, UID, or group..."
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              className="pl-9"
            />
          </div>
        </div>
      )}

      {data.cards.length === 0 ? (
        <div className="rounded-lg border border-dashed p-6 text-center text-muted-foreground">
          No cards found. Program your first card.
        </div>
      ) : filtered.length === 0 ? (
        <div className="rounded-lg border border-dashed p-6 text-center text-muted-foreground">
          No cards match your {search ? "search" : "filter"}.
        </div>
      ) : (
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-16">ID</TableHead>
                <TableHead>UID</TableHead>
                <TableHead>Note</TableHead>
                <TableHead className="text-right font-mono">Balance</TableHead>
                <TableHead>Status</TableHead>
                <TableHead>Group</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {filtered.map((card) => (
                <TableRow
                  key={card.cardId}
                  className="cursor-pointer"
                  onClick={() => navigate(`/cards/${card.cardId}`)}
                >
                  <TableCell className="font-mono">{card.cardId}</TableCell>
                  <TableCell className="font-mono text-xs">
                    {card.uid.slice(0, 14)}
                  </TableCell>
                  <TableCell>{card.note || "\u2014"}</TableCell>
                  <TableCell className="text-right font-mono tabular-nums">
                    {formatSats(card.balanceSats)}
                  </TableCell>
                  <TableCell>
                    <Badge
                      variant={
                        card.lnurlwEnable === "Y" ? "default" : "secondary"
                      }
                    >
                      {card.lnurlwEnable === "Y" ? "Active" : "Disabled"}
                    </Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground">
                    {card.groupTag || "\u2014"}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}
    </div>
  );
}
