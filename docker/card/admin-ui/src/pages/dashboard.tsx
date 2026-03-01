import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api";
import { formatSats } from "@/lib/format";
import { StatCard } from "@/components/stat-card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { CreditCard, Zap, Coins } from "lucide-react";
import { OnboardingChecklist } from "@/components/onboarding-checklist";

interface DashboardData {
  cardCount: number;
  hasCards: boolean;
  phoenixConnected: boolean;
  phoenixBalance: number;
  phoenixFeeCredit: number;
  topCards: {
    cardId: number;
    note: string;
    balanceSats: number;
  }[];
}

export function DashboardPage() {
  const { data, isLoading } = useQuery({
    queryKey: ["dashboard"],
    queryFn: () => apiFetch<DashboardData>("/dashboard"),
    refetchInterval: 30_000,
  });

  if (isLoading || !data) {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold">Dashboard</h1>
        <div className="grid gap-4 md:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              className="h-24 animate-pulse rounded-lg bg-muted"
            />
          ))}
        </div>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Dashboard</h1>

      <div className="grid gap-4 md:grid-cols-3">
        <StatCard
          title="Total Cards"
          value={data.cardCount}
          icon={CreditCard}
        />
        <StatCard
          title="Phoenix Balance"
          value={data.phoenixBalance}
          isSats
          icon={Zap}
        />
        <StatCard
          title="Fee Credit"
          value={data.phoenixFeeCredit}
          isSats
          icon={Coins}
        />
      </div>

      {data.topCards.length > 0 && (
        <div>
          <h2 className="mb-4 text-lg font-semibold">Top Cards by Balance</h2>
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Card ID</TableHead>
                <TableHead>Note</TableHead>
                <TableHead className="text-right font-mono">Balance</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {data.topCards.map((card) => (
                <TableRow key={card.cardId}>
                  <TableCell>{card.cardId}</TableCell>
                  <TableCell>{card.note || "\u2014"}</TableCell>
                  <TableCell className="text-right font-mono tabular-nums">
                    {formatSats(card.balanceSats)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      {(!data.hasCards || !data.phoenixConnected) && (
        <OnboardingChecklist
          phoenixConnected={data.phoenixConnected}
          phoenixBalance={data.phoenixBalance}
          hasCards={data.hasCards}
        />
      )}
    </div>
  );
}
