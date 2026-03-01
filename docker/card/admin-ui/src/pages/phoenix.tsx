import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api";
import { formatSats } from "@/lib/format";
import { StatCard } from "@/components/stat-card";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Badge } from "@/components/ui/badge";
import { Zap, Coins, Copy, Check } from "lucide-react";
import { useState } from "react";

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

export function PhoenixPage() {
  const { data, isLoading } = useQuery({
    queryKey: ["phoenix"],
    queryFn: () => apiFetch<PhoenixData>("/phoenix"),
    refetchInterval: 30_000,
  });
  const [copied, setCopied] = useState(false);

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
      <h1 className="text-2xl font-bold">Phoenix</h1>

      <div className="grid gap-4 md:grid-cols-2">
        <StatCard title="Balance" value={data.balanceSat} isSats icon={Zap} />
        <StatCard
          title="Fee Credit"
          value={data.feeCreditSat}
          isSats
          icon={Coins}
        />
      </div>

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

      <div>
        <h2 className="mb-4 text-lg font-semibold">Channels</h2>
        {data.channels.length === 0 ? (
          <div className="rounded-lg border border-dashed p-6 text-center text-muted-foreground">
            No channels found.
          </div>
        ) : (
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
                        variant={
                          ch.state === "NORMAL" ? "default" : "secondary"
                        }
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
      </div>
    </div>
  );
}
