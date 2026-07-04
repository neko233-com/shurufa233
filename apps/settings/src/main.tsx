import { StrictMode, useEffect, useMemo, useState } from "react";
import { createRoot } from "react-dom/client";
import {
  BookOpen,
  Check,
  CircleAlert,
  Keyboard,
  Palette,
  RotateCcw,
  Save,
  SlidersHorizontal,
  Sparkles,
  Languages,
  RefreshCw,
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

type Candidate = {
  text: string;
  reading: string;
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

const apiBase = "http://127.0.0.1:23333";

const defaultConfig: Config = {
  maxCandidates: 42,
  fuzzyInitials: ["zh=z", "ch=c", "sh=s"],
  doublePinyin: false,
  language: "zh-CN",
  mode: "zh",
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

function App() {
  const [config, setConfig] = useState<Config>(defaultConfig);
  const [preview, setPreview] = useState("nihao");
  const [state, setState] = useState<EngineState | null>(null);
  const [status, setStatus] = useState<"loading" | "ready" | "offline" | "saved">("loading");
  const [updateText, setUpdateText] = useState("未检查");
  const [error, setError] = useState("");

  useEffect(() => {
    void loadConfig();
  }, []);

  useEffect(() => {
    const timeout = window.setTimeout(() => void runPreview(preview), 120);
    return () => window.clearTimeout(timeout);
  }, [preview]);

  const candidateCount = state?.candidates?.length ?? 0;
  const accentStyle = useMemo(() => ({ "--accent": config.skin.accent }) as React.CSSProperties, [config.skin.accent]);
  const candidateBarStyle = useMemo(
    () =>
      ({
        fontFamily: config.skin.fontFamily,
        fontSize: config.skin.fontSize,
        background: config.skin.surface,
        borderColor: config.skin.border,
        color: config.skin.text,
        "--candidate-muted": config.skin.mutedText,
      }) as React.CSSProperties,
    [config.skin],
  );

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
    try {
      const res = await fetch(`${apiBase}/updates/check`);
      if (!res.ok) throw new Error(await res.text());
      const data = await res.json();
      setUpdateText(data.updateAvailable ? `发现 ${data.latestVersion}` : `已是最新 ${data.currentVersion}`);
      setError("");
    } catch (err) {
      setUpdateText("检查失败");
      setError(err instanceof Error ? err.message : "update check failed");
    }
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
                min={7}
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
              <span>启用双拼模式</span>
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
                <span>边框</span>
                <input
                  type="color"
                  value={config.skin.border}
                  onChange={(event) =>
                    setConfig({ ...config, skin: { ...config.skin, border: event.target.value, theme: "custom" } })
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
              <button className="secondary" onClick={checkUpdates}>
                <RefreshCw size={18} />
                检查更新
              </button>
            </div>
          </section>

          <section className="panel previewPanel">
            <div className="panelHeader">
              <h2>拼音预览</h2>
              <span>{candidateCount} 个候选</span>
            </div>
            <label className="field">
              <span>输入串</span>
              <input value={preview} onChange={(event) => setPreview(event.target.value)} />
            </label>
            <div className="candidateBar" style={candidateBarStyle}>
              <span className="buffer">{state?.buffer || "..."}</span>
              {(state?.candidates ?? []).map((candidate, index) => (
                <button key={`${candidate.reading}-${candidate.text}`}>
                  <b>{index + 1}</b>
                  {candidate.text}
                </button>
              ))}
            </div>
          </section>
        </div>
      </section>
    </main>
  );
}

function statusLabel(status: "loading" | "ready" | "offline" | "saved") {
  if (status === "loading") return "连接中";
  if (status === "ready") return "已连接";
  if (status === "saved") return "已保存";
  return "离线";
}

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
