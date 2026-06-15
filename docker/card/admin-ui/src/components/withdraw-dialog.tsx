import { useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { apiFetch, apiPost } from "@/lib/api";
import { formatSats } from "@/lib/format";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { ArrowUpFromLine, AlertTriangle } from "lucide-react";
import { toast } from "sonner";

interface WithdrawInfo {
  nodeBalanceSat: number;
  cardLiabilitySat: number;
  excessSat: number;
}

interface WithdrawResult {
  ok: boolean;
  paymentHash: string;
  feeSat: number;
  breachesLiability: boolean;
}

export function WithdrawDialog() {
  const queryClient = useQueryClient();
  const [open, setOpen] = useState(false);
  const [form, setForm] = useState({
    lnAddress: "",
    amount: "",
    message: "",
    password: "",
  });

  const { data: info } = useQuery({
    queryKey: ["withdraw-info"],
    queryFn: () => apiFetch<WithdrawInfo>("/withdraw"),
    enabled: open,
  });

  const mutation = useMutation({
    mutationFn: (data: {
      lnAddress: string;
      amountSat: number;
      message: string;
      password: string;
    }) => apiPost<WithdrawResult>("/withdraw", data),
    onSuccess: (res) => {
      toast.success(`Withdrew ${formatSats(amountNum)} (fee ${formatSats(res.feeSat)})`);
      queryClient.invalidateQueries({ queryKey: ["phoenix"] });
      queryClient.invalidateQueries({ queryKey: ["phoenix-transactions"] });
      queryClient.invalidateQueries({ queryKey: ["dashboard"] });
      handleOpenChange(false);
    },
  });

  const amountNum = Number(form.amount) || 0;
  // Warn (but don't block) when paying out more than the spare liquidity —
  // this dips into funds owed to cardholders.
  const breaches =
    info !== undefined && amountNum > 0 && amountNum > info.excessSat;
  const exceedsBalance =
    info !== undefined && amountNum > info.nodeBalanceSat;

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    mutation.mutate({
      lnAddress: form.lnAddress.trim(),
      amountSat: amountNum,
      message: form.message,
      password: form.password,
    });
  }

  function handleOpenChange(next: boolean) {
    setOpen(next);
    if (!next) {
      mutation.reset();
      setForm({ lnAddress: "", amount: "", message: "", password: "" });
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button size="sm" variant="outline">
          <ArrowUpFromLine className="mr-2 h-4 w-4" />
          Withdraw
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Withdraw Funds</DialogTitle>
          <DialogDescription>
            Pay out node liquidity to a Lightning address.
          </DialogDescription>
        </DialogHeader>

        {info && (
          <div className="grid grid-cols-2 gap-2 rounded-lg border p-3 text-sm">
            <span className="text-muted-foreground">Node balance</span>
            <span className="text-right font-mono tabular-nums">
              {formatSats(info.nodeBalanceSat)}
            </span>
            <span className="text-muted-foreground">Owed to cards</span>
            <span className="text-right font-mono tabular-nums">
              {formatSats(info.cardLiabilitySat)}
            </span>
            <span className="text-muted-foreground">Spare (safe)</span>
            <span className="text-right font-mono tabular-nums text-[var(--success)]">
              {formatSats(info.excessSat)}
            </span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="lnAddress">Lightning Address</Label>
            <Input
              id="lnAddress"
              value={form.lnAddress}
              onChange={(e) => setForm({ ...form, lnAddress: e.target.value })}
              placeholder="you@wallet.com"
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="amount">Amount (sats)</Label>
            <Input
              id="amount"
              type="number"
              min="1"
              value={form.amount}
              onChange={(e) => setForm({ ...form, amount: e.target.value })}
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="message">Message (optional)</Label>
            <Input
              id="message"
              value={form.message}
              onChange={(e) => setForm({ ...form, message: e.target.value })}
              placeholder="reference / note"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="password">Admin Password</Label>
            <Input
              id="password"
              type="password"
              value={form.password}
              onChange={(e) => setForm({ ...form, password: e.target.value })}
              placeholder="Re-enter to confirm"
              required
            />
          </div>

          {breaches && !exceedsBalance && (
            <Alert variant="destructive">
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>
                This is more than the spare liquidity and will dip into funds
                owed to cardholders.
              </AlertDescription>
            </Alert>
          )}

          {exceedsBalance && (
            <Alert variant="destructive">
              <AlertTriangle className="h-4 w-4" />
              <AlertDescription>
                Amount exceeds the node balance.
              </AlertDescription>
            </Alert>
          )}

          {mutation.error && (
            <p className="text-sm text-destructive">{mutation.error.message}</p>
          )}

          <Button
            type="submit"
            className="w-full"
            disabled={mutation.isPending || exceedsBalance}
          >
            {mutation.isPending ? "Sending..." : "Withdraw"}
          </Button>
        </form>
      </DialogContent>
    </Dialog>
  );
}
