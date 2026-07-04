import { StrictMode, useCallback, useEffect, useMemo, useRef, useState, type CSSProperties, type KeyboardEvent } from "react";
import { createRoot } from "react-dom/client";
import {
  BookOpen,
  Check,
  CircleAlert,
  Download,
  FileDown,
  FileUp,
  Keyboard,
  Palette,
  RotateCcw,
  Save,
  SlidersHorizontal,
  Sparkles,
  Languages,
  RefreshCw,
  Gauge,
  Target,
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
  fuzzyInitials: string[];
  doublePinyin: boolean;
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

type Candidate = {
  text: string;
  reading: string;
  kind?: string;
  source?: string;
  weight: number;
  userScore: number;
};

type EngineState = {
  buffer: string;
  candidates: Candidate[] | null;
  committed?: string;
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
  fuzzyInitials: ["zh=z", "ch=c", "sh=s"],
  doublePinyin: false,
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
  const match = input.match(/[a-zA-Z]{1,32}$/);
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

function App() {
  const [config, setConfig] = useState<Config>(defaultConfig);
  const [preview, setPreview] = useState("nihao");
  const [state, setState] = useState<EngineState | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "offline" | "saved">("loading");
  const [updateText, setUpdateText] = useState("未检查");
  const [updateBusy, setUpdateBusy] = useState<"idle" | "checking" | "applying">("idle");
  const [wordbook, setWordbook] = useState<WordbookEntry[]>([]);
  const [wordbookDraft, setWordbookDraft] = useState("{}");
  const [wordbookText, setWordbookText] = useState("未读取");
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
  }, []);

  useEffect(() => {
    const timeout = window.setTimeout(() => void runPreview(preview), 120);
    return () => window.clearTimeout(timeout);
  }, [preview]);

  const candidateCount = state?.candidates?.length ?? 0;
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
  const previewCandidates = (state?.candidates ?? []).slice(0, 7);
  const typingProbeCandidates = typingProbe.candidates.slice(0, 7);
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
      setConfig(await res.json());
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
      if (status === "offline") setStatus("ready");
      setError("");
    } catch (err) {
      setStatus("offline");
      setError(err instanceof Error ? err.message : "daemon offline");
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
      setConfig(await res.json());
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
            <label className="toggle">
              <input
                type="checkbox"
                checked={config.doublePinyin}
                onChange={(event) => setConfig({ ...config, doublePinyin: event.target.checked })}
              />
              <span>启用小鹤双拼</span>
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
                  <span>{state?.buffer || preview || "nihao"}</span>
                </div>
                <div className="candidateBand">
                  {previewCandidates.length > 0 ? (
                    previewCandidates.map((candidate, index) => (
                      <button
                        key={`${candidate.reading}-${candidate.text}-${index}`}
                        className={index === 0 ? "candidatePill selected" : "candidatePill"}
                      >
                        <b>{index + 1}</b>
                        <span className="candidateText">{candidate.text}</span>
                        {kindLabel(candidate.kind) && <i>{kindLabel(candidate.kind)}</i>}
                      </button>
                    ))
                  ) : (
                    <span className="emptyCandidate">等待输入</span>
                  )}
                  {candidateCount > previewCandidates.length && (
                    <span className="pageIndicator">1-{previewCandidates.length}/{candidateCount}</span>
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
                <div className="probeCandidates" style={candidateBarStyle}>
                  {typingProbeCandidates.length > 0 ? (
                    typingProbeCandidates.map((candidate, index) => (
                      <span className={index === 0 ? "probeCandidate selected" : "probeCandidate"} key={`${candidate.reading}-${candidate.text}-${index}`}>
                        <b>{index + 1}</b>
                        <span>{candidate.text}</span>
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
  return "";
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
