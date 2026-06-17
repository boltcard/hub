import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { apiFetch, apiPost } from "@/lib/api";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Badge } from "@/components/ui/badge";
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog";

interface TwoFaStatus {
  enabled: boolean;
  recoveryCodesRemaining: number;
}

interface SetupData {
  secret: string;
  otpauthUri: string;
  qrPng: string;
}

export function TwoFactorCard() {
  const queryClient = useQueryClient();
  const { data } = useQuery({
    queryKey: ["2fa-status"],
    queryFn: () => apiFetch<TwoFaStatus>("/auth/2fa/status"),
  });

  const [setup, setSetup] = useState<SetupData | null>(null);
  const [code, setCode] = useState("");
  const [recoveryCodes, setRecoveryCodes] = useState<string[] | null>(null);
  const [disablePassword, setDisablePassword] = useState("");
  const [disableCode, setDisableCode] = useState("");
  const [disableUseRecovery, setDisableUseRecovery] = useState(false);
  const [showDisable, setShowDisable] = useState(false);

  const resetDisable = () => {
    setDisablePassword("");
    setDisableCode("");
    setDisableUseRecovery(false);
  };

  const invalidate = () =>
    queryClient.invalidateQueries({ queryKey: ["2fa-status"] });

  const startSetup = useMutation({
    mutationFn: () => apiPost<SetupData>("/auth/2fa/setup"),
    onSuccess: (d) => {
      setSetup(d);
      setCode("");
    },
    onError: (err) => toast.error(err.message),
  });

  const enable = useMutation({
    mutationFn: () => apiPost<{ recoveryCodes: string[] }>("/auth/2fa/enable", { code }),
    onSuccess: (d) => {
      setSetup(null);
      setRecoveryCodes(d.recoveryCodes);
      invalidate();
      toast.success("Two-factor authentication enabled");
    },
    onError: (err) => toast.error(err.message),
  });

  const disable = useMutation({
    mutationFn: () =>
      apiPost("/auth/2fa/disable", {
        password: disablePassword,
        code: disableCode,
      }),
    onSuccess: () => {
      setShowDisable(false);
      resetDisable();
      invalidate();
      toast.success("Two-factor authentication disabled");
    },
    onError: (err) => toast.error(err.message),
  });

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          Two-Factor Authentication
          {data?.enabled ? (
            <Badge>Enabled</Badge>
          ) : (
            <Badge variant="secondary">Disabled</Badge>
          )}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {data?.enabled ? (
          <>
            <p className="text-sm text-muted-foreground">
              Login requires a code from your authenticator app.
              {" "}Recovery codes remaining: {data.recoveryCodesRemaining}.
            </p>
            <Button variant="destructive" onClick={() => setShowDisable(true)}>
              Disable 2FA
            </Button>
          </>
        ) : (
          <>
            <p className="text-sm text-muted-foreground">
              Add a second factor (TOTP) to admin login using an authenticator app.
            </p>
            <Button
              onClick={() => startSetup.mutate()}
              disabled={startSetup.isPending}
            >
              {startSetup.isPending ? "Preparing..." : "Enable 2FA"}
            </Button>
          </>
        )}
      </CardContent>

      {/* Enrollment dialog: QR + confirm code */}
      <Dialog open={!!setup} onOpenChange={(o) => !o && setSetup(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Scan with your authenticator</DialogTitle>
          </DialogHeader>
          {setup && (
            <div className="space-y-4">
              <img
                src={`data:image/png;base64,${setup.qrPng}`}
                alt="TOTP QR code"
                className="mx-auto h-48 w-48"
              />
              <p className="text-xs text-muted-foreground break-all">
                Manual key: <span className="font-mono">{setup.secret}</span>
              </p>
              <div className="space-y-2">
                <Label htmlFor="enable-code">Enter the 6-digit code</Label>
                <Input
                  id="enable-code"
                  inputMode="numeric"
                  value={code}
                  onChange={(e) => setCode(e.target.value.trim())}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && code.length > 0 && !enable.isPending) {
                      e.preventDefault();
                      enable.mutate();
                    }
                  }}
                  placeholder="123456"
                  autoFocus
                />
              </div>
            </div>
          )}
          <DialogFooter>
            <Button
              onClick={() => enable.mutate()}
              disabled={enable.isPending || code.length === 0}
            >
              {enable.isPending ? "Verifying..." : "Confirm"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Recovery codes shown once */}
      <Dialog open={!!recoveryCodes} onOpenChange={(o) => !o && setRecoveryCodes(null)}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Save your recovery codes</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            Store these somewhere safe. Each can be used once if you lose your
            authenticator. They will not be shown again.
          </p>
          <div className="grid grid-cols-2 gap-2 font-mono text-sm">
            {recoveryCodes?.map((c) => (
              <span key={c} className="rounded bg-muted px-2 py-1">{c}</span>
            ))}
          </div>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => {
                navigator.clipboard?.writeText((recoveryCodes ?? []).join("\n"));
                toast.success("Copied");
              }}
            >
              Copy
            </Button>
            <Button onClick={() => setRecoveryCodes(null)}>Done</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* Disable confirmation */}
      <Dialog
        open={showDisable}
        onOpenChange={(o) => {
          setShowDisable(o);
          if (!o) resetDisable();
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Disable two-factor authentication</DialogTitle>
          </DialogHeader>
          <p className="text-sm text-muted-foreground">
            Turning 2FA off requires your password and a current code, to prove
            you still hold the second factor.
          </p>
          <div className="space-y-2">
            <Label htmlFor="disable-pw">Admin password</Label>
            <Input
              id="disable-pw"
              type="password"
              value={disablePassword}
              onChange={(e) => setDisablePassword(e.target.value)}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="disable-code">
              {disableUseRecovery ? "Recovery code" : "Authenticator code"}
            </Label>
            <Input
              id="disable-code"
              inputMode={disableUseRecovery ? "text" : "numeric"}
              autoComplete="one-time-code"
              value={disableCode}
              onChange={(e) => setDisableCode(e.target.value.trim())}
              onKeyDown={(e) => {
                if (
                  e.key === "Enter" &&
                  disablePassword.length > 0 &&
                  disableCode.length > 0 &&
                  !disable.isPending
                ) {
                  e.preventDefault();
                  disable.mutate();
                }
              }}
              placeholder={disableUseRecovery ? "recovery code" : "6-digit code"}
            />
            <button
              type="button"
              className="text-xs text-muted-foreground underline hover:text-foreground"
              onClick={() => {
                setDisableUseRecovery((v) => !v);
                setDisableCode("");
              }}
            >
              {disableUseRecovery
                ? "Use authenticator code instead"
                : "Use a recovery code instead"}
            </button>
          </div>
          <DialogFooter>
            <Button
              variant="destructive"
              onClick={() => disable.mutate()}
              disabled={
                disable.isPending ||
                disablePassword.length === 0 ||
                disableCode.length === 0
              }
            >
              {disable.isPending ? "Disabling..." : "Disable"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </Card>
  );
}
