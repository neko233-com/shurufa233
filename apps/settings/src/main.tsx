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
  schema?: string;
  candidatePageSize: number;
  candidateLayout: "horizontal" | "vertical" | "auto";
  showCandidateComments: boolean;
  fuzzyInitials: string[];
  spellerAlgebra?: string[];
  doublePinyin: boolean;
  doublePinyinScheme: "xiaohe" | "microsoft";
  language: string;
  mode: "zh" | "en";
  punctuation: "full" | "half";
  punctuationFullShape?: Record<string, string[]>;
  punctuationHalfShape?: Record<string, string[]>;
  recognizerPatterns?: Record<string, string>;
  script: "simplified" | "traditional";
  associations: boolean;
  keyProfile: "wechat" | "microsoft" | "rime" | "custom";
  shiftToggleMode: boolean;
  semicolonQuickSelect: boolean;
  quoteQuickSelect: boolean;
  bracketPageKeys: boolean;
  minusEqualPageKeys: boolean;
  commaPeriodPageKeys: boolean;
  appRules: AppRule[];
  skin: Skin;
  update: UpdateConfig;
};

type AppRule = {
  id: string;
  name: string;
  description?: string;
  processNames?: string[];
  exeContains?: string[];
  windowTitle?: string[];
  windowClass?: string[];
  passwordField?: boolean;
  terminal?: boolean;
  gameMode?: boolean;
  mode?: Config["mode"];
  punctuation?: Config["punctuation"];
  candidateLayout?: Config["candidateLayout"];
  disableCandidates?: boolean;
  disableLearning?: boolean;
  priority?: number;
};

type AppRuleResponse = {
  ok: boolean;
  rules: AppRule[];
  config?: Config;
};

type AppContext = {
  processName?: string;
  exePath?: string;
  windowTitle?: string;
  windowClass?: string;
  passwordField?: boolean;
  terminal?: boolean;
  gameMode?: boolean;
};

type AppContextDecision = {
  ok: boolean;
  matched: boolean;
  rule?: AppRule;
  context: AppContext;
  mode: Config["mode"];
  punctuation: Config["punctuation"];
  candidateLayout: Config["candidateLayout"];
  disableCandidates?: boolean;
  disableLearning?: boolean;
  reason?: string;
};

type UpdateConfig = {
  sourcePreset?: string;
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
  manifest?: DictionaryManifest;
};

type SourceProvenance = {
  preset?: string;
  url?: string;
  commit?: string;
  license?: string;
  convertCommand?: string;
};

type DictionaryManifest = {
  version: string;
  channel: string;
  generatedAt?: string;
  source?: SourceProvenance;
};

type UpdateApplyResult = {
  ok: boolean;
  manifestUrl: string;
  version: string;
  applied: string[];
};

type DictionaryRawSource = {
  label: string;
  url: string;
  role: string;
};

type DictionarySource = {
  id: string;
  name: string;
  kind: string;
  description: string;
  homepage: string;
  license: string;
  installable: boolean;
  manifestUrls?: string[];
  mirrorBaseUrls?: string[];
  rawSources?: DictionaryRawSource[];
  convertCommand?: string;
  syncCommand?: string;
};

type DictionarySourceResponse = {
  sources: DictionarySource[];
  selected: string;
};

type SwitchOption = {
  id: string;
  name: string;
  rimeName?: string;
  description: string;
  value: boolean;
  on: string;
  off: string;
  configField: string;
};

type SwitchResponse = {
  ok: boolean;
  selected?: SwitchOption;
  switches: SwitchOption[];
  config?: Config;
};

type SchemaPreset = {
  id: string;
  name: string;
  kind: string;
  rimeId?: string;
  description: string;
  tags?: string[];
  language: string;
  doublePinyin: boolean;
  doublePinyinScheme?: Config["doublePinyinScheme"];
  fuzzyInitials?: string[];
  punctuation?: Config["punctuation"];
  keyProfile?: Config["keyProfile"];
  candidateLayout?: Config["candidateLayout"];
  showCandidateComments: boolean;
  dictionarySourcePreset?: string;
};

type SchemaResponse = {
  selected: string;
  schemas: SchemaPreset[];
  config?: Config;
};

