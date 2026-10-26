import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { StatCard } from "@/components/stat-card";
import { Download, Upload, Database, HardDrive, Layers } from "lucide-react";
import { useState, type FormEvent } from "react";

interface TableCount {
  name: string;
  count: number;
}

interface DatabaseStats {
  fileSizeBytes: number;
  schemaVersion: string;
  tables: TableCount[];
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return bytes + " B";
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + " KB";
  return (bytes / (1024 * 1024)).toFixed(1) + " MB";
}

export function DatabasePage() {
  const [uploading, setUploading] = useState(false);
  const [uploadResult, setUploadResult] = useState<string | null>(null);
  const [password, setPassword] = useState("");
  const [downloading, setDownloading] = useState(false);
  const [downloadError, setDownloadError] = useState<string | null>(null);
  const [downloadPassword, setDownloadPassword] = useState("");

  const { data: stats } = useQuery({
    queryKey: ["database-stats"],
    queryFn: () => apiFetch<DatabaseStats>("/database/stats"),
  });

  const totalRows = stats?.tables?.reduce((sum, t) => sum + t.count, 0) ?? 0;

  // The download carries every card's keys, so the server re-checks the
  // admin password; POST it and save the returned file from a blob.
  async function handleDownload(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setDownloading(true);
    setDownloadError(null);
    try {
      const res = await fetch("/admin/api/database/download", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ password: downloadPassword }),
      });
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        setDownloadError(
          res.status === 401
            ? "Session expired"
            : body.error || `HTTP ${res.status}`
        );
        return;
      }
      const disposition = res.headers.get("Content-Disposition") ?? "";
      const filename =
        /filename="([^"]+)"/.exec(disposition)?.[1] ?? "cards.db";
      const url = URL.createObjectURL(await res.blob());
      const a = document.createElement("a");
      a.href = url;
      a.download = filename;
      document.body.appendChild(a);
      a.click();
      a.remove();
      URL.revokeObjectURL(url);
      setDownloadPassword("");
    } catch {
      setDownloadError("network error");
    } finally {
      setDownloading(false);
    }
  }

  async function handleImport(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const form = e.currentTarget;
    const fileInput = form.querySelector<HTMLInputElement>(
      'input[type="file"]'
    );
    if (!fileInput?.files?.[0]) return;

    setUploading(true);
    setUploadResult(null);

    const formData = new FormData();
    formData.append("database_file", fileInput.files[0]);
    formData.append("password", password);

    try {
      const res = await fetch("/admin/api/database/import", {
        method: "POST",
        body: formData,
      });
      if (res.ok) {
        setPassword("");
        setUploadResult("Database imported. Container will restart.");
      } else {
        const text = await res.text();
        setUploadResult(`Import failed: ${text}`);
      }
    } catch {
      setUploadResult("Import failed: network error");
    } finally {
      setUploading(false);
    }
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Database</h1>

      {stats && (
        <div className="grid gap-4 md:grid-cols-3">
          <StatCard title="File Size" value={stats.fileSizeBytes} icon={HardDrive} format={formatBytes} />
          <StatCard title="Total Rows" value={totalRows} icon={Database} />
          <StatCard title="Schema Version" value={Number(stats.schemaVersion)} icon={Layers} />
        </div>
      )}

      {stats?.tables && (
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">Tables</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
              {stats.tables.map((t) => (
                <div key={t.name} className="flex items-center justify-between rounded-lg bg-muted px-3 py-2">
                  <code className="text-sm">{t.name}</code>
                  <span className="font-mono text-sm tabular-nums text-muted-foreground">
                    {t.count.toLocaleString()}
                  </span>
                </div>
              ))}
            </div>
          </CardContent>
        </Card>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Export</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleDownload} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="db-download-password">Admin Password</Label>
              <Input
                id="db-download-password"
                type="password"
                autoComplete="current-password"
                value={downloadPassword}
                onChange={(e) => setDownloadPassword(e.target.value)}
                required
              />
            </div>
            <Button type="submit" variant="outline" disabled={downloading}>
              <Download className="mr-2 h-4 w-4" />
              {downloading ? "Downloading..." : "Download Database"}
            </Button>
            {downloadError && (
              <p className="text-sm text-destructive">
                Download failed: {downloadError}
              </p>
            )}
          </form>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-lg">Import</CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleImport} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="db-file">SQLite Database File (.db)</Label>
              <Input
                id="db-file"
                type="file"
                accept=".db"
                required
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="db-import-password">Admin Password</Label>
              <Input
                id="db-import-password"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>
            <Button type="submit" disabled={uploading}>
              <Upload className="mr-2 h-4 w-4" />
              {uploading ? "Importing..." : "Import Database"}
            </Button>
            {uploadResult && (
              <p className="text-sm text-muted-foreground">{uploadResult}</p>
            )}
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
