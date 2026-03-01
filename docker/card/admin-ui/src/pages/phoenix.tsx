import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api";
import { formatSats, formatTimestamp } from "@/lib/format";
import { StatCard } from "@/components/stat-card";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Zap, Coins, Copy, Check, ArrowDownLeft, ArrowUpRight } from "lucide-react";
import { useState, useMemo, useCallback } from "react";
import { useWebSocket } from "@/hooks/use-websocket";

interface PhoenixData {
  balanceSat: number;
  feeCreditSat: number;
  offer: string;
  offerQr: string;
  channels: {
    state: string;
    channelId: string;
    balanceMsat: number;
    inboundLiquidMsat: number;
  }[];
}

interface TxItem {
  direction: string;
  amountSat: number;
  paymentHash: string;
  timestamp: number;
  isPaid: boolean;
  description?: string;
  cardNote?: string;
}

export function PhoenixPage() {
  const queryClient = useQueryClient();

  const onEvent = useCallback(() => {
    queryClient.invalidateQueries({ queryKey: ["phoenix"] });
    queryClient.invalidateQueries({ queryKey: ["phoenix-transactions"] });
  }, [queryClient]);

  const { data, isLoading } = useQuery({
    queryKey: ["phoenix"],
    queryFn: () => apiFetch<PhoenixData>("/phoenix"),
    refetchInterval: 30_000,
  });

  const { data: txData } = useQuery({
    queryKey: ["phoenix-transactions"],
    queryFn: () => apiFetch<{ in: TxItem[]; out: TxItem[] }>("/phoenix/transactions"),
  });

  const { received: liveReceived, sent: liveSent, status: wsStatus } = useWebSocket(onEvent);
  const [copied, setCopied] = useState(false);

  // Merge live websocket payments with fetched txs, dedup by paymentHash, limit to 5
  const incomingTxs = useMemo(() => {
    const fetched = txData?.in ?? [];
    const seen = new Set(fetched.map((tx) => tx.paymentHash));
    const live: TxItem[] = liveReceived
      .filter((msg) => !seen.has(msg.paymentHash))
      .map((msg) => ({
        direction: "in",
        amountSat: msg.amountSat,
        paymentHash: msg.paymentHash,
        timestamp: msg.timestamp,
        isPaid: true,
      }));
    return [...live, ...fetched].slice(0, 5);
  }, [txData, liveReceived]);

  const outgoingTxs = useMemo(() => {
    const fetched = txData?.out ?? [];
    const seen = new Set(fetched.map((tx) => tx.paymentHash));
    const live: TxItem[] = liveSent
      .filter((msg) => !seen.has(msg.paymentHash))
      .map((msg) => ({
        direction: "out",
        amountSat: msg.amountSat,
        paymentHash: msg.paymentHash,
        timestamp: msg.timestamp,
        isPaid: true,
      }));
    return [...live, ...fetched].slice(0, 5);
  }, [txData, liveSent]);

  if (isLoading || !data) {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold">Phoenix</h1>
        <div className="grid gap-4 md:grid-cols-2">
          {[1, 2].map((i) => (
            <div key={i} className="h-24 animate-pulse rounded-lg bg-muted" />
          ))}
        </div>
      </div>
    );
  }

  function copyOffer() {
    navigator.clipboard.writeText(data!.offer);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Phoenix</h1>
        <Badge variant={wsStatus === "connected" ? "default" : "secondary"}>
          {wsStatus === "connected" ? "Live" : wsStatus === "connecting" ? "Connecting" : "Disconnected"}
        </Badge>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        <StatCard title="Balance" value={data.balanceSat} isSats icon={Zap} />
        <StatCard
          title="Fee Credit"
          value={data.feeCreditSat}
          isSats
          icon={Coins}
        />
      </div>

      {data.channels.length > 0 && (
        <div className="space-y-3">
          {data.channels.map((ch) => {
            const outbound = Math.floor(ch.balanceMsat / 1000);
            const inbound = Math.floor(ch.inboundLiquidMsat / 1000);
            const total = outbound + inbound;
            const outPct = total > 0 ? (outbound / total) * 100 : 0;

            return (
              <Card key={ch.channelId}>
                <CardContent className="pt-4">
                  <div className="mb-2 flex items-center justify-between">
                    <code className="text-xs text-muted-foreground">
                      {ch.channelId.slice(0, 16)}...
                    </code>
                    <Badge
                      variant={ch.state === "NORMAL" ? "default" : "secondary"}
                    >
                      {ch.state}
                    </Badge>
                  </div>
                  <div className="mb-1 flex h-3 w-full overflow-hidden rounded-full bg-muted">
                    <div
                      className="bg-[var(--success)] transition-all"
                      style={{ width: `${outPct}%` }}
                    />
                  </div>
                  <div className="flex justify-between text-xs">
                    <span className="font-mono tabular-nums text-[var(--success)]">
                      {formatSats(outbound)} out
                    </span>
                    <span className="font-mono tabular-nums text-muted-foreground">
                      {formatSats(inbound)} in
                    </span>
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}

      {data.offer && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Load Sats (Bolt 12 Offer)</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-col items-center gap-4">
            {data.offerQr && (
              <div className="rounded-lg bg-white p-4">
                <img
                  src={`data:image/png;base64,${data.offerQr}`}
                  alt="Bolt 12 Offer QR"
                  className="h-64 w-64"
                  loading="lazy"
                />
              </div>
            )}
            <div className="flex w-full max-w-md items-center gap-2">
              <code className="flex-1 truncate rounded bg-muted px-3 py-2 text-xs">
                {data.offer}
              </code>
              <Button variant="outline" size="icon" onClick={copyOffer}>
                {copied ? (
                  <Check className="h-4 w-4 text-success" />
                ) : (
                  <Copy className="h-4 w-4" />
                )}
              </Button>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Transactions In */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <ArrowDownLeft className="h-4 w-4" />
            Transactions In
          </CardTitle>
        </CardHeader>
        <CardContent>
          {incomingTxs.length === 0 ? (
            <p className="text-sm text-muted-foreground py-4 text-center">
              No incoming payments yet.
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[40%]">Time</TableHead>
                  <TableHead className="w-[35%]">Message</TableHead>
                  <TableHead className="w-[25%] text-right font-mono">Amount</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {incomingTxs.map((tx) => (
                  <TableRow key={tx.paymentHash}>
                    <TableCell className="text-sm">
                      {formatTimestamp(tx.timestamp)}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {tx.description || "\u2014"}
                    </TableCell>
                    <TableCell className="text-right font-mono tabular-nums text-[var(--success)]">
                      +{formatSats(tx.amountSat)}
                    </TableCell>
                  </TableRow>
                ))}
                <TableRow>
                  <TableCell colSpan={3} className="text-center text-muted-foreground">
                    ...
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Transactions Out */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2 text-lg">
            <ArrowUpRight className="h-4 w-4" />
            Transactions Out
          </CardTitle>
        </CardHeader>
        <CardContent>
          {outgoingTxs.length === 0 ? (
            <p className="text-sm text-muted-foreground py-4 text-center">
              No outgoing payments yet.
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead className="w-[40%]">Time</TableHead>
                  <TableHead className="w-[35%]">Card</TableHead>
                  <TableHead className="w-[25%] text-right font-mono">Amount</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {outgoingTxs.map((tx) => (
                  <TableRow key={tx.paymentHash}>
                    <TableCell className="text-sm">
                      {formatTimestamp(tx.timestamp)}
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {tx.cardNote || "\u2014"}
                    </TableCell>
                    <TableCell className="text-right font-mono tabular-nums text-destructive">
                      -{formatSats(tx.amountSat)}
                    </TableCell>
                  </TableRow>
                ))}
                <TableRow>
                  <TableCell colSpan={3} className="text-center text-muted-foreground">
                    ...
                  </TableCell>
                </TableRow>
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
