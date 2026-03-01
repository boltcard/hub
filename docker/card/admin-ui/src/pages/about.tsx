import { useQuery, useMutation } from "@tanstack/react-query";
import { apiFetch, apiPost } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import {
  Table,
  TableBody,
  TableCell,
  TableRow,
} from "@/components/ui/table";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { ArrowUpCircle, Loader2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";

interface AboutData {
  version: string;
  buildDate: string;
  buildTime: string;
  phoenixdVersion: string;
  latestVersion: string;
  updateAvailable: boolean;
}

export function AboutPage() {
  const { data, isLoading } = useQuery({
    queryKey: ["about"],
    queryFn: () => apiFetch<AboutData>("/about"),
  });

  const [dialogOpen, setDialogOpen] = useState(false);

  const [updating, setUpdating] = useState(false);

  const triggerUpdate = useMutation({
    mutationFn: () => apiPost("/about/update"),
    onSuccess: () => {
      setDialogOpen(false);
      setUpdating(true);
      toast.success("Update triggered — restarting containers…");
      // Poll until the server comes back with a new version
      const poll = setInterval(async () => {
        try {
          const res = await fetch("/admin/api/about");
          if (res.ok) {
            clearInterval(poll);
            window.location.reload();
          }
        } catch {
          // server still restarting
        }
      }, 3000);
      // Stop polling after 2 minutes
      setTimeout(() => clearInterval(poll), 120_000);
    },
    onError: (err) => {
      toast.error("Update failed: " + err.message);
    },
  });

  if (isLoading || !data) {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold">About</h1>
        <div className="h-48 animate-pulse rounded-lg bg-muted" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">About</h1>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Software</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableBody>
              <TableRow>
                <TableCell className="font-medium">Version</TableCell>
                <TableCell className="font-mono">{data.version}</TableCell>
              </TableRow>
              <TableRow>
                <TableCell className="font-medium">Build Date</TableCell>
                <TableCell>{data.buildDate}</TableCell>
              </TableRow>
              <TableRow>
                <TableCell className="font-medium">Build Time</TableCell>
                <TableCell>{data.buildTime}</TableCell>
              </TableRow>
              <TableRow>
                <TableCell className="font-medium">Latest Version</TableCell>
                <TableCell>
                  <span className="font-mono">
                    {data.latestVersion || "unable to check"}
                  </span>
                  {data.updateAvailable && (
                    <Badge variant="default" className="ml-2">
                      Update available
                    </Badge>
                  )}
                </TableCell>
              </TableRow>
              <TableRow>
                <TableCell className="font-medium">Phoenixd Version</TableCell>
                <TableCell className="font-mono">
                  {data.phoenixdVersion || "\u2014"}
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>

          {data.updateAvailable && !updating && (
            <div className="mt-4">
              <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
                <DialogTrigger asChild>
                  <Button>
                    <ArrowUpCircle className="mr-2 h-4 w-4" />
                    Update to {data.latestVersion}
                  </Button>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>Confirm Update</DialogTitle>
                    <DialogDescription>
                      Pull latest images and restart containers?
                    </DialogDescription>
                  </DialogHeader>
                  <DialogFooter>
                    <Button
                      variant="outline"
                      onClick={() => setDialogOpen(false)}
                    >
                      Cancel
                    </Button>
                    <Button
                      onClick={() => triggerUpdate.mutate()}
                      disabled={triggerUpdate.isPending}
                    >
                      {triggerUpdate.isPending ? "Updating..." : "Update"}
                    </Button>
                  </DialogFooter>
                </DialogContent>
              </Dialog>
            </div>
          )}

          {updating && (
            <div className="mt-4 flex items-center gap-3 rounded-lg border p-4">
              <Loader2 className="h-5 w-5 animate-spin text-muted-foreground" />
              <div>
                <p className="font-medium">Updating…</p>
                <p className="text-sm text-muted-foreground">
                  Pulling images and restarting containers. This page will
                  reload automatically.
                </p>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

    </div>
  );
}
