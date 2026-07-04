import { StrictMode, useCallback, useEffect, useMemo, useRef, useState, type CSSProperties, type KeyboardEvent } from "react";
import { createRoot } from "react-dom/client";
import {
  BookOpen,
  ChevronLeft,
  ChevronRight,
  Check,
  CircleAlert,
  Download,
  FileDown,
  FileUp,
  Keyboard,
  Palette,
  Plus,
  RotateCcw,
  Save,
  SlidersHorizontal,
  Sparkles,
  Languages,
  RefreshCw,
  Gauge,
  Target,
  Trash2,
} from "lucide-react";
import "./styles.css";

type Skin = {
  fontFamily: string;
  fontSize: number;
  accent: string;
  surface: string;
  text: string;
  mutedText: string;
  border: string;
  highlightText: string;
  theme: string;
};

type Config = {
  maxCandidates: number;
  candidatePageSize: number;
  candidateLayout: "horizontal" | "vertical" | "auto";
  showCandidateComments: boolean;
  fuzzyInitials: string[];
  doublePinyin: boolean;
  doublePinyinScheme: "xiaohe" | "microsoft";
  language: string;
  mode: "zh" | "en";
  punctuation: "full" | "half";
  skin: Skin;
  update: UpdateConfig;
};

type UpdateConfig = {
  channel: string;
  manifestUrls: string[];
  mirrorBaseUrls: string[] | null;
  autoCheck: boolean;
  autoApply: boolean;
  installedVersion: string;
};

type UpdateCheckResult = {
  currentVersion: string;
  latestVersion: string;
  updateAvailable: boolean;
  manifestUrl?: string;
};

type UpdateApplyResult = {
  ok: boolean;
  manifestUrl: string;
  version: string;
  applied: string[];
};

type WordbookResponse = {
  userScores: Record<string, number>;
  count: number;
  updatedAt: string;
};

type WordbookEntry = {
  key: string;
  reading: string;
  text: string;
  score: number;
};

type PhraseEntry = {
  reading: string;
  text: string;
  kind?: string;
  source?: string;
  comment?: string;
  weight?: number;
};

type PhraseResponse = {
  phrases: PhraseEntry[];
  entries?: PhraseEntry[];
  count: number;
  updatedAt: string;
};

type RejectResponse = {
  rejects?: PhraseEntry[];
  entries?: PhraseEntry[];
  count: number;
  updatedAt: string;
};

type CatalogResponse = {
  kind: string;
  query?: string;
  count: number;
  entries: PhraseEntry[];
  updatedAt: string;
};

type Candidate = {
  text: string;
  reading: string;
  kind?: string;
  source?: string;
  comment?: string;
  weight: number;
  userScore: number;
};

type CandidatePageItem = Candidate & {
  index: number;
  displayIndex: number;
  score: number;
};

type EngineState = {
  buffer: string;
  candidates: Candidate[] | null;
  committed?: string;
  updatedAt: string;
};

type CandidateActionResult = {
  ok: boolean;
  action: string;
  start: number;
  limit: number;
  total: number;
  committed?: string;
  rejected?: PhraseEntry;
  state: EngineState;
  candidates: {
    ok: boolean;
    start: number;
    limit: number;
    total: number;
    items: CandidatePageItem[];
    updatedAt: string;
  };
  updatedAt: string;
};

type SkinPreset = {
  id: string;
  name: string;
  skin: Skin;
};

type TypingPrompt = {
  id: string;
  name: string;
  text: string;
};

type TypingMetrics = {
  elapsedMs: number;
  wpm: number;
  cpm: number;
  keysPerSecond: number;
  avgLatencyMs: number;
  p95LatencyMs: number;
  burstKeysPerSecond: number;
  accuracy: number;
  keyCount: number;
  changes: number;
  errors: number;
  composing: boolean;
  compositions: number;
};

type TypingProbe = {
  input: string;
  candidates: Candidate[];
  status: "idle" | "ready" | "offline";
  updatedAt: string;
};

const apiBase = "http://127.0.0.1:23333";

const defaultConfig: Config = {
  maxCandidates: 42,
  candidatePageSize: 7,
  candidateLayout: "horizontal",
  showCandidateComments: true,
  fuzzyInitials: ["zh=z", "ch=c", "sh=s"],
  doublePinyin: false,
  doublePinyinScheme: "xiaohe",
  language: "zh-CN",
  mode: "zh",
  punctuation: "full",
  skin: {
    fontFamily: "Microsoft YaHei UI",
    fontSize: 15,
    accent: "#2563eb",
    surface: "#ffffff",
    text: "#111827",
    mutedText: "#64748b",
    border: "#d1d5db",
    highlightText: "#ffffff",
    theme: "system",
  },
  update: {
    channel: "stable",
    manifestUrls: ["https://github.com/neko233-com/shurufa233/releases/latest/download/dictionary-manifest.json"],
    mirrorBaseUrls: [],
    autoCheck: true,
    autoApply: false,
    installedVersion: "builtin",
  },
};

const skinPresets: SkinPreset[] = [
  {
    id: "wechat-clean",
    name: "清透白",
    skin: {
      ...defaultConfig.skin,
      accent: "#16a34a",
      surface: "#ffffff",
      text: "#111827",
      mutedText: "#64748b",
      border: "#d7dee8",
      highlightText: "#ffffff",
      theme: "wechat-clean",
    },
  },
  {
    id: "ink",
    name: "墨色",
    skin: {
      ...defaultConfig.skin,
      accent: "#38bdf8",
      surface: "#111827",
      text: "#f8fafc",
      mutedText: "#94a3b8",
      border: "#334155",
      highlightText: "#ffffff",
      theme: "ink",
    },
  },
  {
    id: "bamboo",
    name: "竹青",
    skin: {
      ...defaultConfig.skin,
      accent: "#0f766e",
      surface: "#f8fffc",
      text: "#10201d",
      mutedText: "#52756f",
      border: "#b9d8d0",
      highlightText: "#ffffff",
      theme: "bamboo",
    },
  },
  {
    id: "berry",
    name: "莓果",
    skin: {
      ...defaultConfig.skin,
      accent: "#db2777",
      surface: "#fff7fb",
      text: "#26111c",
      mutedText: "#8b5d72",
      border: "#f0bfd4",
      highlightText: "#ffffff",
      theme: "berry",
    },
  },
];

const typingPrompts: TypingPrompt[] = [
  {
    id: "pinyin-rhythm",
    name: "拼音节奏",
    text: "shurufa233 ganjing qingkuai xiang weixin shurufa yiyang shunshou",
  },
  {
    id: "zh-commit",
    name: "中文上屏",
    text: "输入法要干净、顺手、低延迟，候选窗清楚好看。",
  },
  {
    id: "esports",
    name: "电竞短句",
    text: "flash mid peek left hold angle reset tap strafe confirm",
  },
  {
    id: "symbols",
    name: "颜表情候选",
    text: "zan kaixin wuyu shengqi aixin shengluehao",
  },
  {
    id: "dynamic",
    name: "动态候选",
    text: "rq sj xq dt ts date time week datetime timestamp",
  },
];

const defaultTypingMetrics: TypingMetrics = {
  elapsedMs: 0,
  wpm: 0,
  cpm: 0,
  keysPerSecond: 0,
  avgLatencyMs: 0,
  p95LatencyMs: 0,
  burstKeysPerSecond: 0,
  accuracy: 100,
  keyCount: 0,
  changes: 0,
  errors: 0,
  composing: false,
  compositions: 0,
};

