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
import { ArrowUpCircle } from "lucide-react";
import { useState } from "react";

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

  const triggerUpdate = useMutation({
    mutationFn: () => apiPost("/about/update"),
    onSuccess: () => setDialogOpen(false),
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
            </TableBody>
          </Table>

          {data.updateAvailable && (
            <div className="mt-4">
              <Dialog open={dialogOpen} onOpenChange={setDialogOpen}>
                <DialogTrigger asChild>
                  <Button>
                    <ArrowUpCircle className="mr-2 h-4 w-4" />
                    Update
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
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Phoenixd</CardTitle>
        </CardHeader>
        <CardContent>
          <Table>
            <TableBody>
              <TableRow>
                <TableCell className="font-medium">Version</TableCell>
                <TableCell className="font-mono">
                  {data.phoenixdVersion || "\u2014"}
                </TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </CardContent>
      </Card>
    </div>
  );
}
