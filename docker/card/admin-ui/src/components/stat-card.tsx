import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { LucideIcon } from "lucide-react";
import { formatSats } from "@/lib/format";

interface StatCardProps {
  title: string;
  value: number;
  isSats?: boolean;
  format?: (value: number) => string;
  icon: LucideIcon;
}

export function StatCard({ title, value, isSats, format, icon: Icon }: StatCardProps) {
  const display = format ? format(value) : isSats ? formatSats(value) : value.toLocaleString();
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">
          {title}
        </CardTitle>
        <Icon className="h-4 w-4 text-muted-foreground" />
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold font-mono tabular-nums">
          {display}
        </div>
      </CardContent>
    </Card>
  );
}