function createTypingStats() {
  return {
    startedAt: 0,
    lastKeyAt: 0,
    keyCount: 0,
    changes: 0,
    latencyTotalMs: 0,
    latencySamples: 0,
    latencyWindow: [] as number[],
    keyWindow: [] as number[],
    burstKeysPerSecond: 0,
    recentKeys: [] as string[],
    compositions: 0,
    composing: false,
  };
}

function percentile(values: number[], ratio: number) {
  if (values.length === 0) return 0;
  const sorted = [...values].sort((left, right) => left - right);
  const index = Math.min(sorted.length - 1, Math.max(0, Math.ceil(sorted.length * ratio) - 1));
  return sorted[index];
}

function extractTypingProbe(input: string) {
  const match = input.match(/\/?[a-zA-Z]{1,32}$/);
  return match ? match[0].toLowerCase() : "";
}

function countTypingErrors(input: string, target: string) {
  let errors = Math.max(0, input.length - target.length);
  const compareLength = Math.min(input.length, target.length);
  for (let index = 0; index < compareLength; index += 1) {
    if (input[index] !== target[index]) {
      errors += 1;
    }
  }
  return errors;
}

function wordbookEntries(scores: Record<string, number>): WordbookEntry[] {
  return Object.entries(scores)
    .map(([key, score]) => {
      const splitAt = key.indexOf("|");
      return {
        key,
        reading: splitAt >= 0 ? key.slice(0, splitAt) : key,
        text: splitAt >= 0 ? key.slice(splitAt + 1) : "",
        score,
      };
    })
    .sort((left, right) => right.score - left.score || left.key.localeCompare(right.key));
}

function phraseEntries(response: PhraseResponse): PhraseEntry[] {
  return [...(response.phrases ?? response.entries ?? [])].sort(
    (left, right) => left.reading.localeCompare(right.reading) || left.text.localeCompare(right.text),
  );
}

function rejectEntries(response: RejectResponse): PhraseEntry[] {
  return [...(response.rejects ?? response.entries ?? [])].sort(
    (left, right) => left.reading.localeCompare(right.reading) || left.text.localeCompare(right.text),
  );
}

function hydrateConfig(config: Config): Config {
  return {
    ...defaultConfig,
    ...config,
    candidatePageSize: Math.min(9, Math.max(3, config.candidatePageSize || defaultConfig.candidatePageSize)),
    candidateLayout: normalizeCandidateLayout(config.candidateLayout),
    showCandidateComments: config.showCandidateComments ?? defaultConfig.showCandidateComments,
    skin: {
      ...defaultConfig.skin,
      ...config.skin,
    },
    update: {
      ...defaultConfig.update,
      ...config.update,
      manifestUrls: config.update?.manifestUrls ?? defaultConfig.update.manifestUrls,
      mirrorBaseUrls: config.update?.mirrorBaseUrls ?? defaultConfig.update.mirrorBaseUrls,
    },
  };
}

function normalizeCandidateLayout(layout?: string): Config["candidateLayout"] {
  if (layout === "vertical" || layout === "auto") return layout;
  return "horizontal";
}

