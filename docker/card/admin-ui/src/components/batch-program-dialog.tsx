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
import { Copy, Check, Plus } from "lucide-react";

interface BatchResult {
  ok: boolean;
  boltcardLink: string;
  programUrl: string;
  qr: string;
}

export function BatchProgramDialog() {
  const [open, setOpen] = useState(false);
  const [copied, setCopied] = useState(false);
  const [form, setForm] = useState({
    groupTag: "",
    maxCards: "10",
    initialBalance: "0",
    expiryHours: "24",
  });

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
      setForm({ groupTag: "", maxCards: "10", initialBalance: "0", expiryHours: "24" });
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button size="sm">
          <Plus className="mr-2 h-4 w-4" />
          Batch Program
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Batch Program Cards</DialogTitle>
          <DialogDescription>
            Create a programming link for multiple NFC cards. Scan the QR code
            with the Bolt Card app.
          </DialogDescription>
        </DialogHeader>

        {mutation.data ? (
          <div className="space-y-4">
            {mutation.data.qr && (
              <div className="flex justify-center">
                <div className="rounded-lg bg-white p-4">
                  <img
                    src={`data:image/png;base64,${mutation.data.qr}`}
                    alt="Batch Program QR"
                    className="h-56 w-56"
                  />
                </div>
              </div>
            )}
            <div className="flex items-center gap-2">
              <code className="flex-1 truncate rounded bg-muted px-3 py-2 text-xs">
                {mutation.data.boltcardLink}
              </code>
              <Button variant="outline" size="icon" onClick={copyLink}>
                {copied ? (
                  <Check className="h-4 w-4 text-[var(--success)]" />
                ) : (
                  <Copy className="h-4 w-4" />
                )}
              </Button>
            </div>
          </div>
        ) : (
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="groupTag">Group Tag</Label>
              <Input
                id="groupTag"
                value={form.groupTag}
                onChange={(e) =>
                  setForm({ ...form, groupTag: e.target.value })
                }
                placeholder="e.g. meetup-jan"
                required
              />
            </div>
            <div className="grid grid-cols-3 gap-3">
              <div className="space-y-2">
                <Label htmlFor="maxCards">Max Cards</Label>
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
              <div className="space-y-2">
                <Label htmlFor="expiryHours">Expiry (hrs)</Label>
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
            {mutation.error && (
              <p className="text-sm text-destructive">
                {mutation.error.message}
              </p>
            )}
            <Button type="submit" className="w-full" disabled={mutation.isPending}>
              {mutation.isPending ? "Creating..." : "Create Batch"}
            </Button>
          </form>
        )}
      </DialogContent>
    </Dialog>
  );
}
