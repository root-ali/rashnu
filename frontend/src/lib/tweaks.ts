/* ============================================================
   useTweaks — single source of truth for the app's display /
   allocation settings. Persists to localStorage so choices
   survive reloads. (In the original design-tool prototype this
   talked to a host edit-mode protocol; as a standalone app it
   simply owns its own state.)
   ============================================================ */
import { useCallback, useEffect, useState } from "react";

const STORAGE_KEY = "ic_tweaks";

export type SetTweak<T> = {
  <K extends keyof T>(key: K, val: T[K]): void;
  (edits: Partial<T>): void;
};

export function useTweaks<T extends object>(defaults: T): [T, SetTweak<T>] {
  const [values, setValues] = useState<T>(() => {
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      return raw ? { ...defaults, ...(JSON.parse(raw) as Partial<T>) } : defaults;
    } catch {
      return defaults;
    }
  });

  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(values));
    } catch {
      /* ignore quota / privacy-mode errors */
    }
  }, [values]);

  const setTweak = useCallback((keyOrEdits: keyof T | Partial<T>, val?: unknown) => {
    const edits =
      typeof keyOrEdits === "object" && keyOrEdits !== null
        ? (keyOrEdits as Partial<T>)
        : ({ [keyOrEdits as keyof T]: val } as Partial<T>);
    setValues((prev) => ({ ...prev, ...edits }));
  }, []) as SetTweak<T>;

  return [values, setTweak];
}
