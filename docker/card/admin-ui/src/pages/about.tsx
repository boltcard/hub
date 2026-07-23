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
import { useState, type ReactNode } from "react";
import { toast } from "sonner";

interface AboutData {
  version: string;
  buildDate: string;
  buildTime: string;
  latestVersion: string;
  updateAvailable: boolean;
}

interface LogsData {
  logs: string[];
}

interface Release {
  version: string;
  name: string;
  body: string;
  date: string;
  url: string;
  isCurrent: boolean;
}

interface ReleasesData {
  releases: Release[];
}

// linkify turns bare http(s) URLs in a line into anchor elements; other text is
// returned verbatim (React escapes it).
function linkify(text: string): ReactNode {
  const parts = text.split(/(https?:\/\/[^\s]+)/g);
  return parts.map((part, i) =>
    /^https?:\/\//.test(part) ? (
      <a
        key={i}
        href={part}
        target="_blank"
        rel="noopener noreferrer"
        className="text-primary hover:underline"
      >
        {part}
      </a>
    ) : (
      <span key={i}>{part}</span>
    ),
  );
}

// ReleaseNotes renders a release body: "- " lines become a bullet list, blank
// lines break groups, everything else is a paragraph. No markdown dependency.
function ReleaseNotes({ body }: { body: string }) {
  const elements: ReactNode[] = [];
  let bullets: string[] = [];

  const flush = () => {
    if (bullets.length > 0) {
      const items = bullets;
      elements.push(
        <ul
          key={elements.length}
          className="list-disc space-y-0.5 pl-5 text-sm"
        >
          {items.map((b, i) => (
            <li key={i}>{linkify(b)}</li>
          ))}
        </ul>,
      );
      bullets = [];
    }
  };

  for (const raw of body.split("\n")) {
    const line = raw.trimEnd();
    if (line.startsWith("- ")) {
      bullets.push(line.slice(2));
    } else if (line.trim() === "") {
      flush();
    } else {
      flush();
      elements.push(
        <p key={elements.length} className="text-sm text-muted-foreground">
          {linkify(line)}
        </p>,
      );
    }
  }
  flush();

  return <div className="space-y-2">{elements}</div>;
}

export function AboutPage() {
  const { data, isLoading } = useQuery({
    queryKey: ["about"],
    queryFn: () => apiFetch<AboutData>("/about"),
    // poll so a newly-published version surfaces the "Update available" button
    // without a manual refresh; the backend caches the Docker Hub check
    refetchInterval: 60_000,
  });

  const { data: logsData } = useQuery({
    queryKey: ["about-logs"],
    queryFn: () => apiFetch<LogsData>("/about/logs"),
  });

  const { data: releasesData } = useQuery({
    queryKey: ["about-releases", data?.latestVersion],
    queryFn: () =>
      apiFetch<ReleasesData>(
        `/about/releases?latest=${encodeURIComponent(data?.latestVersion ?? "")}`,
      ),
    enabled: !!data,
  });

  const [dialogOpen, setDialogOpen] = useState(false);

  const [updating, setUpdating] = useState(false);

  const triggerUpdate = useMutation({
    mutationFn: () => apiPost("/about/update"),
    onSettled: () => {
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

      {logsData && logsData.logs.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Recent Logs</CardTitle>
          </CardHeader>
          <CardContent>
            <pre
              className="overflow-x-auto rounded-md bg-muted p-3 text-xs leading-relaxed"
              dangerouslySetInnerHTML={{ __html: logsData.logs.join("\n") }}
            />
          </CardContent>
        </Card>
      )}

      {releasesData && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Recent Releases</CardTitle>
          </CardHeader>
          <CardContent className="space-y-6">
            {releasesData.releases.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                No release notes available.
              </p>
            ) : (
              releasesData.releases.map((rel) => (
                <div key={rel.version} className="space-y-2">
                  <div className="flex items-baseline gap-2">
                    <a
                      href={rel.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="font-mono text-sm font-medium hover:underline"
                    >
                      v{rel.version}
                    </a>
                    {rel.isCurrent && (
                      <Badge variant="secondary" className="text-xs">
                        Current
                      </Badge>
                    )}
                    {rel.date && (
                      <span className="text-xs text-muted-foreground">
                        {new Date(rel.date).toLocaleDateString(undefined, {
                          year: "numeric",
                          month: "short",
                          day: "numeric",
                        })}
                      </span>
                    )}
                  </div>
                  {rel.body.trim() && <ReleaseNotes body={rel.body} />}
                </div>
              ))
            )}
          </CardContent>
        </Card>
      )}
    </div>
  );
}
