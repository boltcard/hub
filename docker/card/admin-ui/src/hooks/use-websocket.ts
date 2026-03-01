import { useState, useEffect, useRef, useCallback } from "react";

export interface PaymentEvent {
  type: string;
  amountSat: number;
  paymentHash: string;
  timestamp: number;
}

type Status = "connecting" | "connected" | "disconnected";

export function useWebSocket(onEvent?: () => void) {
  const [received, setReceived] = useState<PaymentEvent[]>([]);
  const [sent, setSent] = useState<PaymentEvent[]>([]);
  const [status, setStatus] = useState<Status>("connecting");
  const wsRef = useRef<WebSocket | null>(null);
  const pingRef = useRef<ReturnType<typeof setInterval> | undefined>(undefined);
  const onEventRef = useRef(onEvent);
  onEventRef.current = onEvent;

  const connect = useCallback(() => {
    const proto = location.protocol === "https:" ? "wss:" : "ws:";
    const ws = new WebSocket(`${proto}//${location.host}/websocket`);
    wsRef.current = ws;
    setStatus("connecting");

    ws.onopen = () => {
      setStatus("connected");
      pingRef.current = setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) ws.send("ping");
      }, 30_000);
    };

    ws.onmessage = (e) => {
      if (e.data === "pong") return;
      try {
        const data = JSON.parse(e.data);
        if (data.type === "payment_received") {
          setReceived((prev) => [data as PaymentEvent, ...prev]);
        } else if (data.type === "payment_sent") {
          setSent((prev) => [data as PaymentEvent, ...prev]);
        }
        onEventRef.current?.();
      } catch {
        // ignore non-JSON messages
      }
    };

    ws.onclose = () => {
      clearInterval(pingRef.current);
      setStatus("disconnected");
    };

    ws.onerror = () => {
      ws.close();
    };
  }, []);

  useEffect(() => {
    let alive = true;
    let reconnectTimer: ReturnType<typeof setTimeout> | undefined;

    function connectWithReconnect() {
      if (!alive) return;
      connect();

      const ws = wsRef.current!;
      const origOnclose = ws.onclose;
      ws.onclose = (ev) => {
        if (origOnclose) (origOnclose as (ev: CloseEvent) => void)(ev);
        if (alive) reconnectTimer = setTimeout(connectWithReconnect, 5_000);
      };
    }

    connectWithReconnect();

    return () => {
      alive = false;
      clearInterval(pingRef.current);
      clearTimeout(reconnectTimer);
      wsRef.current?.close();
    };
  }, [connect]);

  return { received, sent, status };
}
