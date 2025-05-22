import { Terminal } from '@xterm/xterm';
import { z } from 'zod';

// WebSocketメッセージのスキーマ
export const WebSocketMessageSchema = z.object({
  type: z.string().optional(),
  message: z
    .union([
      z.object({
        result: z.string().optional(),
        error: z.string().optional(),
        pwd: z.string().optional(),
        username: z.string().optional(),
        command: z.string().optional(),
      }),
      z.string(),
      z.number(),
    ])
    .optional(),
  identifier: z.string().optional(),
});

export type WebSocketMessage = z.infer<typeof WebSocketMessageSchema>;

// WebSocketエラーのスキーマ
export const WebSocketErrorSchema = z
  .object({
    message: z.string().optional(),
  })
  .and(z.instanceof(Event));

export type WebSocketError = z.infer<typeof WebSocketErrorSchema>;

// ターミナル状態のスキーマ
export const TerminalStateSchema = z.object({
  isConnected: z.boolean(),
  isSubscribed: z.boolean(),
  isReadyForInput: z.boolean(),
  currentDir: z.string(),
  username: z.string(),
  commandHistory: z.array(z.string()),
  historyIndex: z.number(),
  isProcessingCommand: z.boolean(),
});

export type TerminalState = z.infer<typeof TerminalStateSchema>;

// ターミナル関数の戻り値の型
export interface TerminalReturn {
  term: Terminal;
  store: TerminalState;
  dispose: () => void;
  executeCommand: (command: string) => void;
}

// 定数
export const INITIAL_DIR = '/home/nonroot';
export const MAX_HISTORY_SIZE = 100;
export const WS_HOST =
  typeof window !== 'undefined' && window.location.hostname === 'localhost'
    ? 'localhost'
    : '192.168.97.1';
export const WS_URL = `ws://${WS_HOST}:8000/api/v1/cable`;

// ターミナルの設定
export const TERMINAL_OPTIONS = {
  convertEol: true,
  cursorBlink: true,
  allowProposedApi: true,
} as const;

// コマンドのバリデーションスキーマ
export const CommandSchema = z.object({
  command: z
    .string()
    .min(1, 'コマンドを入力してください')
    .max(1000, 'コマンドが長すぎます（最大1000文字）')
    .refine(
      (cmd) => !cmd.includes('\x00') && !cmd.includes('\x1b'),
      '無効な制御文字が含まれています'
    ),
  session_id: z.string().uuid(),
});

export type Command = z.infer<typeof CommandSchema>;

// WebSocket再接続の設定
export const WS_RECONNECT_CONFIG = {
  maxRetries: 5,
  initialDelay: 1000, // 1秒
  maxDelay: 30000, // 30秒
  backoffFactor: 1.5,
} as const;
