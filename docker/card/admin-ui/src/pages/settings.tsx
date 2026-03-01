import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiFetch, apiPut } from "@/lib/api";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";

interface SettingsData {
  settings: { name: string; value: string }[];
  logLevel: string;
  logLevels: string[];
}

export function SettingsPage() {
  const queryClient = useQueryClient();
  const { data, isLoading } = useQuery({
    queryKey: ["settings"],
    queryFn: () => apiFetch<SettingsData>("/settings"),
  });

  const setLogLevel = useMutation({
    mutationFn: (level: string) => apiPut("/settings/log-level", { level }),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["settings"] }),
  });

  if (isLoading || !data) {
    return (
      <div className="space-y-4">
        <h1 className="text-2xl font-bold">Settings</h1>
        <div className="h-64 animate-pulse rounded-lg bg-muted" />
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Settings</h1>

      {data.settings.length === 0 ? (
        <div className="rounded-lg border border-dashed p-6 text-center text-muted-foreground">
          No settings found.
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Name</TableHead>
              <TableHead>Value</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {data.settings.map((s) => (
              <TableRow key={s.name}>
                <TableCell className="font-mono text-sm">{s.name}</TableCell>
                <TableCell>
                  {s.name === "log_level" ? (
                    <Select
                      value={data.logLevel}
                      onValueChange={(v) => setLogLevel.mutate(v)}
                      disabled={setLogLevel.isPending}
                    >
                      <SelectTrigger className="w-32">
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {data.logLevels.map((lvl) => (
                          <SelectItem key={lvl} value={lvl}>
                            {lvl}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  ) : (
                    <span
                      className={
                        s.value === "REDACTED"
                          ? "text-muted-foreground italic"
                          : "font-mono text-sm break-all"
                      }
                    >
                      {s.value || "\u2014"}
                    </span>
                  )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  );
}
