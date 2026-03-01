import { NavLink, Link } from "react-router-dom";
import { useAuth } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";
import { Separator } from "@/components/ui/separator";
import {
  LayoutDashboard,
  Zap,
  CreditCard,
  Settings,
  Database,
  Info,
  LogOut,
} from "lucide-react";
import { cn } from "@/lib/utils";

const navItems = [
  { to: "/", icon: LayoutDashboard, label: "Dashboard" },
  { to: "/phoenix", icon: Zap, label: "Phoenix" },
  { to: "/cards", icon: CreditCard, label: "Cards" },
  { to: "/settings", icon: Settings, label: "Settings" },
  { to: "/database", icon: Database, label: "Database" },
  { to: "/about", icon: Info, label: "About" },
];

export function Sidebar({ onNavigate }: { onNavigate?: () => void }) {
  const { logout } = useAuth();

  return (
    <div className="flex h-full flex-col gap-2">
      <Link to="/" onClick={onNavigate} className="flex items-center gap-2 px-4 py-4">
        <Zap className="h-5 w-5 text-primary" />
        <span className="text-lg font-semibold">Bolt Card Hub</span>
      </Link>
      <Separator />
      <nav className="flex-1 space-y-1 px-2 py-2">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === "/"}
            onClick={onNavigate}
            className={({ isActive }) =>
              cn(
                "flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors",
                isActive
                  ? "bg-accent text-accent-foreground"
                  : "text-muted-foreground hover:bg-accent hover:text-accent-foreground"
              )
            }
          >
            <item.icon className="h-4 w-4" />
            {item.label}
          </NavLink>
        ))}
      </nav>
      <Separator />
      <div className="px-2 py-2">
        <Button
          variant="ghost"
          className="w-full justify-start gap-3 text-muted-foreground"
          onClick={async () => {
            await logout();
            onNavigate?.();
          }}
        >
          <LogOut className="h-4 w-4" />
          Logout
        </Button>
      </div>
    </div>
  );
}
