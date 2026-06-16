import { useState, type FormEvent } from "react";
import { useAuth, TotpRequiredError } from "@/hooks/use-auth";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Alert, AlertDescription } from "@/components/ui/alert";
import { Zap } from "lucide-react";

export function LoginPage() {
  const { login } = useAuth();
  const [password, setPassword] = useState("");
  const [code, setCode] = useState("");
  const [totpRequired, setTotpRequired] = useState(false);
  const [useRecovery, setUseRecovery] = useState(false);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);

  async function handleSubmit(e: FormEvent) {
    e.preventDefault();
    setError("");
    setLoading(true);
    try {
      await login(password, totpRequired ? code : undefined);
    } catch (err) {
      if (err instanceof TotpRequiredError) {
        setTotpRequired(true);
        setError("");
      } else {
        setError(err instanceof Error ? err.message : "Login failed");
      }
    } finally {
      setLoading(false);
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center p-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <a href="/" className="inline-block mx-auto">
            <Zap className="mx-auto h-8 w-8 text-primary" />
          </a>
          <CardTitle className="text-xl">
            <a href="/" className="no-underline text-foreground hover:text-primary transition-colors">
              Bolt Card Hub
            </a>
          </CardTitle>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            {error && (
              <Alert variant="destructive">
                <AlertDescription>{error}</AlertDescription>
              </Alert>
            )}
            <div className="space-y-2">
              <Label htmlFor="password">Admin Password</Label>
              <Input
                id="password"
                type="password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                autoFocus
                disabled={totpRequired}
              />
            </div>
            {totpRequired && (
              <div className="space-y-2">
                <Label htmlFor="code">
                  {useRecovery ? "Recovery Code" : "Authentication Code"}
                </Label>
                <Input
                  id="code"
                  type="text"
                  inputMode={useRecovery ? "text" : "numeric"}
                  autoComplete="one-time-code"
                  value={code}
                  onChange={(e) => setCode(e.target.value.trim())}
                  required
                  autoFocus
                  placeholder={useRecovery ? "recovery code" : "6-digit code"}
                />
                <button
                  type="button"
                  className="text-xs text-muted-foreground underline hover:text-foreground"
                  onClick={() => {
                    setUseRecovery((v) => !v);
                    setCode("");
                  }}
                >
                  {useRecovery
                    ? "Use authenticator code instead"
                    : "Use a recovery code instead"}
                </button>
              </div>
            )}
            <Button type="submit" className="w-full" disabled={loading}>
              {loading
                ? "Logging in..."
                : totpRequired
                  ? "Verify"
                  : "Login"}
            </Button>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
