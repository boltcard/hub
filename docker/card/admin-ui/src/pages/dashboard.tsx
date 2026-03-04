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
import { Badge } from "@/components/ui/badge";
import { CreditCard, Wallet, Zap } from "lucide-react";
import { useNavigate, Link } from "react-router-dom";
import { OnboardingChecklist } from "@/components/onboarding-checklist";

interface DashboardData {
  cardCount: number;
  hasCards: boolean;
  phoenixConnected: boolean;
  phoenixBalance: number;
  phoenixFeeCredit: number;
  totalCardBalance: number;
  topCards: {
    cardId: number;
    note: string;
    balanceSats: number;
    lnurlwEnable: string;
  }[];
}

export function DashboardPage() {
  const navigate = useNavigate();
  const { data, isLoading } = useQuery({
    queryKey: ["dashboard"],
    queryFn: () => apiFetch<DashboardData>("/dashboard"),
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
          title="Active Cards - Total Balance"
          value={data.totalCardBalance}
          isSats
          icon={Wallet}
        />
        <StatCard
          title="Phoenix Balance"
          value={data.phoenixBalance}
          isSats
          icon={Zap}
        />
      </div>

      {data.topCards.length > 0 && (
        <div>
          <h2 className="mb-4 text-lg font-semibold">Top Cards by Balance</h2>
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-16">ID</TableHead>
                  <TableHead>Note</TableHead>
                  <TableHead className="text-right font-mono">Balance</TableHead>
                  <TableHead>Status</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {data.topCards.map((card) => (
                  <TableRow
                    key={card.cardId}
                    className="cursor-pointer"
                    onClick={() => navigate(`/cards/${card.cardId}`)}
                  >
                    <TableCell className="font-mono">{card.cardId}</TableCell>
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
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
          {data.cardCount > data.topCards.length && (
            <div className="mt-2 text-center">
              <Link
                to="/cards"
                className="text-sm text-muted-foreground hover:text-foreground"
              >
                &hellip;
              </Link>
            </div>
          )}
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
