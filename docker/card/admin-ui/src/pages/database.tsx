import { useQuery } from "@tanstack/react-query";
import { apiFetch } from "@/lib/api";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { StatCard } from "@/components/stat-card";
import { Download, Upload, Copy, Check, KeyRound, Database, HardDrive, Layers } from "lucide-react";
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

const RESET_COMMANDS = `docker exec -it card bash
NEW_HASH=$(htpasswd -bnBC 10 "" 'YOUR_NEW_PASSWORD' | tr -d ':\\n' | sed 's/$2y/$2a/')
sqlite3 /card_data/cards.db "UPDATE settings SET value='$NEW_HASH' WHERE name='admin_password_hash';"
exit`;

export function DatabasePage() {
  const [copied, setCopied] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [uploadResult, setUploadResult] = useState<string | null>(null);

  const { data: stats } = useQuery({
    queryKey: ["database-stats"],
    queryFn: () => apiFetch<DatabaseStats>("/database/stats"),
  });

  const totalRows = stats?.tables?.reduce((sum, t) => sum + t.count, 0) ?? 0;

  function copyReset() {
    navigator.clipboard.writeText(RESET_COMMANDS);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
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

    try {
      const res = await fetch("/admin/api/database/import", {
        method: "POST",
        body: formData,
      });
      if (res.ok) {
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
          <a href="/admin/api/database/download">
            <Button variant="outline">
              <Download className="mr-2 h-4 w-4" />
              Download Database
            </Button>
          </a>
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

      <Card>
        <CardHeader>
          <div className="flex items-center justify-between">
            <CardTitle className="text-lg">
              <KeyRound className="mr-2 inline h-5 w-5" />
              Admin Password Reset
            </CardTitle>
            <Button variant="outline" size="icon" onClick={copyReset}>
              {copied ? (
                <Check className="h-4 w-4 text-[var(--success)]" />
              ) : (
                <Copy className="h-4 w-4" />
              )}
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          <pre className="overflow-x-auto rounded-lg bg-muted p-4 text-xs font-mono">
            {RESET_COMMANDS}
          </pre>
        </CardContent>
      </Card>
    </div>
  );
}
