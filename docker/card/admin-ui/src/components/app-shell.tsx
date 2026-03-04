import { useState } from "react";
import { Outlet } from "react-router-dom";
import { Sidebar } from "./sidebar";
import { Button } from "@/components/ui/button";
import { Sheet, SheetContent, SheetTrigger } from "@/components/ui/sheet";
import { Menu } from "lucide-react";
import { WebSocketProvider, useWebSocketContext } from "@/hooks/use-websocket-context";
import { cn } from "@/lib/utils";

function LiveIndicator() {
  const { status } = useWebSocketContext();
  return (
    <div className="flex items-center gap-1.5">
      <span
        className={cn(
          "h-2 w-2 rounded-full",
          status === "connected"
            ? "bg-[var(--success)]"
            : status === "connecting"
              ? "bg-yellow-500 animate-pulse"
              : "bg-muted-foreground"
        )}
      />
      <span className="text-xs text-muted-foreground">
        {status === "connected" ? "Live" : status === "connecting" ? "Connecting" : "Offline"}
      </span>
    </div>
  );
}

export function AppShell() {
  const [open, setOpen] = useState(false);

  return (
    <WebSocketProvider>
      <div className="flex h-screen">
        {/* Desktop sidebar */}
        <aside className="hidden w-56 border-r bg-card md:block">
          <Sidebar />
        </aside>

        {/* Mobile drawer */}
        <Sheet open={open} onOpenChange={setOpen}>
          <div className="flex flex-1 flex-col">
            <header className="flex h-14 items-center border-b px-4 md:hidden">
              <SheetTrigger asChild>
                <Button variant="ghost" size="icon">
                  <Menu className="h-5 w-5" />
                </Button>
              </SheetTrigger>
              <span className="ml-2 font-semibold">Bolt Card Hub</span>
              <div className="ml-auto">
                <LiveIndicator />
              </div>
            </header>

            <main className="relative flex-1 overflow-auto p-4 md:p-6">
              <div className="pointer-events-none absolute right-4 top-4 z-10 hidden md:block md:right-6 md:top-6">
                <LiveIndicator />
              </div>
              <Outlet />
            </main>
          </div>

          <SheetContent side="left" className="w-56 p-0">
            <Sidebar onNavigate={() => setOpen(false)} />
          </SheetContent>
        </Sheet>
      </div>
    </WebSocketProvider>
  );
}