type RimeCustomResult = {
  ok: boolean;
  config?: Config;
  schema?: string;
  applied?: string[];
  warnings?: string[];
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

type PinResponse = {
  pins?: PhraseEntry[];
  entries?: PhraseEntry[];
  count: number;
  updatedAt: string;
};

type ProfileBundle = {
  ok?: boolean;
  version: number;
  product: string;
  exportedAt?: string;
  config?: Config;
  userScores?: Record<string, number>;
  phrases?: PhraseEntry[];
  rejects?: PhraseEntry[];
  pins?: PhraseEntry[];
  merge?: boolean;
  counts?: Record<string, number>;
};

type CatalogResponse = {
  kind: string;
  query?: string;
  count: number;
  entries: PhraseEntry[];
  updatedAt: string;
};

type ReverseLookupResponse = {
  query: string;
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
  pinned?: boolean;
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
  pinned?: PhraseEntry;
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
  schema: "wechat-pinyin",
  candidatePageSize: 7,
  candidateLayout: "horizontal",
  showCandidateComments: true,
  fuzzyInitials: ["zh=z", "ch=c", "sh=s"],
  spellerAlgebra: [],
  doublePinyin: false,
  doublePinyinScheme: "xiaohe",
  language: "zh-CN",
  mode: "zh",
  punctuation: "full",
  punctuationFullShape: {},
  punctuationHalfShape: {},
  recognizerPatterns: {
    email: "^[A-Za-z][-_.0-9A-Za-z]*@.*$",
    url: "^(www[.]|https?:|ftp:|mailto:).*$",
    reverse_lookup: "`[a-z]*'?$",
    uppercase: "[A-Z][-_+.'0-9A-Za-z]*$",
  },
  script: "simplified",
  associations: true,
  keyProfile: "wechat",
  shiftToggleMode: true,
  semicolonQuickSelect: true,
  quoteQuickSelect: true,
  bracketPageKeys: true,
  minusEqualPageKeys: true,
  commaPeriodPageKeys: false,
  appRules: [
    {
      id: "password-field-ascii",
      name: "密码框英文直通",
      passwordField: true,
      mode: "en",
      punctuation: "half",
      disableCandidates: true,
      disableLearning: true,
      priority: 1000,
    },
    {
      id: "terminal-ascii",
      name: "终端/命令行英文优先",
      processNames: ["windowsterminal.exe", "powershell.exe", "pwsh.exe", "cmd.exe"],
      mode: "en",
      punctuation: "half",
      priority: 800,
    },
    {
      id: "game-performance-ascii",
      name: "游戏/电竞性能模式",
      processNames: ["wegame.exe", "steam.exe", "cs2.exe", "valorant.exe", "mumunxmain.exe"],
      mode: "en",
      punctuation: "half",
      disableCandidates: true,
      priority: 700,
    },
  ],
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
    sourcePreset: "shurufa233-github",
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

function pinEntries(response: PinResponse): PhraseEntry[] {
  return [...(response.pins ?? response.entries ?? [])].sort(
    (left, right) => left.reading.localeCompare(right.reading) || left.text.localeCompare(right.text),
  );
}

function hydrateConfig(config: Config): Config {
  return {
    ...defaultConfig,
    ...config,
    candidatePageSize: Math.min(9, Math.max(3, config.candidatePageSize || defaultConfig.candidatePageSize)),
    candidateLayout: normalizeCandidateLayout(config.candidateLayout),
    script: normalizeScript(config.script),
    associations: config.associations ?? defaultConfig.associations,
    recognizerPatterns: config.recognizerPatterns ?? defaultConfig.recognizerPatterns,
    keyProfile: normalizeKeyProfile(config.keyProfile),
    shiftToggleMode: config.shiftToggleMode ?? defaultConfig.shiftToggleMode,
    semicolonQuickSelect: config.semicolonQuickSelect ?? defaultConfig.semicolonQuickSelect,
    quoteQuickSelect: config.quoteQuickSelect ?? defaultConfig.quoteQuickSelect,
    bracketPageKeys: config.bracketPageKeys ?? defaultConfig.bracketPageKeys,
    minusEqualPageKeys: config.minusEqualPageKeys ?? defaultConfig.minusEqualPageKeys,
    commaPeriodPageKeys: config.commaPeriodPageKeys ?? defaultConfig.commaPeriodPageKeys,
    appRules: config.appRules?.length ? config.appRules : defaultConfig.appRules,
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

function normalizeScript(script?: string): Config["script"] {
  return script === "traditional" ? "traditional" : "simplified";
}

function normalizeKeyProfile(profile?: string): Config["keyProfile"] {
  if (profile === "microsoft" || profile === "rime" || profile === "custom") return profile;
  return "wechat";
}

function applyKeyProfileConfig(config: Config, profile: Config["keyProfile"]): Config {
  if (profile === "custom") {
    return { ...config, keyProfile: "custom" };
  }
  if (profile === "rime") {
    return {
      ...config,
      keyProfile: "rime",
      shiftToggleMode: true,
      semicolonQuickSelect: false,
      quoteQuickSelect: false,
      bracketPageKeys: true,
      minusEqualPageKeys: true,
      commaPeriodPageKeys: true,
    };
  }
  return {
    ...config,
    keyProfile: profile,
    shiftToggleMode: true,
    semicolonQuickSelect: true,
    quoteQuickSelect: true,
    bracketPageKeys: true,
    minusEqualPageKeys: true,
    commaPeriodPageKeys: false,
  };
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
  const [dictionarySources, setDictionarySources] = useState<DictionarySource[]>([]);
  const [dictionarySourceText, setDictionarySourceText] = useState("未读取");
  const [schemas, setSchemas] = useState<SchemaPreset[]>([]);
  const [schemaText, setSchemaText] = useState("未读取");
  const [switches, setSwitches] = useState<SwitchOption[]>([]);
  const [switchText, setSwitchText] = useState("未读取");
  const [rimeCustomDraft, setRimeCustomDraft] = useState(`patch:
  schema_list:
    - schema: double_pinyin_flypy
  menu/page_size: 8
  key_binder/import_preset: alternative
  punctuator/import_preset: symbols
  switches:
    - name: ascii_punct
      reset: 0
    - name: candidate_comments
      reset: 1`);
  const [rimeCustomText, setRimeCustomText] = useState("未导入");
  const [appRules, setAppRules] = useState<AppRule[]>(defaultConfig.appRules);
  const [appRuleText, setAppRuleText] = useState("未读取");
  const [appContextProbe, setAppContextProbe] = useState<AppContext>({
    processName: "WeGame.exe",
    exePath: "",
    windowTitle: "",
    windowClass: "",
    gameMode: true,
  });
  const [appContextDecision, setAppContextDecision] = useState<AppContextDecision | null>(null);
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
  const [pins, setPins] = useState<PhraseEntry[]>([]);
  const [pinDraft, setPinDraft] = useState("{\n  \"entries\": []\n}");
  const [pinText, setPinText] = useState("未读取");
  const [profileDraft, setProfileDraft] = useState("{}");
  const [profileText, setProfileText] = useState("未导出");
  const [catalogKind, setCatalogKind] = useState("all");
  const [catalogQuery, setCatalogQuery] = useState("");
  const [catalogEntries, setCatalogEntries] = useState<PhraseEntry[]>([]);
  const [catalogText, setCatalogText] = useState("未读取");
  const [reverseQuery, setReverseQuery] = useState("你好");
  const [reverseEntries, setReverseEntries] = useState<PhraseEntry[]>([]);
  const [reverseText, setReverseText] = useState("未查询");
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
    void loadPins();
    void loadProfile(false);
    void loadCatalog();
    void loadDictionarySources();
    void loadSchemas();
    void loadSwitches();
    void loadAppRules();
    void resolveAppContext(appContextProbe);
    void runReverseLookup("你好");
  }, []);

  useEffect(() => {
    const timeout = window.setTimeout(() => void loadCatalog(), 160);
    return () => window.clearTimeout(timeout);
  }, [catalogKind, catalogQuery]);

  useEffect(() => {
    const timeout = window.setTimeout(() => void runReverseLookup(reverseQuery), 180);
    return () => window.clearTimeout(timeout);
  }, [reverseQuery]);

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
      if (data.pinned) {
        setPinText(`已置顶 ${data.pinned.reading}|${data.pinned.text}`);
        void loadPins();
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

  async function loadSchemas() {
    try {
      const res = await fetch(`${apiBase}/schemas`);
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as SchemaResponse;
      setSchemas(data.schemas ?? []);
      setSchemaText(data.selected ? `当前 ${data.selected}` : "未选择");
      if (data.config) {
        setConfig(hydrateConfig(data.config));
      }
      setError("");
    } catch (err) {
      setSchemaText("读取失败");
      setError(err instanceof Error ? err.message : "schema load failed");
    }
  }

  async function applySchema(id: string) {
    try {
      const res = await fetch(`${apiBase}/schemas/apply`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id }),
      });
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as SchemaResponse;
      setSchemas(data.schemas ?? schemas);
      setSchemaText(`已应用 ${data.selected}`);
      if (data.config) {
        setConfig(hydrateConfig(data.config));
      }
      setStatus("saved");
      window.setTimeout(() => setStatus("ready"), 1000);
      void runPreview(preview);
      setError("");
    } catch (err) {
      setSchemaText("应用失败");
      setError(err instanceof Error ? err.message : "schema apply failed");
    }
  }

  async function loadSwitches() {
    try {
      const res = await fetch(`${apiBase}/switches`);
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as SwitchResponse;
      setSwitches(data.switches ?? []);
      setSwitchText(data.switches?.length ? `${data.switches.length} 个开关` : "无开关");
      if (data.config) {
        setConfig(hydrateConfig(data.config));
      }
      setError("");
    } catch (err) {
      setSwitchText("读取失败");
      setError(err instanceof Error ? err.message : "switch load failed");
    }
  }

  async function applyRuntimeSwitch(id: string, value: boolean) {
    try {
      const res = await fetch(`${apiBase}/switches/apply`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id, value }),
      });
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as SwitchResponse;
      setSwitches(data.switches ?? []);
      setSwitchText(data.selected ? `${data.selected.name}: ${data.selected.value ? data.selected.on : data.selected.off}` : "已应用");
      if (data.config) {
        setConfig(hydrateConfig(data.config));
      }
      setStatus("saved");
      window.setTimeout(() => setStatus("ready"), 900);
      setError("");
    } catch (err) {
      setSwitchText("应用失败");
      setError(err instanceof Error ? err.message : "switch apply failed");
    }
  }

  async function importRimeCustom() {
    try {
      const res = await fetch(`${apiBase}/rime/custom`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ yaml: rimeCustomDraft }),
      });
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as RimeCustomResult;
      if (data.config) {
        setConfig(hydrateConfig(data.config));
      }
      setRimeCustomText(
        `${data.schema ?? data.config?.schema ?? "已应用"} · ${data.applied?.length ?? 0} 项${
          data.config?.spellerAlgebra?.length ? ` · algebra ${data.config.spellerAlgebra.length}` : ""
        }${
          data.config?.punctuationFullShape ? ` · 全角标点 ${Object.keys(data.config.punctuationFullShape).length}` : ""
        }${
          data.config?.recognizerPatterns ? ` · 识别器 ${Object.keys(data.config.recognizerPatterns).length}` : ""
        }${
          data.warnings?.length ? ` · ${data.warnings.length} 个警告` : ""
        }`,
      );
      void loadSchemas();
      void loadSwitches();
      void runPreview(preview);
      setStatus("saved");
      window.setTimeout(() => setStatus("ready"), 900);
      setError(data.warnings?.join("; ") ?? "");
    } catch (err) {
      setRimeCustomText("导入失败");
      setError(err instanceof Error ? err.message : "rime custom import failed");
    }
  }

  async function loadAppRules() {
    try {
      const res = await fetch(`${apiBase}/app-rules`);
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as AppRuleResponse;
      setAppRules(data.rules ?? []);
      setAppRuleText(data.rules?.length ? `${data.rules.length} 条规则` : "无规则");
      if (data.config) {
        setConfig(hydrateConfig(data.config));
      }
      setError("");
    } catch (err) {
      setAppRuleText("读取失败");
      setError(err instanceof Error ? err.message : "app rules load failed");
    }
  }

  async function resolveAppContext(context: AppContext) {
    try {
      const res = await fetch(`${apiBase}/app-context/resolve`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(context),
      });
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as AppContextDecision;
      setAppContextDecision(data);
      setError("");
    } catch (err) {
      setAppContextDecision(null);
      setError(err instanceof Error ? err.message : "app context resolve failed");
    }
  }

  async function checkUpdates() {
    setUpdateBusy("checking");
    try {
      const res = await fetch(`${apiBase}/updates/check`);
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as UpdateCheckResult;
      const source = data.manifest?.source?.preset || data.manifest?.source?.license || data.manifestUrl || "";
      const sourceText = source ? ` · ${source}` : "";
      setUpdateText(
        data.updateAvailable
          ? `发现 ${data.latestVersion}${sourceText}`
          : `已是最新 ${data.currentVersion}${sourceText}`,
      );
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

  async function loadDictionarySources() {
    try {
      const res = await fetch(`${apiBase}/updates/sources`);
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as DictionarySourceResponse;
      setDictionarySources(data.sources ?? []);
      setConfig((current) => ({
        ...current,
        update: {
          ...current.update,
          sourcePreset: data.selected || current.update.sourcePreset,
        },
      }));
      setDictionarySourceText(`${data.sources?.length ?? 0} 个来源`);
      setError("");
    } catch (err) {
      setDictionarySourceText("读取失败");
      setError(err instanceof Error ? err.message : "dictionary sources failed");
    }
  }

  async function applyDictionarySource(source: DictionarySource) {
    if (!source.installable) {
      setDictionarySourceText("需先转换发布");
      setError("该来源是 Rime/OpenCC 上游源码，需要先用转换命令生成 shurufa233 manifest");
      return;
    }
    try {
      const res = await fetch(`${apiBase}/updates/source`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ id: source.id }),
      });
      if (!res.ok) throw new Error(await res.text());
      setConfig(hydrateConfig(await res.json()));
      setDictionarySourceText(`已选择 ${source.name}`);
      setError("");
    } catch (err) {
      setDictionarySourceText("选择失败");
      setError(err instanceof Error ? err.message : "dictionary source apply failed");
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

  async function loadPins() {
    try {
      const res = await fetch(`${apiBase}/pins`);
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as PinResponse;
      const entries = pinEntries(data);
      setPins(entries);
      setPinDraft(JSON.stringify({ entries }, null, 2));
      setPinText(`${data.count ?? entries.length} 条已置顶`);
      setError("");
    } catch (err) {
      setPinText("读取失败");
      setError(err instanceof Error ? err.message : "pins load failed");
    }
  }

  async function savePinEntries(entries: PhraseEntry[], merge: boolean, statusText: string) {
    const res = await fetch(`${apiBase}/pins`, {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ entries, merge }),
    });
    if (!res.ok) throw new Error(await res.text());
    const data = (await res.json()) as PinResponse;
    const nextEntries = pinEntries(data);
    setPins(nextEntries);
    setPinDraft(JSON.stringify({ entries: nextEntries }, null, 2));
    setPinText(`${statusText} ${data.count ?? nextEntries.length} 条`);
    setError("");
    return nextEntries;
  }

  async function importPins(merge: boolean) {
    try {
      const parsed = JSON.parse(pinDraft) as PhraseEntry[] | { entries?: PhraseEntry[]; pins?: PhraseEntry[] };
      const entries = Array.isArray(parsed) ? parsed : parsed.entries ?? parsed.pins ?? [];
      await savePinEntries(entries, merge, merge ? "已合并" : "已替换");
      void runPreview(preview);
    } catch (err) {
      setPinText("导入失败");
      setError(err instanceof Error ? err.message : "pin import failed");
    }
  }

  async function deletePin(entry: PhraseEntry) {
    try {
      const key = `${entry.reading}|${entry.text}`;
      const res = await fetch(`${apiBase}/pins?key=${encodeURIComponent(key)}`, { method: "DELETE" });
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as PinResponse;
      const nextEntries = pinEntries(data);
      setPins(nextEntries);
      setPinDraft(JSON.stringify({ entries: nextEntries }, null, 2));
      setPinText(`剩余 ${data.count ?? nextEntries.length} 条`);
      void runPreview(preview);
      setError("");
    } catch (err) {
      setPinText("取消失败");
      setError(err instanceof Error ? err.message : "pin delete failed");
    }
  }

  async function clearPins() {
    if (!window.confirm("取消所有置顶候选？")) return;
    try {
      const res = await fetch(`${apiBase}/pins`, { method: "DELETE" });
      if (!res.ok) throw new Error(await res.text());
      setPins([]);
      setPinDraft("{\n  \"entries\": []\n}");
      setPinText("已全部取消");
      void runPreview(preview);
      setError("");
    } catch (err) {
      setPinText("取消失败");
      setError(err instanceof Error ? err.message : "pin clear failed");
    }
  }

  function exportPins() {
    const blob = new Blob([JSON.stringify({ entries: pins }, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = "shurufa233-user-pins.json";
    anchor.click();
    URL.revokeObjectURL(url);
  }

  async function loadProfile(updateDraft = true) {
    try {
      const res = await fetch(`${apiBase}/profile`);
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as ProfileBundle;
      const counts = data.counts ?? {};
      if (updateDraft) {
        setProfileDraft(JSON.stringify(data, null, 2));
      }
      setProfileText(
        `配置 1 · 用户词 ${counts.userScores ?? Object.keys(data.userScores ?? {}).length} · 短语 ${
          counts.phrases ?? data.phrases?.length ?? 0
        }`,
      );
      setError("");
      return data;
    } catch (err) {
      setProfileText("导出失败");
      setError(err instanceof Error ? err.message : "profile export failed");
      return null;
    }
  }

  async function exportProfileFile() {
    const data = await loadProfile(true);
    if (!data) return;
    const stamp = new Date().toISOString().replaceAll(":", "-").replace(/\.\d+Z$/, "Z");
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const anchor = document.createElement("a");
    anchor.href = url;
    anchor.download = `shurufa233-profile-${stamp}.json`;
    anchor.click();
    URL.revokeObjectURL(url);
  }

  async function importProfile(merge: boolean) {
    try {
      const parsed = JSON.parse(profileDraft) as ProfileBundle | { profile?: ProfileBundle };
      const bundle = "profile" in parsed ? parsed.profile ?? parsed : parsed;
      const res = await fetch(`${apiBase}/profile`, {
        method: "PUT",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ ...bundle, merge }),
      });
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as ProfileBundle;
      const counts = data.counts ?? {};
      setProfileDraft(JSON.stringify(data, null, 2));
      setProfileText(
        `${merge ? "已合并" : "已替换"} · 用户词 ${counts.userScores ?? Object.keys(data.userScores ?? {}).length} · 短语 ${
          counts.phrases ?? data.phrases?.length ?? 0
        }`,
      );
      if (data.config) {
        setConfig(hydrateConfig(data.config));
      }
      void loadWordbook();
      void loadPhrases();
      void loadRejects();
      void loadPins();
      void runPreview(preview);
      setError("");
    } catch (err) {
      setProfileText("导入失败");
      setError(err instanceof Error ? err.message : "profile import failed");
    }
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

  async function runReverseLookup(text: string) {
    const trimmed = text.trim();
    if (!trimmed) {
      setReverseEntries([]);
      setReverseText("未查询");
      return;
    }
    try {
      const query = new URLSearchParams({
        q: trimmed,
        limit: "20",
      });
      const res = await fetch(`${apiBase}/engine/reverse?${query.toString()}`);
      if (!res.ok) throw new Error(await res.text());
      const data = (await res.json()) as ReverseLookupResponse;
      setReverseEntries(data.entries ?? []);
      setReverseText(`${data.count ?? data.entries?.length ?? 0} 条读音`);
      setError("");
    } catch (err) {
      setReverseEntries([]);
      setReverseText("反查失败");
      setError(err instanceof Error ? err.message : "reverse lookup failed");
    }
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
              <span>{schemaText}</span>
            </div>
            <div className="schemaGrid">
              {schemas.map((schema) => (
                <button
                  key={schema.id}
                  className={config.schema === schema.id ? "schemaCard selected" : "schemaCard"}
                  onClick={() => void applySchema(schema.id)}
                  title={schema.description}
                >
                  <strong>{schema.name}</strong>
                  <span>
                    {schema.doublePinyin ? `双拼 · ${schema.doublePinyinScheme ?? ""}` : "全拼"}
                    {schema.rimeId ? ` · ${schema.rimeId}` : ""}
                  </span>
                  {schema.dictionarySourcePreset && <em>{schema.dictionarySourcePreset}</em>}
                </button>
              ))}
            </div>
            <div className="subHeader">
              <span>Rime 开关</span>
              <small>{switchText}</small>
            </div>
            <div className="toggleGrid">
              {switches.map((item) => (
                <label className="toggle" key={item.id} title={item.description}>
                  <input type="checkbox" checked={item.value} onChange={(event) => void applyRuntimeSwitch(item.id, event.target.checked)} />
                  <span>{item.name} · {item.value ? item.on : item.off}</span>
                </label>
              ))}
            </div>
            <div className="subHeader">
              <span>Rime 配方</span>
              <small>{rimeCustomText}</small>
            </div>
            <label className="field">
              <span>custom.yaml patch</span>
              <textarea
                className="wordbookDraft phraseDraft"
                spellCheck={false}
                value={rimeCustomDraft}
                onChange={(event) => setRimeCustomDraft(event.target.value)}
              />
            </label>
            <div className="rowControls">
              <button className="secondary" onClick={() => void importRimeCustom()}>
                <FileUp size={18} />
                导入配方
              </button>
            </div>
            <div className="subHeader">
              <span>应用规则</span>
              <small>{appRuleText}</small>
            </div>
            <div className="appRuleGrid">
              {appRules.slice(0, 4).map((rule) => (
                <div className="appRuleCard" key={rule.id}>
                  <strong>{rule.name}</strong>
                  <span>{rule.mode ?? "keep"} · {rule.punctuation ?? "keep"}{rule.disableCandidates ? " · 禁候选" : ""}</span>
                  <small>{[...(rule.processNames ?? []), ...(rule.exeContains ?? [])].slice(0, 3).join(" / ") || rule.id}</small>
                </div>
              ))}
            </div>
            <div className="contextProbe">
              <input
                value={appContextProbe.processName ?? ""}
                onChange={(event) => setAppContextProbe({ ...appContextProbe, processName: event.target.value })}
                placeholder="process.exe"
              />
              <input
                value={appContextProbe.windowTitle ?? ""}
                onChange={(event) => setAppContextProbe({ ...appContextProbe, windowTitle: event.target.value })}
                placeholder="window title"
              />
              <label>
                <input
                  type="checkbox"
                  checked={!!appContextProbe.passwordField}
                  onChange={(event) => setAppContextProbe({ ...appContextProbe, passwordField: event.target.checked })}
                />
                密码
              </label>
              <label>
                <input
                  type="checkbox"
                  checked={!!appContextProbe.terminal}
                  onChange={(event) => setAppContextProbe({ ...appContextProbe, terminal: event.target.checked })}
                />
                终端
              </label>
              <label>
                <input
                  type="checkbox"
                  checked={!!appContextProbe.gameMode}
                  onChange={(event) => setAppContextProbe({ ...appContextProbe, gameMode: event.target.checked })}
                />
                游戏
              </label>
              <button onClick={() => void resolveAppContext(appContextProbe)}>探测</button>
            </div>
            {appContextDecision && (
              <div className={appContextDecision.matched ? "contextDecision matched" : "contextDecision"}>
                <span>{appContextDecision.matched ? appContextDecision.rule?.name ?? appContextDecision.reason : "默认规则"}</span>
                <strong>{appContextDecision.mode} · {appContextDecision.punctuation}{appContextDecision.disableCandidates ? " · 禁候选" : ""}</strong>
              </div>
            )}
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
            <div className="segmented">
              <button
                className={(config.script ?? "simplified") === "simplified" ? "selected" : ""}
                onClick={() => setConfig({ ...config, script: "simplified" })}
              >
                简体输出
              </button>
              <button
                className={config.script === "traditional" ? "selected" : ""}
                onClick={() => setConfig({ ...config, script: "traditional" })}
              >
                繁体输出
              </button>
            </div>
            <div className="segmented three">
              <button
                className={(config.keyProfile ?? "wechat") === "wechat" ? "selected" : ""}
                onClick={() => setConfig(applyKeyProfileConfig(config, "wechat"))}
              >
                微信键位
              </button>
              <button
                className={config.keyProfile === "rime" ? "selected" : ""}
                onClick={() => setConfig(applyKeyProfileConfig(config, "rime"))}
              >
                Rime 键位
              </button>
              <button
                className={config.keyProfile === "custom" ? "selected" : ""}
                onClick={() => setConfig(applyKeyProfileConfig(config, "custom"))}
              >
                自定义
              </button>
            </div>
            <div className="toggleGrid">
              <label className="toggle">
                <input
                  type="checkbox"
                  checked={config.shiftToggleMode}
                  onChange={(event) => setConfig({ ...config, keyProfile: "custom", shiftToggleMode: event.target.checked })}
                />
                <span>Shift 切中英</span>
              </label>
              <label className="toggle">
                <input
                  type="checkbox"
                  checked={config.semicolonQuickSelect}
                  onChange={(event) => setConfig({ ...config, keyProfile: "custom", semicolonQuickSelect: event.target.checked })}
                />
                <span>; 选二候选</span>
              </label>
              <label className="toggle">
                <input
                  type="checkbox"
                  checked={config.quoteQuickSelect}
                  onChange={(event) => setConfig({ ...config, keyProfile: "custom", quoteQuickSelect: event.target.checked })}
                />
                <span>' 选三候选</span>
              </label>
              <label className="toggle">
                <input
                  type="checkbox"
                  checked={config.bracketPageKeys}
                  onChange={(event) => setConfig({ ...config, keyProfile: "custom", bracketPageKeys: event.target.checked })}
                />
                <span>[] 翻页</span>
              </label>
              <label className="toggle">
                <input
                  type="checkbox"
                  checked={config.minusEqualPageKeys}
                  onChange={(event) => setConfig({ ...config, keyProfile: "custom", minusEqualPageKeys: event.target.checked })}
                />
                <span>-= 翻页</span>
              </label>
              <label className="toggle">
                <input
                  type="checkbox"
                  checked={config.commaPeriodPageKeys}
                  onChange={(event) => setConfig({ ...config, keyProfile: "custom", commaPeriodPageKeys: event.target.checked })}
                />
                <span>,. 翻页</span>
              </label>
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
                checked={config.associations ?? true}
                onChange={(event) => setConfig({ ...config, associations: event.target.checked })}
              />
              <span>上屏后显示联想词</span>
            </label>
            <label className="toggle">
              <input
                type="checkbox"
                checked={config.doublePinyin}
                onChange={(event) =>
                  setConfig({
                    ...config,
                    schema: event.target.checked ? `double-pinyin-${config.doublePinyinScheme ?? "xiaohe"}` : "wechat-pinyin",
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
                    schema: `double-pinyin-${event.target.value}`,
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
            <div className="sourceHeader">
              <span>{dictionarySourceText}</span>
              <button className="smallButton" onClick={() => void loadDictionarySources()}>
                刷新来源
              </button>
            </div>
            <div className="sourceGrid">
              {dictionarySources.map((source) => (
                <button
                  className={`sourceItem ${config.update.sourcePreset === source.id ? "selected" : ""}`}
                  key={source.id}
                  onClick={() => void applyDictionarySource(source)}
                  title={source.homepage}
                >
                  <strong>{source.name}</strong>
                  <span>{source.installable ? "可直接热更" : "上游转换源"} · {source.license}</span>
                  <small>{source.description}</small>
                  {source.rawSources && source.rawSources.length > 0 && (
                    <i>{source.rawSources.slice(0, 2).map((item) => item.label).join(", ")}</i>
                  )}
                  {source.convertCommand && <code>{source.convertCommand}</code>}
                  {source.syncCommand && <code>{source.syncCommand}</code>}
                </button>
              ))}
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
              <h2>资料同步</h2>
              <span>{profileText}</span>
            </div>
            <label className="field">
              <span>迁移包 JSON</span>
              <textarea
                className="wordbookDraft phraseDraft"
                spellCheck={false}
                value={profileDraft}
                onChange={(event) => setProfileDraft(event.target.value)}
              />
            </label>
            <div className="rowControls">
              <button className="secondary" onClick={() => void loadProfile(true)}>
                <RotateCcw size={18} />
                刷新
              </button>
              <button className="secondary" onClick={() => void exportProfileFile()}>
                <FileDown size={18} />
                导出资料
              </button>
              <button className="secondary" onClick={() => void importProfile(true)}>
                <FileUp size={18} />
                合并导入
              </button>
              <button className="secondary danger" onClick={() => void importProfile(false)}>
                <Trash2 size={18} />
                替换导入
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

          <section className="panel rejectPanel">
            <div className="panelHeader">
              <h2>候选置顶</h2>
              <span>{pinText}</span>
            </div>
            <div className="rejectList">
              {pins.slice(0, 8).map((entry) => (
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
                  <button className="smallButton" title="取消置顶" onClick={() => void deletePin(entry)}>
                    取消
                  </button>
                </div>
              ))}
              {pins.length === 0 && <div className="emptyWordbook">暂无置顶候选</div>}
            </div>
            <label className="field">
              <span>批量 JSON</span>
              <textarea
                className="wordbookDraft phraseDraft"
                spellCheck={false}
                value={pinDraft}
                onChange={(event) => setPinDraft(event.target.value)}
              />
            </label>
            <div className="rowControls">
              <button className="secondary" onClick={() => void loadPins()}>
                <RotateCcw size={18} />
                刷新
              </button>
              <button className="secondary" onClick={exportPins}>
                <FileDown size={18} />
                导出
              </button>
              <button className="secondary" onClick={() => void importPins(true)}>
                <FileUp size={18} />
                合并导入
              </button>
              <button className="secondary" onClick={() => void importPins(false)}>
                替换
              </button>
              <button className="secondary danger" onClick={() => void clearPins()}>
                全部取消
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

          <section className="panel reversePanel">
            <div className="panelHeader">
              <h2>拼音反查</h2>
              <span>{reverseText}</span>
            </div>
            <label className="field">
              <span>中文词</span>
              <input
                value={reverseQuery}
                placeholder="你好 / 输入法"
                onChange={(event) => setReverseQuery(event.target.value)}
              />
            </label>
            <div className="reverseList">
              {reverseEntries.slice(0, 10).map((entry, index) => (
                <button
                  key={`${entry.reading}-${entry.text}-${entry.source}-${index}`}
                  className="reverseRow"
                  onClick={() => {
                    setPreview(entry.reading);
                    void runPreview(entry.reading);
                  }}
                  title={entry.source ?? ""}
                >
                  <strong>{entry.text}</strong>
                  <span>{entry.reading}</span>
                  <em>
                    {entry.comment || kindLabel(entry.kind) || entry.source || "反查"}
                  </em>
                </button>
              ))}
              {reverseEntries.length === 0 && <div className="emptyWordbook">暂无反查结果</div>}
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
                        onDoubleClick={(event) => {
                          event.preventDefault();
                          void runCandidateAction("pin", index + 1);
                        }}
                        onContextMenu={(event) => {
                          event.preventDefault();
                          void runCandidateAction("forget", index + 1);
                        }}
                        title="左键上屏，双击置顶，右键屏蔽候选"
                      >
                        <b>{index + 1}</b>
                        <span className="candidateText">{candidate.text}</span>
                        {showCandidateComments && candidate.comment && <span className="candidateComment">{candidate.comment}</span>}
                        {candidate.pinned && <i>顶</i>}
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
                        {candidate.pinned && <i>顶</i>}
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
