import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { apiPost } from "@/lib/api";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Copy, Check, Plus, ChevronDown } from "lucide-react";

interface BatchResult {
  ok: boolean;
  boltcardLink: string;
  programUrl: string;
  qr: string;
}

const DEFAULTS = {
  groupTag: "",
  maxCards: "1",
  initialBalance: "0",
  expiryHours: "24",
};

export function BatchProgramDialog() {
  const [open, setOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [showAdvanced, setShowAdvanced] = useState(false);
  const [form, setForm] = useState(DEFAULTS);

  const multiple = (Number(form.maxCards) || 0) > 1;

  const mutation = useMutation({
    mutationFn: (data: {
      groupTag: string;
      maxCards: number;
      initialBalance: number;
      expiryHours: number;
    }) => apiPost<BatchResult>("/batch/create", data),
  });

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    mutation.mutate({
      groupTag: form.groupTag,
      maxCards: Number(form.maxCards) || 0,
      initialBalance: Number(form.initialBalance) || 0,
      expiryHours: Number(form.expiryHours) || 0,
    });
  }

  function copyLink() {
    if (mutation.data) {
      navigator.clipboard.writeText(mutation.data.boltcardLink);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    }
  }

  function handleOpenChange(next: boolean) {
    setOpen(next);
    if (!next) {
      mutation.reset();
      setCopied(false);
      setShowAdvanced(false);
      setForm(DEFAULTS);
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button size="sm">
          <Plus className="mr-2 h-4 w-4" />
          Add Card
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md overflow-hidden">
        <DialogHeader>
          <DialogTitle>{multiple ? "Program Cards" : "Program a Card"}</DialogTitle>
          <DialogDescription>
            Creates a programming link.
            <br />
            Scan the QR code with the Bolt Card app to program{" "}
            {multiple ? "each card" : "your card"}.
          </DialogDescription>
        </DialogHeader>

        {mutation.data ? (
          <div className="space-y-4">
            <div className="flex justify-center">
              <div className="rounded-lg bg-white p-4">
                <img
                  src={`data:image/png;base64,${mutation.data.qr}`}
                  alt="Program Card QR"
                  className="w-full max-w-64"
                />
              </div>
            </div>
            {multiple && (
              <p className="text-center text-sm text-muted-foreground">
                Scan this same link once per card &mdash; up to {form.maxCards} cards.
              </p>
            )}
            <Button variant="outline" className="w-full" onClick={copyLink}>
              {copied ? (
                <Check className="mr-2 h-4 w-4 text-[var(--success)]" />
              ) : (
                <Copy className="mr-2 h-4 w-4" />
              )}
              {copied ? "Copied" : "Copy Link"}
            </Button>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="maxCards">Number of cards</Label>
                <Input
                  id="maxCards"
                  type="number"
                  min="1"
                  value={form.maxCards}
                  onChange={(e) =>
                    setForm({ ...form, maxCards: e.target.value })
                  }
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="initialBalance">Balance (sats)</Label>
                <Input
                  id="initialBalance"
                  type="number"
                  min="0"
                  value={form.initialBalance}
                  onChange={(e) =>
                    setForm({ ...form, initialBalance: e.target.value })
                  }
                />
              </div>
            </div>
            {multiple && (
              <p className="text-xs text-muted-foreground">
                One link programs up to this many cards. Leave at 1 for a single card.
              </p>
            )}

            <button
              type="button"
              onClick={() => setShowAdvanced((v) => !v)}
              className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
            >
              <ChevronDown
                className={`h-4 w-4 transition-transform ${
                  showAdvanced ? "rotate-180" : ""
                }`}
              />
              Advanced
            </button>

            {showAdvanced && (
              <div className="grid grid-cols-2 gap-3">
                <div className="space-y-2">
                  <Label htmlFor="groupTag">Group (optional)</Label>
                  <Input
                    id="groupTag"
                    value={form.groupTag}
                    onChange={(e) =>
                      setForm({ ...form, groupTag: e.target.value })
                    }
                    placeholder="e.g. meetup-jan"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="expiryHours">Link expiry (hrs)</Label>
                  <Input
                    id="expiryHours"
                    type="number"
                    min="1"
                    value={form.expiryHours}
                    onChange={(e) =>
                      setForm({ ...form, expiryHours: e.target.value })
                    }
                    required
                  />
                </div>
              </div>
            )}

            {mutation.error && (
              <p className="text-sm text-destructive">
                {mutation.error.message}
              </p>
            )}
            <Button type="submit" className="w-full" disabled={mutation.isPending}>
              {mutation.isPending
                ? "Creating..."
                : multiple
                  ? "Create Link"
                  : "Create Card Link"}
            </Button>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
