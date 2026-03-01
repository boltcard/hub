import { Link } from "react-router-dom";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { CheckCircle2, Circle, ArrowRight } from "lucide-react";

interface Step {
  label: string;
  hint: string;
  done: boolean;
  link: string;
}

interface OnboardingChecklistProps {
  phoenixConnected: boolean;
  phoenixBalance: number;
  hasCards: boolean;
}

export function OnboardingChecklist({
  phoenixConnected,
  phoenixBalance,
  hasCards,
}: OnboardingChecklistProps) {
  const steps: Step[] = [
    {
      label: "Phoenix connected",
      hint: "Check that phoenixd is running and reachable.",
      done: phoenixConnected,
      link: "/phoenix",
    },
    {
      label: "Channel open",
      hint: "A Lightning channel is needed to send and receive payments.",
      done: phoenixBalance > 0,
      link: "/phoenix",
    },
    {
      label: "Sats loaded",
      hint: "Send sats to the Bolt 12 offer to fund the hub.",
      done: phoenixBalance > 0,
      link: "/phoenix",
    },
    {
      label: "First card programmed",
      hint: "Use the Bolt Card app to program an NFC card.",
      done: hasCards,
      link: "/cards",
    },
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-lg">Getting Started</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        {steps.map((step) => (
          <div key={step.label} className="flex items-start gap-3">
            {step.done ? (
              <CheckCircle2 className="mt-0.5 h-5 w-5 shrink-0 text-[var(--success)]" />
            ) : (
              <Circle className="mt-0.5 h-5 w-5 shrink-0 text-muted-foreground" />
            )}
            <div className="flex-1">
              <p
                className={
                  step.done ? "text-sm line-through text-muted-foreground" : "text-sm font-medium"
                }
              >
                {step.label}
              </p>
              {!step.done && (
                <Link
                  to={step.link}
                  className="inline-flex items-center gap-1 text-xs text-primary hover:underline"
                >
                  {step.hint}
                  <ArrowRight className="h-3 w-3" />
                </Link>
              )}
            </div>
          </div>
        ))}
      </CardContent>
    </Card>
  );
}