function App() {
  const [config, setConfig] = useState<Config>(defaultConfig);
  const [preview, setPreview] = useState("nihao");
  const [previewPageStart, setPreviewPageStart] = useState(0);
  const [previewCommitted, setPreviewCommitted] = useState("");
  const [state, setState] = useState<EngineState | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "offline" | "saved">("loading");
  const [updateText, setUpdateText] = useState("未检查");
  const [updateBusy, setUpdateBusy] = useState<"idle" | "checking" | "applying">("idle");
  const [wordbook, setWordbook] = useState<WordbookEntry[]>([]);
  const [wordbookDraft, setWordbookDraft] = useState("{}");
  const [wordbookText, setWordbookText] = useState("未读取");
  const [phrases, setPhrases] = useState<PhraseEntry[]>([]);
  const [phraseDraft, setPhraseDraft] = useState(`{
  "entries": [
    { "reading": "msd", "text": "马上到！", "weight": 60000 }
  ]
}`);
  const [phraseText, setPhraseText] = useState("未读取");
  const [phraseReading, setPhraseReading] = useState("msd");
  const [phraseValue, setPhraseValue] = useState("马上到！");
  const [phraseWeight, setPhraseWeight] = useState(60000);
  const [rejects, setRejects] = useState<PhraseEntry[]>([]);
  const [rejectDraft, setRejectDraft] = useState("{\n  \"entries\": []\n}");
  const [rejectText, setRejectText] = useState("未读取");
  const [catalogKind, setCatalogKind] = useState("all");
  const [catalogQuery, setCatalogQuery] = useState("");
  const [catalogEntries, setCatalogEntries] = useState<PhraseEntry[]>([]);
  const [catalogText, setCatalogText] = useState("未读取");
  const [error, setError] = useState("");
  const [typingPromptId, setTypingPromptId] = useState(typingPrompts[0].id);
  const [typingText, setTypingText] = useState("");
  const [typingMetrics, setTypingMetrics] = useState<TypingMetrics>(defaultTypingMetrics);
  const [typingProbe, setTypingProbe] = useState<TypingProbe>({
    input: "",
    candidates: [],
    status: "idle",
    updatedAt: "",
  });
  const typingStatsRef = useRef(createTypingStats());

  useEffect(() => {
    void loadConfig();
    void loadWordbook();
    void loadPhrases();
    void loadRejects();
    void loadCatalog();
  }, []);

  useEffect(() => {
    const timeout = window.setTimeout(() => void loadCatalog(), 160);
    return () => window.clearTimeout(timeout);
  }, [catalogKind, catalogQuery]);

  useEffect(() => {
    const timeout = window.setTimeout(() => void runPreview(preview), 120);
    return () => window.clearTimeout(timeout);
  }, [preview]);

  const candidateCount = state?.candidates?.length ?? 0;
  const candidatePageSize = Math.min(9, Math.max(3, config.candidatePageSize || defaultConfig.candidatePageSize));
  const candidateLayout = normalizeCandidateLayout(config.candidateLayout);
  const showCandidateComments = config.showCandidateComments ?? defaultConfig.showCandidateComments;
  const normalizedPreviewPageStart =
    candidateCount > 0 ? Math.min(previewPageStart, Math.floor((candidateCount - 1) / candidatePageSize) * candidatePageSize) : 0;
  const typingPrompt = useMemo(
    () => typingPrompts.find((prompt) => prompt.id === typingPromptId) ?? typingPrompts[0],
    [typingPromptId],
  );
  const typingProgress = useMemo(
    () =>
      typingPrompt.text.length > 0
        ? Math.min(100, (typingText.length / typingPrompt.text.length) * 100)
        : 0,
    [typingPrompt.text, typingText.length],
  );
  const accentStyle = useMemo(() => ({ "--accent": config.skin.accent }) as CSSProperties, [config.skin.accent]);
  const candidateBarStyle = useMemo(
    () =>
      ({
        fontFamily: config.skin.fontFamily,
        fontSize: config.skin.fontSize,
        "--candidate-accent": config.skin.accent,
        "--candidate-surface": config.skin.surface,
        "--candidate-text": config.skin.text,
        "--candidate-muted": config.skin.mutedText,
        "--candidate-border": config.skin.border,
        "--candidate-highlight": config.skin.highlightText,
      }) as CSSProperties,
    [config.skin],
  );
  const previewCandidates = (state?.candidates ?? []).slice(normalizedPreviewPageStart, normalizedPreviewPageStart + candidatePageSize);
  const typingProbeCandidates = typingProbe.candidates.slice(0, candidatePageSize);
  const typingProbeKinds = useMemo(() => {
    const kinds = new Set(typingProbeCandidates.map((candidate) => candidate.kind).filter(Boolean));
    return Array.from(kinds)
      .map((kind) => kindLabel(kind))
      .filter(Boolean);
  }, [typingProbeCandidates]);

  const refreshTypingMetrics = useCallback(
    (input: string) => {
      const stats = typingStatsRef.current;
      const now = performance.now();
      const elapsedMs = stats.startedAt > 0 ? Math.max(1, now - stats.startedAt) : 0;
      const elapsedMinutes = elapsedMs > 0 ? elapsedMs / 60000 : 0;
      const errors = countTypingErrors(input, typingPrompt.text);
      const accuracy = input.length > 0 ? Math.max(0, ((input.length - errors) / input.length) * 100) : 100;
      const activityEvents = Math.max(stats.keyCount, stats.changes);
      const p95LatencyMs = percentile(stats.latencyWindow, 0.95);
      setTypingMetrics({
        elapsedMs,
        wpm: elapsedMinutes > 0 ? input.length / 5 / elapsedMinutes : 0,
        cpm: elapsedMinutes > 0 ? input.length / elapsedMinutes : 0,
        keysPerSecond: elapsedMs > 0 ? activityEvents / (elapsedMs / 1000) : 0,
        avgLatencyMs: stats.latencySamples > 0 ? stats.latencyTotalMs / stats.latencySamples : 0,
        p95LatencyMs,
        burstKeysPerSecond: stats.burstKeysPerSecond,
        accuracy,
        keyCount: stats.keyCount,
        changes: stats.changes,
        errors,
        composing: stats.composing,
        compositions: stats.compositions,
      });
    },
    [typingPrompt.text],
  );

  const resetTypingLab = useCallback(() => {
    typingStatsRef.current = createTypingStats();
    setTypingText("");
    setTypingMetrics(defaultTypingMetrics);
    setTypingProbe({ input: "", candidates: [], status: "idle", updatedAt: "" });
  }, []);

  const handleTypingKeyDown = useCallback((event: KeyboardEvent<HTMLTextAreaElement>) => {
    const stats = typingStatsRef.current;
    const now = performance.now();
    if (stats.startedAt === 0) {
      stats.startedAt = now;
    }
    stats.keyCount += 1;
    stats.lastKeyAt = now;
    stats.keyWindow = [...stats.keyWindow.filter((time) => now - time <= 1000), now];
    stats.burstKeysPerSecond = Math.max(stats.burstKeysPerSecond, stats.keyWindow.length);
    if (event.key.length === 1 || event.key === "Backspace" || event.key === "Enter" || event.key === " ") {
      stats.recentKeys = [...stats.recentKeys.slice(-11), event.key === " " ? "Space" : event.key];
    }
  }, []);

  const handleTypingChange = useCallback(
    (value: string) => {
      const stats = typingStatsRef.current;
      if (stats.startedAt === 0) {
        stats.startedAt = performance.now();
      }
      stats.changes += 1;
      if (stats.lastKeyAt > 0) {
        const latency = performance.now() - stats.lastKeyAt;
        if (latency >= 0 && latency < 1000) {
          stats.latencyTotalMs += latency;
          stats.latencySamples += 1;
          stats.latencyWindow = [...stats.latencyWindow.slice(-255), latency];
        }
      }
      setTypingText(value);
      refreshTypingMetrics(value);
    },
    [refreshTypingMetrics],
  );

  const handleCompositionStart = useCallback(() => {
    const stats = typingStatsRef.current;
    stats.composing = true;
    stats.compositions += 1;
    refreshTypingMetrics(typingText);
  }, [refreshTypingMetrics, typingText]);

  const handleCompositionEnd = useCallback(() => {
    typingStatsRef.current.composing = false;
    refreshTypingMetrics(typingText);
  }, [refreshTypingMetrics, typingText]);

  useEffect(() => {
    const interval = window.setInterval(() => {
      refreshTypingMetrics(typingText);
    }, 120);
    return () => window.clearInterval(interval);
  }, [refreshTypingMetrics, typingText]);

  useEffect(() => {
    const probe = extractTypingProbe(typingText);
    if (!probe) {
      setTypingProbe({ input: "", candidates: [], status: "idle", updatedAt: "" });
      return;
    }
    const timeout = window.setTimeout(() => {
      void runTypingProbe(probe);
    }, 90);
    return () => window.clearTimeout(timeout);
  }, [typingText]);

  async function loadConfig() {
    try {
      const res = await fetch(`${apiBase}/config`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(hydrateConfig(await res.json()));
      setStatus("ready");
      setError("");
    } catch (err) {
      setStatus("offline");
      setError(err instanceof Error ? err.message : "daemon offline");
    }
  }

  async function runPreview(input: string) {
    try {
      const res = await fetch(`${apiBase}/engine/preview`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ input }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setState(await res.json());
      setPreviewPageStart(0);
      setPreviewCommitted("");
      if (status === "offline") setStatus("ready");
      setError("");
    } catch (err) {
      setStatus("offline");
      setError(err instanceof Error ? err.message : "daemon offline");
    }
  }

  async function runCandidateAction(action: string, displayIndex?: number) {
    try {
      const res = await fetch(`${apiBase}/ime/candidate-action`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          action,
          start: normalizedPreviewPageStart,
          limit: candidatePageSize,
          displayIndex,
        }),
      });
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as CandidateActionResult;
      setState(data.state);
      setPreviewPageStart(data.start);
      if (data.committed) {
        setPreviewCommitted(data.committed);
        setStatus("ready");
      }
      if (data.rejected) {
        setRejectText(`已屏蔽 ${data.rejected.reading}|${data.rejected.text}`);
        void loadRejects();
      }
      setError("");
    } catch (err) {
      setStatus("offline");
      setError(err instanceof Error ? err.message : "candidate action failed");
    }
  }

  async function runTypingProbe(input: string) {
    try {
      const res = await fetch(`${apiBase}/engine/preview`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ input }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = (await res.json()) as EngineState;
      setTypingProbe({
        input,
        candidates: data.candidates ?? [],
        status: "ready",
        updatedAt: data.updatedAt,
      });
      if (status === "offline") setStatus("ready");
    } catch {
      setTypingProbe({
        input,
        candidates: [],
        status: "offline",
        updatedAt: new Date().toISOString(),
      });
    }
  }

  async function saveConfig() {
    try {
      const res = await fetch(`${apiBase}/config`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(config),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      setConfig(hydrateConfig(await res.json()));
      setStatus("saved");
      window.setTimeout(() => setStatus("ready"), 1000);
      setError("");
    } catch (err) {
      setStatus("offline");
      setError(err instanceof Error ? err.message : "save failed");
    }
  }

  async function checkUpdates() {
    setUpdateBusy("checking");
    try {
      const res = await fetch(`${apiBase}/updates/check`);
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as UpdateCheckResult;
      setUpdateText(data.updateAvailable ? `发现 ${data.latestVersion}` : `已是最新 ${data.currentVersion}`);
      setError("");
    } catch (err) {
      setUpdateText("检查失败");
      setError(err instanceof Error ? err.message : "update check failed");
    } finally {
      setUpdateBusy("idle");
    }
  }

  async function applyUpdates() {
    setUpdateBusy("applying");
    try {
      const res = await fetch(`${apiBase}/updates/apply`, { method: "POST" });
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as UpdateApplyResult;
      const applied = data.applied.length > 0 ? data.applied.join(", ") : "无需更新";
      setUpdateText(`已应用 ${data.version} · ${applied}`);
      setConfig({
        ...config,
        update: {
          ...config.update,
          installedVersion: data.version,
        },
      });
      void runPreview(preview);
      setError("");
    } catch (err) {
      setUpdateText("更新失败");
      setError(err instanceof Error ? err.message : "update apply failed");
    } finally {
      setUpdateBusy("idle");
    }
  }

  async function loadWordbook() {
    try {
      const res = await fetch(`${apiBase}/wordbook`);
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as WordbookResponse;
      const scores = data.userScores ?? {};
      setWordbook(wordbookEntries(scores));
      setWordbookDraft(JSON.stringify(scores, null, 2));
      setWordbookText(`${data.count ?? Object.keys(scores).length} 条用户词`);
      setError("");
    } catch (err) {
      setWordbookText("读取失败");
      setError(err instanceof Error ? err.message : "wordbook load failed");
    }
  }

  async function importWordbook(merge: boolean) {
    try {
      const parsed = JSON.parse(wordbookDraft) as Record<string, number> | { userScores?: Record<string, number>; scores?: Record<string, number> };
      const userScores = "userScores" in parsed || "scores" in parsed ? parsed.userScores ?? parsed.scores ?? {} : parsed;
      const res = await fetch(`${apiBase}/wordbook`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ userScores, merge }),
      });
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as WordbookResponse;
      const scores = data.userScores ?? {};
      setWordbook(wordbookEntries(scores));
      setWordbookDraft(JSON.stringify(scores, null, 2));
      setWordbookText(`已导入 ${data.count ?? Object.keys(scores).length} 条`);
      void runPreview(preview);
      setError("");
    } catch (err) {
      setWordbookText("导入失败");
      setError(err instanceof Error ? err.message : "wordbook import failed");
    }
  }

  async function deleteWordbookEntry(key: string) {
    try {
      const res = await fetch(`${apiBase}/wordbook?key=${encodeURIComponent(key)}`, { method: "DELETE" });
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as WordbookResponse;
      const scores = data.userScores ?? {};
      setWordbook(wordbookEntries(scores));
      setWordbookDraft(JSON.stringify(scores, null, 2));
      setWordbookText(`剩余 ${data.count ?? Object.keys(scores).length} 条`);
      void runPreview(preview);
      setError("");
    } catch (err) {
      setWordbookText("删除失败");
      setError(err instanceof Error ? err.message : "wordbook delete failed");
    }
  }

  async function clearWordbook() {
    if (!window.confirm("清空所有用户词学习记录？")) return;
    try {
      const res = await fetch(`${apiBase}/wordbook`, { method: "DELETE" });
      if (!res.ok) throw new Error(await res.text());
      setWordbook([]);
      setWordbookDraft("{}");
      setWordbookText("已清空");
      void runPreview(preview);
      setError("");
    } catch (err) {
      setWordbookText("清空失败");
      setError(err instanceof Error ? err.message : "wordbook clear failed");
    }
  }

  function exportWordbook() {
    const scores = Object.fromEntries(wordbook.map((entry) => [entry.key, entry.score]));
    const blob = new Blob([JSON.stringify({ userScores: scores }, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = "shurufa233-user-wordbook.json";
    anchor.click();
    URL.revokeObjectURL(url);
  }

  async function loadPhrases() {
    try {
      const res = await fetch(`${apiBase}/phrases`);
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as PhraseResponse;
      const entries = phraseEntries(data);
      setPhrases(entries);
      setPhraseDraft(JSON.stringify({ entries }, null, 2));
      setPhraseText(`${data.count ?? entries.length} 条固定短语`);
      setError("");
    } catch (err) {
      setPhraseText("读取失败");
      setError(err instanceof Error ? err.message : "phrases load failed");
    }
  }

  async function savePhraseEntries(entries: PhraseEntry[], merge: boolean, statusText: string) {
    const res = await fetch(`${apiBase}/phrases`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ entries, merge }),
    });
    if (!res.ok) throw new Error(await res.text());
    const data = (await res.json()) as PhraseResponse;
    const nextEntries = phraseEntries(data);
    setPhrases(nextEntries);
    setPhraseDraft(JSON.stringify({ entries: nextEntries }, null, 2));
    setPhraseText(`${statusText} ${data.count ?? nextEntries.length} 条`);
    setError("");
    return nextEntries;
  }

  async function addPhrase() {
    const reading = phraseReading.trim().toLowerCase();
    const text = phraseValue.trim();
    if (!reading || !text) {
      setPhraseText("需要编码和短语");
      return;
    }
    try {
      await savePhraseEntries([{ reading, text, weight: phraseWeight }], true, "已保存");
      setPreview(reading);
      void runPreview(reading);
    } catch (err) {
      setPhraseText("保存失败");
      setError(err instanceof Error ? err.message : "phrase add failed");
    }
  }

  async function importPhrases(merge: boolean) {
    try {
      const parsed = JSON.parse(phraseDraft) as PhraseEntry[] | { entries?: PhraseEntry[]; phrases?: PhraseEntry[] };
      const entries = Array.isArray(parsed) ? parsed : parsed.entries ?? parsed.phrases ?? [];
      const nextEntries = await savePhraseEntries(entries, merge, merge ? "已合并" : "已替换");
      if (nextEntries.length > 0) {
        setPreview(nextEntries[0].reading);
        void runPreview(nextEntries[0].reading);
      }
    } catch (err) {
      setPhraseText("导入失败");
      setError(err instanceof Error ? err.message : "phrase import failed");
    }
  }

  async function deletePhrase(entry: PhraseEntry) {
    try {
      const key = `${entry.reading}|${entry.text}`;
      const res = await fetch(`${apiBase}/phrases?key=${encodeURIComponent(key)}`, { method: "DELETE" });
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as PhraseResponse;
      const nextEntries = phraseEntries(data);
      setPhrases(nextEntries);
      setPhraseDraft(JSON.stringify({ entries: nextEntries }, null, 2));
      setPhraseText(`剩余 ${data.count ?? nextEntries.length} 条`);
      void runPreview(preview);
      setError("");
    } catch (err) {
      setPhraseText("删除失败");
      setError(err instanceof Error ? err.message : "phrase delete failed");
    }
  }

  async function clearPhrases() {
    if (!window.confirm("清空所有固定短语？")) return;
    try {
      const res = await fetch(`${apiBase}/phrases`, { method: "DELETE" });
      if (!res.ok) throw new Error(await res.text());
      setPhrases([]);
      setPhraseDraft("{\n  \"entries\": []\n}");
      setPhraseText("已清空");
      void runPreview(preview);
      setError("");
    } catch (err) {
      setPhraseText("清空失败");
      setError(err instanceof Error ? err.message : "phrase clear failed");
    }
  }

  function exportPhrases() {
    const blob = new Blob([JSON.stringify({ entries: phrases }, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = "shurufa233-user-phrases.json";
    anchor.click();
    URL.revokeObjectURL(url);
  }

  async function loadRejects() {
    try {
      const res = await fetch(`${apiBase}/rejects`);
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as RejectResponse;
      const entries = rejectEntries(data);
      setRejects(entries);
      setRejectDraft(JSON.stringify({ entries }, null, 2));
      setRejectText(`${data.count ?? entries.length} 条已屏蔽`);
      setError("");
    } catch (err) {
      setRejectText("读取失败");
      setError(err instanceof Error ? err.message : "rejects load failed");
    }
  }

  async function saveRejectEntries(entries: PhraseEntry[], merge: boolean, statusText: string) {
    const res = await fetch(`${apiBase}/rejects`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ entries, merge }),
    });
    if (!res.ok) throw new Error(await res.text());
    const data = (await res.json()) as RejectResponse;
    const nextEntries = rejectEntries(data);
    setRejects(nextEntries);
    setRejectDraft(JSON.stringify({ entries: nextEntries }, null, 2));
    setRejectText(`${statusText} ${data.count ?? nextEntries.length} 条`);
    setError("");
    return nextEntries;
  }

  async function importRejects(merge: boolean) {
    try {
      const parsed = JSON.parse(rejectDraft) as PhraseEntry[] | { entries?: PhraseEntry[]; rejects?: PhraseEntry[] };
      const entries = Array.isArray(parsed) ? parsed : parsed.entries ?? parsed.rejects ?? [];
      await saveRejectEntries(entries, merge, merge ? "已合并" : "已替换");
      void runPreview(preview);
    } catch (err) {
      setRejectText("导入失败");
      setError(err instanceof Error ? err.message : "reject import failed");
    }
  }

  async function deleteReject(entry: PhraseEntry) {
    try {
      const key = `${entry.reading}|${entry.text}`;
      const res = await fetch(`${apiBase}/rejects?key=${encodeURIComponent(key)}`, { method: "DELETE" });
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as RejectResponse;
      const nextEntries = rejectEntries(data);
      setRejects(nextEntries);
      setRejectDraft(JSON.stringify({ entries: nextEntries }, null, 2));
      setRejectText(`剩余 ${data.count ?? nextEntries.length} 条`);
      void runPreview(preview);
      setError("");
    } catch (err) {
      setRejectText("恢复失败");
      setError(err instanceof Error ? err.message : "reject delete failed");
    }
  }

  async function clearRejects() {
    if (!window.confirm("恢复所有已屏蔽候选？")) return;
    try {
      const res = await fetch(`${apiBase}/rejects`, { method: "DELETE" });
      if (!res.ok) throw new Error(await res.text());
      setRejects([]);
      setRejectDraft("{\n  \"entries\": []\n}");
      setRejectText("已全部恢复");
      void runPreview(preview);
      setError("");
    } catch (err) {
      setRejectText("恢复失败");
      setError(err instanceof Error ? err.message : "reject clear failed");
    }
  }

  function exportRejects() {
    const blob = new Blob([JSON.stringify({ entries: rejects }, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = "shurufa233-user-rejects.json";
    anchor.click();
    URL.revokeObjectURL(url);
  }

  async function loadCatalog() {
    try {
      const query = new URLSearchParams({
        kind: catalogKind,
        limit: "80",
      });
      if (catalogQuery.trim()) {
        query.set("q", catalogQuery.trim());
      }
      const res = await fetch(`${apiBase}/catalog?${query.toString()}`);
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as CatalogResponse;
      setCatalogEntries(data.entries ?? []);
      setCatalogText(`${data.count ?? data.entries?.length ?? 0} 项资源`);
      setError("");
    } catch (err) {
      setCatalogText("读取失败");
      setError(err instanceof Error ? err.message : "catalog load failed");
    }
  }

  function previewCatalogEntry(entry: PhraseEntry) {
    const code = entry.reading ? `/${entry.reading}` : entry.text;
    setPreview(code);
    void runPreview(code);
  }

  function exportTypingReport() {
    const report = {
      generatedAt: new Date().toISOString(),
      prompt: typingPrompt,
      input: typingText,
      metrics: typingMetrics,
      probe: typingProbe,
      recentKeys: typingStatsRef.current.recentKeys,
    };
    const blob = new Blob([JSON.stringify(report, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = "shurufa233-typing-lab-report.json";
    anchor.click();
    URL.revokeObjectURL(url);
  }

  return (
    <main className="app" style={accentStyle}>
      <aside className="sidebar">
        <div className="brand">
          <Keyboard size={22} />
          <div>
            <strong>shurufa233</strong>
            <span>全局输入法控制台</span>
          </div>
        </div>
        <nav>
          <a className="active"><SlidersHorizontal size={18} /> 输入</a>
          <a><Palette size={18} /> 候选栏</a>
          <a><BookOpen size={18} /> 词库</a>
          <a><Languages size={18} /> 中英切换</a>
          <a><Gauge size={18} /> 性能</a>
          <a><Sparkles size={18} /> 更新</a>
        </nav>
      </aside>

      <section className="workspace">
        <header className="topbar">
          <div>
            <h1>设置与候选预览</h1>
            <p>本地 daemon: {statusLabel(status)}</p>
          </div>
          <div className="actions">
            <button className="iconButton" title="重新读取配置" onClick={loadConfig}>
              <RotateCcw size={18} />
            </button>
            <button className="primary" onClick={saveConfig}>
              {status === "saved" ? <Check size={18} /> : <Save size={18} />}
              保存
            </button>
          </div>
        </header>

        {status === "offline" && (
          <div className="alert">
            <CircleAlert size={18} />
            <span>daemon 未连接：先运行 `go run ./cmd/daemon`。{error}</span>
          </div>
        )}

        <div className="grid">
          <section className="panel">
            <div className="panelHeader">
              <h2>输入规则</h2>
              <span>Go 内核实时生效</span>
            </div>
            <div className="segmented">
              <button className={config.mode === "zh" ? "selected" : ""} onClick={() => setConfig({ ...config, mode: "zh" })}>
                中文
              </button>
              <button className={config.mode === "en" ? "selected" : ""} onClick={() => setConfig({ ...config, mode: "en" })}>
                English
              </button>
            </div>
            <div className="segmented">
              <button
                className={(config.punctuation ?? "full") === "full" ? "selected" : ""}
                onClick={() => setConfig({ ...config, punctuation: "full" })}
              >
                中文标点
              </button>
              <button
                className={config.punctuation === "half" ? "selected" : ""}
                onClick={() => setConfig({ ...config, punctuation: "half" })}
              >
                半角标点
              </button>
            </div>
            <label className="field">
              <span>语言词库</span>
              <select value={config.language} onChange={(event) => setConfig({ ...config, language: event.target.value })}>
                <option value="zh-CN">中文拼音</option>
                <option value="en-US">English</option>
                <option value="ja-JP">日本語</option>
                <option value="ko-KR">한국어</option>
              </select>
            </label>
            <label className="field">
              <span>候选池上限</span>
              <input
                type="number"
                min={42}
                max={99}
                value={config.maxCandidates}
                onChange={(event) => setConfig({ ...config, maxCandidates: Number(event.target.value) })}
              />
            </label>
            <label className="field">
              <span>每页候选数</span>
              <input
                type="number"
                min={3}
                max={9}
                value={candidatePageSize}
                onChange={(event) => setConfig({ ...config, candidatePageSize: Number(event.target.value) })}
              />
            </label>
            <div className="segmented">
              <button
                className={candidateLayout === "horizontal" ? "selected" : ""}
                onClick={() => setConfig({ ...config, candidateLayout: "horizontal" })}
              >
                横排候选
              </button>
              <button
                className={candidateLayout === "vertical" ? "selected" : ""}
                onClick={() => setConfig({ ...config, candidateLayout: "vertical" })}
              >
                竖排候选
              </button>
            </div>
            <label className="toggle">
              <input
                type="checkbox"
                checked={showCandidateComments}
                onChange={(event) => setConfig({ ...config, showCandidateComments: event.target.checked })}
              />
              <span>显示候选注释</span>
            </label>
            <label className="toggle">
              <input
                type="checkbox"
                checked={config.doublePinyin}
                onChange={(event) =>
                  setConfig({
                    ...config,
                    doublePinyin: event.target.checked,
                    doublePinyinScheme: config.doublePinyinScheme ?? "xiaohe",
                  })
                }
              />
              <span>启用双拼</span>
            </label>
            <label className="field">
              <span>双拼方案</span>
              <select
                value={config.doublePinyinScheme ?? "xiaohe"}
                onChange={(event) =>
                  setConfig({
                    ...config,
                    doublePinyinScheme: event.target.value as Config["doublePinyinScheme"],
                  })
                }
                disabled={!config.doublePinyin}
              >
                <option value="xiaohe">小鹤双拼</option>
                <option value="microsoft">微软/搜狗双拼</option>
              </select>
            </label>
            <label className="field">
              <span>模糊音</span>
              <input
                value={config.fuzzyInitials.join(", ")}
                onChange={(event) =>
                  setConfig({
                    ...config,
                    fuzzyInitials: event.target.value
                      .split(",")
                      .map((item) => item.trim())
                      .filter(Boolean),
                  })
                }
              />
            </label>
          </section>

          <section className="panel">
            <div className="panelHeader">
              <h2>候选栏外观</h2>
              <span>TSF/IMKit 原生窗口读取</span>
            </div>
            <div className="skinPresetGrid">
              {skinPresets.map((preset) => (
                <button
                  key={preset.id}
                  className={config.skin.theme === preset.id ? "skinPreset selected" : "skinPreset"}
                  onClick={() => setConfig({ ...config, skin: { ...preset.skin, fontFamily: config.skin.fontFamily } })}
                >
                  <span className="skinPreview" style={{ background: preset.skin.surface, borderColor: preset.skin.border }}>
                    <i style={{ background: preset.skin.accent }} />
                    <b style={{ color: preset.skin.text }}>你</b>
                  </span>
                  <span>{preset.name}</span>
                </button>
              ))}
            </div>
            <label className="field">
              <span>字体</span>
              <input
                value={config.skin.fontFamily}
                onChange={(event) =>
                  setConfig({ ...config, skin: { ...config.skin, fontFamily: event.target.value } })
                }
              />
            </label>
            <label className="field">
              <span>字号</span>
              <input
                type="number"
                min={12}
                max={24}
                value={config.skin.fontSize}
                onChange={(event) =>
                  setConfig({ ...config, skin: { ...config.skin, fontSize: Number(event.target.value) } })
                }
              />
            </label>
            <div className="swatches">
              {["#2563eb", "#16a34a", "#dc2626", "#db2777", "#0f766e"].map((color) => (
                <button
                  key={color}
                  className={color === config.skin.accent ? "swatch selected" : "swatch"}
                  style={{ backgroundColor: color }}
                  title={color}
                  onClick={() => setConfig({ ...config, skin: { ...config.skin, accent: color, theme: "custom" } })}
                />
              ))}
            </div>
            <div className="colorGrid">
              <label className="colorField">
                <span>底色</span>
                <input
                  type="color"
                  value={config.skin.surface}
                  onChange={(event) =>
                    setConfig({ ...config, skin: { ...config.skin, surface: event.target.value, theme: "custom" } })
                  }
                />
              </label>
              <label className="colorField">
                <span>文字</span>
                <input
                  type="color"
                  value={config.skin.text}
                  onChange={(event) =>
                    setConfig({ ...config, skin: { ...config.skin, text: event.target.value, theme: "custom" } })
                  }
                />
              </label>
              <label className="colorField">
                <span>次要文字</span>
                <input
                  type="color"
                  value={config.skin.mutedText}
                  onChange={(event) =>
                    setConfig({ ...config, skin: { ...config.skin, mutedText: event.target.value, theme: "custom" } })
                  }
                />
              </label>
              <label className="colorField">
                <span>边框</span>
                <input
                  type="color"
                  value={config.skin.border}
                  onChange={(event) =>
                    setConfig({ ...config, skin: { ...config.skin, border: event.target.value, theme: "custom" } })
                  }
                />
              </label>
              <label className="colorField">
                <span>高亮文字</span>
                <input
                  type="color"
                  value={config.skin.highlightText}
                  onChange={(event) =>
                    setConfig({ ...config, skin: { ...config.skin, highlightText: event.target.value, theme: "custom" } })
                  }
                />
              </label>
            </div>
          </section>

          <section className="panel">
            <div className="panelHeader">
              <h2>词库热更</h2>
              <span>{updateText}</span>
            </div>
            <label className="field">
              <span>GitHub Manifest</span>
              <input
                value={config.update.manifestUrls.join(", ")}
                onChange={(event) =>
                  setConfig({
                    ...config,
                    update: {
                      ...config.update,
                      manifestUrls: event.target.value
                        .split(",")
                        .map((item) => item.trim())
                        .filter(Boolean),
                    },
                  })
                }
              />
            </label>
            <label className="field">
              <span>镜像/CDN Base URL</span>
              <input
                value={(config.update.mirrorBaseUrls ?? []).join(", ")}
                onChange={(event) =>
                  setConfig({
                    ...config,
                    update: {
                      ...config.update,
                      mirrorBaseUrls: event.target.value
                        .split(",")
                        .map((item) => item.trim())
                        .filter(Boolean),
                    },
                  })
                }
              />
            </label>
            <div className="rowControls">
              <label className="toggle">
                <input
                  type="checkbox"
                  checked={config.update.autoCheck}
                  onChange={(event) =>
                    setConfig({ ...config, update: { ...config.update, autoCheck: event.target.checked } })
                  }
                />
                <span>自动检查</span>
              </label>
              <label className="toggle">
                <input
                  type="checkbox"
                  checked={config.update.autoApply}
                  onChange={(event) =>
                    setConfig({ ...config, update: { ...config.update, autoApply: event.target.checked } })
                  }
                />
                <span>自动应用</span>
              </label>
              <button className="secondary" disabled={updateBusy !== "idle"} onClick={checkUpdates}>
                <RefreshCw size={18} />
                {updateBusy === "checking" ? "检查中" : "检查更新"}
              </button>
              <button className="secondary" disabled={updateBusy !== "idle"} onClick={applyUpdates}>
                <Download size={18} />
                {updateBusy === "applying" ? "更新中" : "应用更新"}
              </button>
            </div>
          </section>

          <section className="panel">
            <div className="panelHeader">
              <h2>用户词库</h2>
              <span>{wordbookText}</span>
            </div>
            <div className="wordbookList">
              {wordbook.slice(0, 8).map((entry) => (
                <div className="wordbookRow" key={entry.key}>
                  <div>
                    <strong>{entry.text || entry.key}</strong>
                    <span>{entry.reading} · {entry.score}</span>
                  </div>
                  <button className="smallButton" onClick={() => deleteWordbookEntry(entry.key)}>
                    删除
                  </button>
                </div>
              ))}
              {wordbook.length === 0 && <div className="emptyWordbook">暂无用户词学习记录</div>}
            </div>
            <label className="field">
              <span>导入 / 编辑 JSON</span>
              <textarea
                className="wordbookDraft"
                spellCheck={false}
                value={wordbookDraft}
                onChange={(event) => setWordbookDraft(event.target.value)}
              />
            </label>
            <div className="rowControls">
              <button className="secondary" onClick={loadWordbook}>
                <RotateCcw size={18} />
                刷新
              </button>
              <button className="secondary" onClick={exportWordbook}>
                <FileDown size={18} />
                导出
              </button>
              <button className="secondary" onClick={() => importWordbook(true)}>
                <FileUp size={18} />
                合并导入
              </button>
              <button className="secondary danger" onClick={clearWordbook}>
                清空
              </button>
            </div>
          </section>

          <section className="panel phrasePanel">
            <div className="panelHeader">
              <h2>固定短语</h2>
              <span>{phraseText}</span>
            </div>
            <div className="phraseComposer">
              <label className="field">
                <span>编码</span>
                <input value={phraseReading} onChange={(event) => setPhraseReading(event.target.value)} />
              </label>
              <label className="field phraseTextField">
                <span>短语</span>
                <input value={phraseValue} onChange={(event) => setPhraseValue(event.target.value)} />
              </label>
              <label className="field">
                <span>权重</span>
                <input
                  type="number"
                  min={1}
                  max={99999}
                  value={phraseWeight}
                  onChange={(event) => setPhraseWeight(Number(event.target.value))}
                />
              </label>
              <button className="primary phraseAddButton" onClick={() => void addPhrase()}>
                <Plus size={18} />
                添加
              </button>
            </div>
            <div className="phraseList">
              {phrases.slice(0, 8).map((entry) => (
                <div className="phraseRow" key={`${entry.reading}|${entry.text}`}>
                  <button
                    className="phrasePreviewButton"
                    onClick={() => {
                      setPreview(entry.reading);
                      void runPreview(entry.reading);
                    }}
                  >
                    <strong>{entry.text}</strong>
                    <span>
                      {entry.reading} · {entry.weight ?? 50000}
                    </span>
                  </button>
                  <button className="smallButton" title="删除短语" onClick={() => void deletePhrase(entry)}>
                    <Trash2 size={14} />
                  </button>
                </div>
              ))}
              {phrases.length === 0 && <div className="emptyWordbook">暂无固定短语</div>}
            </div>
            <label className="field">
              <span>批量 JSON</span>
              <textarea
                className="wordbookDraft phraseDraft"
                spellCheck={false}
                value={phraseDraft}
                onChange={(event) => setPhraseDraft(event.target.value)}
              />
            </label>
            <div className="rowControls">
              <button className="secondary" onClick={() => void loadPhrases()}>
                <RotateCcw size={18} />
                刷新
              </button>
              <button className="secondary" onClick={exportPhrases}>
                <FileDown size={18} />
                导出
              </button>
              <button className="secondary" onClick={() => void importPhrases(true)}>
                <FileUp size={18} />
                合并导入
              </button>
              <button className="secondary" onClick={() => void importPhrases(false)}>
                替换
              </button>
              <button className="secondary danger" onClick={() => void clearPhrases()}>
                清空
              </button>
            </div>
          </section>

          <section className="panel rejectPanel">
            <div className="panelHeader">
              <h2>候选屏蔽</h2>
              <span>{rejectText}</span>
            </div>
            <div className="rejectList">
              {rejects.slice(0, 8).map((entry) => (
                <div className="phraseRow" key={`${entry.reading}|${entry.text}`}>
                  <button
                    className="phrasePreviewButton"
                    onClick={() => {
                      setPreview(entry.reading);
                      void runPreview(entry.reading);
                    }}
                  >
                    <strong>{entry.text}</strong>
                    <span>
                      {entry.reading}
                      {entry.comment ? ` · ${entry.comment}` : ""}
                    </span>
                  </button>
                  <button className="smallButton" title="恢复候选" onClick={() => void deleteReject(entry)}>
                    恢复
                  </button>
                </div>
              ))}
              {rejects.length === 0 && <div className="emptyWordbook">暂无已屏蔽候选</div>}
            </div>
            <label className="field">
              <span>批量 JSON</span>
              <textarea
                className="wordbookDraft phraseDraft"
                spellCheck={false}
                value={rejectDraft}
                onChange={(event) => setRejectDraft(event.target.value)}
              />
            </label>
            <div className="rowControls">
              <button className="secondary" onClick={() => void loadRejects()}>
                <RotateCcw size={18} />
                刷新
              </button>
              <button className="secondary" onClick={exportRejects}>
                <FileDown size={18} />
                导出
              </button>
              <button className="secondary" onClick={() => void importRejects(true)}>
                <FileUp size={18} />
                合并导入
              </button>
              <button className="secondary" onClick={() => void importRejects(false)}>
                替换
              </button>
              <button className="secondary danger" onClick={() => void clearRejects()}>
                全部恢复
              </button>
            </div>
          </section>

          <section className="panel catalogPanel">
            <div className="panelHeader">
              <div>
                <h2>表情与符号</h2>
                <span>{catalogText}</span>
              </div>
              <button className="iconButton" title="刷新资源目录" onClick={() => void loadCatalog()}>
                <RefreshCw size={18} />
              </button>
            </div>
            <div className="catalogControls">
              <label className="field">
                <span>类型</span>
                <select value={catalogKind} onChange={(event) => setCatalogKind(event.target.value)}>
                  <option value="all">全部</option>
                  <option value="emoji">Emoji</option>
                  <option value="kaomoji">颜文字</option>
                  <option value="symbol">符号</option>
                  <option value="agent">Agent</option>
                </select>
              </label>
              <label className="field">
                <span>搜索</span>
                <input
                  value={catalogQuery}
                  placeholder="zan / fs / 开心 / rewrite"
                  onChange={(event) => setCatalogQuery(event.target.value)}
                />
              </label>
            </div>
            <div className="catalogGrid">
              {catalogEntries.slice(0, 36).map((entry, index) => (
                <button
                  key={`${entry.kind}-${entry.source}-${entry.reading}-${entry.text}-${index}`}
                  className="catalogItem"
                  onClick={() => previewCatalogEntry(entry)}
                  title={`${entry.reading} · ${entry.source ?? ""}`}
                >
                  <strong>{entry.text}</strong>
                  <span>
                    {entry.reading}
                    {entry.comment ? ` · ${entry.comment}` : ""}
                  </span>
                  {kindLabel(entry.kind) && <i>{kindLabel(entry.kind)}</i>}
                </button>
              ))}
              {catalogEntries.length === 0 && <div className="emptyWordbook">暂无资源</div>}
            </div>
          </section>

          <section className="panel previewPanel">
            <div className="panelHeader">
              <h2>候选窗预览</h2>
              <span>{candidateCount} 个候选</span>
            </div>
            <label className="field">
              <span>输入串</span>
              <input value={preview} onChange={(event) => setPreview(event.target.value)} />
            </label>
            <div className="candidatePreviewShell">
              <div className="candidatePreview" style={candidateBarStyle}>
                <div className="compositionRow">
                  <span>{state?.buffer || previewCommitted || preview || "nihao"}</span>
                </div>
                <div className={candidateLayout === "vertical" ? "candidateBand vertical" : "candidateBand"}>
                  {previewCandidates.length > 0 ? (
                    previewCandidates.map((candidate, index) => (
                      <button
                        key={`${candidate.reading}-${candidate.text}-${index}`}
                        className={index === 0 ? "candidatePill selected" : "candidatePill"}
                        onClick={() => void runCandidateAction("select", index + 1)}
                        onContextMenu={(event) => {
                          event.preventDefault();
                          void runCandidateAction("forget", index + 1);
                        }}
                        title="左键上屏，右键屏蔽候选"
                      >
                        <b>{index + 1}</b>
                        <span className="candidateText">{candidate.text}</span>
                        {showCandidateComments && candidate.comment && <span className="candidateComment">{candidate.comment}</span>}
                        {kindLabel(candidate.kind) && <i>{kindLabel(candidate.kind)}</i>}
                      </button>
                    ))
                  ) : (
                    <span className="emptyCandidate">{previewCommitted ? `已上屏 ${previewCommitted}` : "等待输入"}</span>
                  )}
                  {candidateCount > candidatePageSize && (
                    <div className="candidatePager">
                      <button
                        className="candidatePageButton"
                        disabled={normalizedPreviewPageStart === 0}
                        onClick={() => void runCandidateAction("prev-page")}
                        title="上一页"
                      >
                        <ChevronLeft size={15} />
                      </button>
                      <span className="pageIndicator">
                        {normalizedPreviewPageStart + 1}-{normalizedPreviewPageStart + previewCandidates.length}/{candidateCount}
                      </span>
                      <button
                        className="candidatePageButton"
                        disabled={normalizedPreviewPageStart + previewCandidates.length >= candidateCount}
                        onClick={() => void runCandidateAction("next-page")}
                        title="下一页"
                      >
                        <ChevronRight size={15} />
                      </button>
                    </div>
                  )}
                </div>
              </div>
            </div>
          </section>

          <section className="panel typingLabPanel">
            <div className="panelHeader">
              <div>
                <h2>电竞打字性能实验室</h2>
                <span>React 输入路径 + 原生 SmokeEdit 双轨验证</span>
              </div>
              <div className="panelActions">
                <button className="iconButton" title="导出测试报告" onClick={exportTypingReport}>
                  <FileDown size={18} />
                </button>
                <button className="secondary" onClick={resetTypingLab}>
                  <RotateCcw size={18} />
                  重置
                </button>
              </div>
            </div>

            <div className="promptTabs">
              {typingPrompts.map((prompt) => (
                <button
                  key={prompt.id}
                  className={prompt.id === typingPrompt.id ? "selected" : ""}
                  onClick={() => {
                    setTypingPromptId(prompt.id);
                    resetTypingLab();
                  }}
                >
                  {prompt.name}
                </button>
              ))}
            </div>

            <div className="typingArena">
              <div className="targetLine">
                <Target size={18} />
                <span>{typingPrompt.text}</span>
              </div>
              <textarea
                value={typingText}
                spellCheck={false}
                onKeyDown={handleTypingKeyDown}
                onChange={(event) => handleTypingChange(event.target.value)}
                onCompositionStart={handleCompositionStart}
                onCompositionEnd={handleCompositionEnd}
                placeholder="在这里输入上方文本，测试键盘、候选选择、上屏和节奏..."
              />
              <div className="typingProgress">
                <span style={{ width: `${typingProgress}%` }} />
              </div>
            </div>

            <div className="probeGrid">
              <div className="probePanel">
                <div className="probeHeader">
                  <span>实时候选</span>
                  <strong>{typingProbe.input || "idle"}</strong>
                </div>
                <div className={candidateLayout === "vertical" ? "probeCandidates vertical" : "probeCandidates"} style={candidateBarStyle}>
                  {typingProbeCandidates.length > 0 ? (
                    typingProbeCandidates.map((candidate, index) => (
                      <span className={index === 0 ? "probeCandidate selected" : "probeCandidate"} key={`${candidate.reading}-${candidate.text}-${index}`}>
                        <b>{index + 1}</b>
                        <span>{candidate.text}</span>
                        {showCandidateComments && candidate.comment && <em>{candidate.comment}</em>}
                        {kindLabel(candidate.kind) && <i>{kindLabel(candidate.kind)}</i>}
                      </span>
                    ))
                  ) : (
                    <span className="probeEmpty">{typingProbe.status === "offline" ? "daemon offline" : "waiting"}</span>
                  )}
                </div>
              </div>
              <div className="probePanel">
                <div className="probeHeader">
                  <span>候选类型</span>
                  <strong>{typingProbeCandidates.length}</strong>
                </div>
                <div className="probeTags">
                  {(typingProbeKinds.length > 0 ? typingProbeKinds : ["普通"]).map((label) => (
                    <span key={label}>{label}</span>
                  ))}
                </div>
              </div>
              <div className="probePanel">
                <div className="probeHeader">
                  <span>最近按键</span>
                  <strong>{typingStatsRef.current.recentKeys.length}</strong>
                </div>
                <div className="keyTrail">
                  {typingStatsRef.current.recentKeys.length > 0 ? (
                    typingStatsRef.current.recentKeys.map((key, index) => <span key={`${key}-${index}`}>{key}</span>)
                  ) : (
                    <span>idle</span>
                  )}
                </div>
              </div>
            </div>

            <div className="metricsGrid">
              <Metric label="WPM" value={typingMetrics.wpm.toFixed(1)} />
              <Metric label="CPM" value={typingMetrics.cpm.toFixed(0)} />
              <Metric label="Events/s" value={typingMetrics.keysPerSecond.toFixed(1)} />
              <Metric label="Burst/s" value={typingMetrics.burstKeysPerSecond.toFixed(0)} />
              <Metric label="Avg latency" value={`${typingMetrics.avgLatencyMs.toFixed(1)} ms`} />
              <Metric label="P95 latency" value={`${typingMetrics.p95LatencyMs.toFixed(1)} ms`} />
              <Metric label="Accuracy" value={`${typingMetrics.accuracy.toFixed(1)}%`} />
              <Metric label="IME" value={typingMetrics.composing ? "Composing" : "Idle"} />
            </div>

            <div className="labFooter">
              <span>{formatDuration(typingMetrics.elapsedMs)}</span>
              <span>{typingMetrics.keyCount} keys</span>
              <span>{typingMetrics.changes} changes</span>
              <span>{typingMetrics.compositions} compositions</span>
              <span>{typingMetrics.errors} errors</span>
            </div>
          </section>
        </div>
      </section>
    </main>
  );
}

function Metric({ label, value }: { label: string; value: string }) {
  return (
    <div className="metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function formatDuration(ms: number) {
  const totalSeconds = Math.floor(ms / 1000);
  const minutes = Math.floor(totalSeconds / 60).toString().padStart(2, "0");
  const seconds = (totalSeconds % 60).toString().padStart(2, "0");
  return `${minutes}:${seconds}`;
}

function statusLabel(status: "loading" | "ready" | "offline" | "saved") {
  if (status === "loading") return "连接中";
  if (status === "ready") return "已连接";
  if (status === "saved") return "已保存";
  return "离线";
}

function kindLabel(kind?: string) {
  if (kind === "emoji") return "Emoji";
  if (kind === "kaomoji") return "颜";
  if (kind === "symbol") return "符";
  if (kind === "phrase") return "短";
  if (kind === "agent") return "AI";
  if (kind === "dynamic") return "时";
  return "";
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
