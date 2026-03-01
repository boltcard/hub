import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiFetch, apiPut, apiPost } from "@/lib/api";
import { formatSats, formatTimestamp } from "@/lib/format";
import { toast } from "sonner";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog";
import { ArrowLeft, Pencil, Check, X, Trash2 } from "lucide-react";

interface CardDetail {
  cardId: number;
  uid: string;
  note: string;
  balanceSats: number;
  lnurlwEnable: string;
  txLimitSats: number;
  dayLimitSats: number;
  pinEnable: string;
  pinLimitSats: number;
  wiped: string;
}

interface CardTx {
  receiptId: number;
  paymentId: number;
  timestamp: number;
  amountSats: number;
  feeSats: number;
}

export function CardDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();

  const { data: card, isLoading } = useQuery({
    queryKey: ["card", id],
    queryFn: () => apiFetch<CardDetail>(`/cards/${id}`),
  });

  const { data: txData } = useQuery({
    queryKey: ["card-txs", id],
    queryFn: () => apiFetch<{ txs: CardTx[] }>(`/cards/${id}/txs`),
  });

  // Editable note
  const [editingNote, setEditingNote] = useState(false);
  const [noteValue, setNoteValue] = useState("");

  const noteMutation = useMutation({
    mutationFn: (note: string) => apiPut(`/cards/${id}/note`, { note }),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["card", id] });
      setEditingNote(false);
      toast.success("Note updated");
    },
    onError: (err) => toast.error(err.message),
  });

  // Limits form
  const [limitsForm, setLimitsForm] = useState<{
    txLimitSats: string;
    dayLimitSats: string;
    lnurlwEnable: string;
  } | null>(null);

  const limitsMutation = useMutation({
    mutationFn: (data: {
      txLimitSats: number;
      dayLimitSats: number;
      lnurlwEnable: string;
    }) => apiPut(`/cards/${id}/limits`, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["card", id] });
      setLimitsForm(null);
      toast.success("Limits updated");
    },
    onError: (err) => toast.error(err.message),
  });

  // Wipe
  const wipeMutation = useMutation({
    mutationFn: () => apiPost(`/cards/${id}/wipe`),
    onSuccess: () => {
      toast.success("Card wiped");
      navigate("/cards");
    },
    onError: (err) => toast.error(err.message),
  });

  if (isLoading || !card) {
    return (
      <div className="space-y-4">
        <div className="h-8 w-48 animate-pulse rounded bg-muted" />
        <div className="h-64 animate-pulse rounded-lg bg-muted" />
      </div>
    );
  }

  const txs = txData?.txs ?? [];

  function startEditNote() {
    setNoteValue(card!.note);
    setEditingNote(true);
  }

  function startEditLimits() {
    setLimitsForm({
      txLimitSats: String(card!.txLimitSats),
      dayLimitSats: String(card!.dayLimitSats),
      lnurlwEnable: card!.lnurlwEnable,
    });
  }

  function saveLimits() {
    if (!limitsForm) return;
    limitsMutation.mutate({
      txLimitSats: Number(limitsForm.txLimitSats) || 0,
      dayLimitSats: Number(limitsForm.dayLimitSats) || 0,
      lnurlwEnable: limitsForm.lnurlwEnable,
    });
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={() => navigate("/cards")}>
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <h1 className="text-2xl font-bold">Card #{card.cardId}</h1>
        <Badge variant={card.lnurlwEnable === "Y" ? "default" : "secondary"}>
          {card.lnurlwEnable === "Y" ? "Active" : "Disabled"}
        </Badge>
      </div>

      <div className="grid gap-4 md:grid-cols-2">
        {/* Info card */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Info</CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div>
              <span className="text-sm text-muted-foreground">UID</span>
              <p className="font-mono text-sm">{card.uid}</p>
            </div>
            <div>
              <span className="text-sm text-muted-foreground">Note</span>
              {editingNote ? (
                <div className="flex items-center gap-2">
                  <Input
                    value={noteValue}
                    onChange={(e) => setNoteValue(e.target.value)}
                    className="h-8"
                    autoFocus
                    onKeyDown={(e) => {
                      if (e.key === "Enter") noteMutation.mutate(noteValue);
                      if (e.key === "Escape") setEditingNote(false);
                    }}
                  />
                  <Button
                    size="icon"
                    variant="ghost"
                    className="h-8 w-8"
                    onClick={() => noteMutation.mutate(noteValue)}
                    disabled={noteMutation.isPending}
                  >
                    <Check className="h-4 w-4" />
                  </Button>
                  <Button
                    size="icon"
                    variant="ghost"
                    className="h-8 w-8"
                    onClick={() => setEditingNote(false)}
                  >
                    <X className="h-4 w-4" />
                  </Button>
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <p className="text-sm">{card.note || "\u2014"}</p>
                  <Button
                    size="icon"
                    variant="ghost"
                    className="h-6 w-6"
                    onClick={startEditNote}
                  >
                    <Pencil className="h-3 w-3" />
                  </Button>
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Balance card */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Balance</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-3xl font-bold font-mono tabular-nums">
              {formatSats(card.balanceSats)}
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Limits section */}
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-lg">Limits</CardTitle>
          {!limitsForm && (
            <Button variant="outline" size="sm" onClick={startEditLimits}>
              Edit
            </Button>
          )}
        </CardHeader>
        <CardContent>
          {limitsForm ? (
            <div className="space-y-4">
              <div className="grid gap-4 sm:grid-cols-3">
                <div className="space-y-2">
                  <Label>Tx Limit (sats)</Label>
                  <Input
                    type="number"
                    value={limitsForm.txLimitSats}
                    onChange={(e) =>
                      setLimitsForm({
                        ...limitsForm,
                        txLimitSats: e.target.value,
                      })
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>Day Limit (sats)</Label>
                  <Input
                    type="number"
                    value={limitsForm.dayLimitSats}
                    onChange={(e) =>
                      setLimitsForm({
                        ...limitsForm,
                        dayLimitSats: e.target.value,
                      })
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>Withdrawals</Label>
                  <Select
                    value={limitsForm.lnurlwEnable}
                    onValueChange={(v) =>
                      setLimitsForm({ ...limitsForm, lnurlwEnable: v })
                    }
                  >
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="Y">Enabled</SelectItem>
                      <SelectItem value="N">Disabled</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="flex gap-2">
                <Button
                  size="sm"
                  onClick={saveLimits}
                  disabled={limitsMutation.isPending}
                >
                  Save
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => setLimitsForm(null)}
                >
                  Cancel
                </Button>
              </div>
            </div>
          ) : (
            <div className="grid gap-4 text-sm sm:grid-cols-3">
              <div>
                <span className="text-muted-foreground">Tx Limit</span>
                <p className="font-mono tabular-nums">
                  {formatSats(card.txLimitSats)}
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">Day Limit</span>
                <p className="font-mono tabular-nums">
                  {formatSats(card.dayLimitSats)}
                </p>
              </div>
              <div>
                <span className="text-muted-foreground">Withdrawals</span>
                <p>{card.lnurlwEnable === "Y" ? "Enabled" : "Disabled"}</p>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Transaction history */}
      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Transaction History</CardTitle>
        </CardHeader>
        <CardContent>
          {txs.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              No transactions yet.
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Type</TableHead>
                  <TableHead>Date</TableHead>
                  <TableHead className="text-right font-mono">
                    Amount
                  </TableHead>
                  <TableHead className="text-right font-mono">Fee</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {txs.map((tx, i) => {
                  const isReceipt = tx.receiptId > 0;
                  return (
                    <TableRow key={i}>
                      <TableCell>
                        <Badge variant={isReceipt ? "default" : "secondary"}>
                          {isReceipt ? "Received" : "Sent"}
                        </Badge>
                      </TableCell>
                      <TableCell className="text-sm">
                        {formatTimestamp(tx.timestamp)}
                      </TableCell>
                      <TableCell
                        className={`text-right font-mono tabular-nums ${
                          isReceipt
                            ? "text-[var(--success)]"
                            : "text-destructive"
                        }`}
                      >
                        {isReceipt ? "+" : ""}
                        {formatSats(tx.amountSats)}
                      </TableCell>
                      <TableCell className="text-right font-mono tabular-nums text-muted-foreground">
                        {tx.feeSats !== 0 ? formatSats(tx.feeSats) : "\u2014"}
                      </TableCell>
                    </TableRow>
                  );
                })}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {/* Danger zone */}
      <Card className="border-destructive/50">
        <CardHeader>
          <CardTitle className="text-lg text-destructive">
            Danger Zone
          </CardTitle>
        </CardHeader>
        <CardContent>
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button variant="destructive" size="sm">
                <Trash2 className="mr-2 h-4 w-4" />
                Wipe Card
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Wipe Card #{card.cardId}?</AlertDialogTitle>
                <AlertDialogDescription>
                  This will permanently disable this card. The card's keys will
                  be reset and it can no longer be used for payments. This action
                  cannot be undone.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  onClick={() => wipeMutation.mutate()}
                  disabled={wipeMutation.isPending}
                  className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                >
                  Wipe Card
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        </CardContent>
      </Card>
    </div>
  );
}
