import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import type { LucideIcon } from "lucide-react";
import { formatSats } from "@/lib/format";

interface StatCardProps {
  title: string;
  value: number;
  isSats?: boolean;
  icon: LucideIcon;
}

export function StatCard({ title, value, isSats, icon: Icon }: StatCardProps) {
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
          {isSats ? formatSats(value) : value.toLocaleString()}
        </div>
      </CardContent>
    </Card>
  );
}
