import { createContext, useContext, useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { useWebSocket, type PaymentEvent } from "./use-websocket";

interface WebSocketContextValue {
  received: PaymentEvent[];
  sent: PaymentEvent[];
  status: "connecting" | "connected" | "disconnected";
}

const WebSocketContext = createContext<WebSocketContextValue | null>(null);

const invalidateKeys = [
  ["dashboard"],
  ["cards"],
  ["card"],
  ["card-txs"],
  ["phoenix"],
  ["phoenix-transactions"],
];

export function WebSocketProvider({ children }: { children: React.ReactNode }) {
  const queryClient = useQueryClient();

  const onEvent = useCallback(() => {
    for (const key of invalidateKeys) {
      queryClient.invalidateQueries({ queryKey: key });
    }
  }, [queryClient]);

  const value = useWebSocket(onEvent);

  return (
    <WebSocketContext.Provider value={value}>
      {children}
    </WebSocketContext.Provider>
  );
}

export function useWebSocketContext() {
  const ctx = useContext(WebSocketContext);
  if (!ctx) throw new Error("useWebSocketContext must be used within WebSocketProvider");
  return ctx;
}
