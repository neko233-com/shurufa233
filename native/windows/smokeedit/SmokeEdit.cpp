#define NOMINMAX
#include <windows.h>
#include <imm.h>
#include <msctf.h>
#include <strsafe.h>

#include <algorithm>
#include <vector>
#include <cwchar>

namespace {

constexpr wchar_t kClassName[] = L"Shurufa233SmokeEditWindow";
constexpr wchar_t kSingletonName[] = L"Local\\Shurufa233SmokeEditSingleton";
constexpr UINT_PTR kStatsTimer = 1;
constexpr int kEditTop = 628;
constexpr int kRecentKeyWindow = 256;
constexpr int kLatencyWindow = 512;
constexpr int kRecentKeyLabels = 14;

const CLSID kClsidTextService = {
    0x3d7b8d06,
    0x9872,
    0x4c31,
    {0xb7, 0x7d, 0x3b, 0x87, 0x32, 0x7c, 0xbf, 0x64}};

const GUID kProfileGuid = {
    0xb68911a2,
    0x4478,
    0x491c,
    {0xa6, 0x24, 0x97, 0x84, 0x41, 0x64, 0x8e, 0x20}};

constexpr LANGID kLanguage = MAKELANGID(LANG_CHINESE, SUBLANG_CHINESE_SIMPLIFIED);

struct Metrics {
  LARGE_INTEGER frequency{};
  LARGE_INTEGER startedAt{};
  LARGE_INTEGER lastKeyAt{};
  int keyDowns = 0;
  int chars = 0;
  int textLength = 0;
  int changes = 0;
  int imeStarts = 0;
  int imeEnds = 0;
  double latencyTotalMs = 0.0;
  int latencySamples = 0;
  double peakKeysPerSecond = 0.0;
  LARGE_INTEGER recentKeys[kRecentKeyWindow]{};
  int recentKeyCursor = 0;
  int recentKeyCount = 0;
  double latencyWindow[kLatencyWindow]{};
  int latencyCursor = 0;
  int latencyCount = 0;
  wchar_t recentKeyLabels[kRecentKeyLabels][32]{};
  int recentKeyLabelCursor = 0;
  int recentKeyLabelCount = 0;
  bool started = false;
};

HWND g_edit = nullptr;
HFONT g_titleFont = nullptr;
HFONT g_bodyFont = nullptr;
HFONT g_editFont = nullptr;
WNDPROC g_originalEditProc = nullptr;
Metrics g_metrics{};
wchar_t g_imeStatus[128] = L"F6 activate shurufa233 for this lab";
bool g_shurufaActive = false;
bool g_suppressInputMetrics = false;

COLORREF Rgb(int r, int g, int b) {
  return RGB(r, g, b);
}

double MsSince(const LARGE_INTEGER &from, const LARGE_INTEGER &to) {
  if (g_metrics.frequency.QuadPart == 0 || from.QuadPart == 0) {
    return 0.0;
  }
  return static_cast<double>(to.QuadPart - from.QuadPart) * 1000.0 /
         static_cast<double>(g_metrics.frequency.QuadPart);
}

LARGE_INTEGER Now() {
  LARGE_INTEGER value{};
  QueryPerformanceCounter(&value);
  return value;
}

void CancelImeComposition(HWND hwnd) {
  if (!hwnd) {
    return;
  }
  HIMC context = ImmGetContext(hwnd);
  if (context) {
    ImmNotifyIME(context, NI_COMPOSITIONSTR, CPS_CANCEL, 0);
    ImmNotifyIME(context, NI_CLOSECANDIDATE, 0, 0);
    ImmReleaseContext(hwnd, context);
  }
  HWND imeWindow = ImmGetDefaultIMEWnd(hwnd);
  if (imeWindow) {
    SendMessageW(imeWindow, WM_IME_CONTROL, IMC_CLOSESTATUSWINDOW, 0);
  }
}

void SendEscapeToFocusedInput() {
  if (!g_edit) {
    return;
  }
  g_suppressInputMetrics = true;
  INPUT input[2]{};
  input[0].type = INPUT_KEYBOARD;
  input[0].ki.wVk = VK_ESCAPE;
  input[1].type = INPUT_KEYBOARD;
  input[1].ki.wVk = VK_ESCAPE;
  input[1].ki.dwFlags = KEYEVENTF_KEYUP;
  SendInput(ARRAYSIZE(input), input, sizeof(INPUT));
  Sleep(60);
  g_suppressInputMetrics = false;
}

void PrepareEditForInputSwitch(bool clearText) {
  if (!g_edit) {
    return;
  }
  SetFocus(g_edit);
  SendEscapeToFocusedInput();
  CancelImeComposition(g_edit);
  if (clearText) {
    SetWindowTextW(g_edit, L"");
  }
  SendMessageW(g_edit, EM_SETSEL, static_cast<WPARAM>(-1), static_cast<LPARAM>(-1));
}

void ResetMetrics(HWND hwnd) {
  g_metrics = Metrics{};
  QueryPerformanceFrequency(&g_metrics.frequency);
  PrepareEditForInputSwitch(true);
  InvalidateRect(hwnd, nullptr, TRUE);
}

void EnsureStarted() {
  if (!g_metrics.started) {
    g_metrics.startedAt = Now();
    g_metrics.started = true;
  }
}

void PushKeyLabel(const wchar_t *label) {
  StringCchCopyW(g_metrics.recentKeyLabels[g_metrics.recentKeyLabelCursor],
                 ARRAYSIZE(g_metrics.recentKeyLabels[g_metrics.recentKeyLabelCursor]), label);
  g_metrics.recentKeyLabelCursor = (g_metrics.recentKeyLabelCursor + 1) % kRecentKeyLabels;
  if (g_metrics.recentKeyLabelCount < kRecentKeyLabels) {
    g_metrics.recentKeyLabelCount++;
  }
}

void RecordKeyLabel(WPARAM key, LPARAM lparam) {
  wchar_t label[32]{};
  if (key >= L'A' && key <= L'Z') {
    StringCchPrintfW(label, ARRAYSIZE(label), L"%c", static_cast<wchar_t>(key));
    PushKeyLabel(label);
    return;
  }
  if (key >= L'0' && key <= L'9') {
    StringCchPrintfW(label, ARRAYSIZE(label), L"%c", static_cast<wchar_t>(key));
    PushKeyLabel(label);
    return;
  }
  switch (key) {
    case VK_PROCESSKEY:
      PushKeyLabel(L"IME");
      return;
    case VK_SPACE:
      PushKeyLabel(L"Space");
      return;
    case VK_BACK:
      PushKeyLabel(L"Backspace");
      return;
    case VK_RETURN:
      PushKeyLabel(L"Enter");
      return;
    default:
      break;
  }
  LONG keyInfo = static_cast<LONG>(lparam);
  if (GetKeyNameTextW(keyInfo, label, ARRAYSIZE(label)) == 0) {
    StringCchPrintfW(label, ARRAYSIZE(label), L"VK %02X", static_cast<unsigned int>(key));
  }
  PushKeyLabel(label);
}

void RecordKeyDown(WPARAM key, LPARAM lparam) {
  EnsureStarted();
  const LARGE_INTEGER now = Now();
  g_metrics.keyDowns++;
  g_metrics.lastKeyAt = now;
  RecordKeyLabel(key, lparam);
  g_metrics.recentKeys[g_metrics.recentKeyCursor] = now;
  g_metrics.recentKeyCursor = (g_metrics.recentKeyCursor + 1) % kRecentKeyWindow;
  if (g_metrics.recentKeyCount < kRecentKeyWindow) {
    g_metrics.recentKeyCount++;
  }

  int inWindow = 0;
  for (int i = 0; i < g_metrics.recentKeyCount; ++i) {
    if (MsSince(g_metrics.recentKeys[i], now) <= 1000.0) {
      inWindow++;
    }
  }
  g_metrics.peakKeysPerSecond =
      std::max(g_metrics.peakKeysPerSecond, static_cast<double>(inWindow));
}

void RecordLatencySample(double latencyMs) {
  g_metrics.latencyTotalMs += latencyMs;
  g_metrics.latencySamples++;
  g_metrics.latencyWindow[g_metrics.latencyCursor] = latencyMs;
  g_metrics.latencyCursor = (g_metrics.latencyCursor + 1) % kLatencyWindow;
  if (g_metrics.latencyCount < kLatencyWindow) {
    g_metrics.latencyCount++;
  }
}

double PercentileLatency(double percentile) {
  if (g_metrics.latencyCount <= 0) {
    return 0.0;
  }
  std::vector<double> samples;
  samples.reserve(g_metrics.latencyCount);
  for (int i = 0; i < g_metrics.latencyCount; ++i) {
    samples.push_back(g_metrics.latencyWindow[i]);
  }
  std::sort(samples.begin(), samples.end());
  const double scaled = percentile * static_cast<double>(samples.size() - 1);
  const size_t index = static_cast<size_t>(std::clamp(scaled, 0.0, static_cast<double>(samples.size() - 1)));
  return samples[index];
}

HRESULT GetActiveKeyboardProfile(TF_INPUTPROCESSORPROFILE *profile) {
  if (!profile) {
    return E_INVALIDARG;
  }
  ITfInputProcessorProfileMgr *mgr = nullptr;
  HRESULT hr = CoCreateInstance(CLSID_TF_InputProcessorProfiles, nullptr, CLSCTX_INPROC_SERVER,
                                IID_ITfInputProcessorProfileMgr,
                                reinterpret_cast<void **>(&mgr));
  if (SUCCEEDED(hr) && mgr) {
    hr = mgr->GetActiveProfile(GUID_TFCAT_TIP_KEYBOARD, profile);
    mgr->Release();
  }
  return hr;
}

bool IsShurufaActive() {
  TF_INPUTPROCESSORPROFILE active{};
  if (FAILED(GetActiveKeyboardProfile(&active))) {
    return false;
  }
  return active.langid == kLanguage && IsEqualGUID(active.clsid, kClsidTextService) &&
         IsEqualGUID(active.guidProfile, kProfileGuid);
}

HRESULT ActivateShurufaProfileOnce() {
  HRESULT hr = S_OK;
  ITfInputProcessorProfiles *profiles = nullptr;
  hr = CoCreateInstance(CLSID_TF_InputProcessorProfiles, nullptr, CLSCTX_INPROC_SERVER,
                        IID_ITfInputProcessorProfiles,
                        reinterpret_cast<void **>(&profiles));
  if (SUCCEEDED(hr) && profiles) {
    profiles->EnableLanguageProfile(kClsidTextService, kLanguage, kProfileGuid, TRUE);
    profiles->ChangeCurrentLanguage(kLanguage);
    profiles->ActivateLanguageProfile(kClsidTextService, kLanguage, kProfileGuid);
    profiles->Release();
  }

  ITfInputProcessorProfileMgr *mgr = nullptr;
  hr = CoCreateInstance(CLSID_TF_InputProcessorProfiles, nullptr, CLSCTX_INPROC_SERVER,
                        IID_ITfInputProcessorProfileMgr,
                        reinterpret_cast<void **>(&mgr));
  if (SUCCEEDED(hr) && mgr) {
    DWORD flags = TF_IPPMF_FORSESSION | TF_IPPMF_ENABLEPROFILE;
#ifdef TF_IPPMF_DONTCARECURRENTINPUTLANGUAGE
    flags |= TF_IPPMF_DONTCARECURRENTINPUTLANGUAGE;
#endif
    hr = mgr->ActivateProfile(TF_PROFILETYPE_INPUTPROCESSOR, kLanguage, kClsidTextService,
                              kProfileGuid, nullptr, flags);
    if (FAILED(hr)) {
      hr = mgr->ActivateProfile(TF_PROFILETYPE_INPUTPROCESSOR, kLanguage, kClsidTextService,
                                kProfileGuid, nullptr, TF_IPPMF_FORSESSION);
    }
    mgr->Release();
  }
  return hr;
}

HRESULT ActivateShurufaProfile() {
  PrepareEditForInputSwitch(false);
  HRESULT hr = S_OK;
  for (int attempt = 0; attempt < 5; ++attempt) {
    hr = ActivateShurufaProfileOnce();
    if (SUCCEEDED(hr) && IsShurufaActive()) {
      PrepareEditForInputSwitch(false);
      return S_OK;
    }
    Sleep(120);
  }
  return SUCCEEDED(hr) ? HRESULT_FROM_WIN32(ERROR_RETRY) : hr;
}

void UpdateImeStatus(HWND hwnd, HRESULT hr) {
  if (SUCCEEDED(hr)) {
    g_shurufaActive = true;
    StringCchCopyW(g_imeStatus, ARRAYSIZE(g_imeStatus),
                   L"shurufa233 active in this lab");
  } else {
    g_shurufaActive = false;
    StringCchPrintfW(g_imeStatus, ARRAYSIZE(g_imeStatus),
                     L"F6 activation failed: 0x%08X", static_cast<unsigned int>(hr));
  }
  if (g_edit) {
    SetFocus(g_edit);
  }
  InvalidateRect(hwnd, nullptr, FALSE);
}

void RoundedFill(HDC dc, RECT rect, COLORREF fill, COLORREF border, int radius) {
  HBRUSH brush = CreateSolidBrush(fill);
  HPEN pen = CreatePen(PS_SOLID, 1, border);
  HGDIOBJ oldBrush = SelectObject(dc, brush);
  HGDIOBJ oldPen = SelectObject(dc, pen);
  RoundRect(dc, rect.left, rect.top, rect.right, rect.bottom, radius, radius);
  SelectObject(dc, oldPen);
  SelectObject(dc, oldBrush);
  DeleteObject(pen);
  DeleteObject(brush);
}

void FillSolidRect(HDC dc, RECT rect, COLORREF fill) {
  HBRUSH brush = CreateSolidBrush(fill);
  FillRect(dc, &rect, brush);
  DeleteObject(brush);
}

void DrawTextLine(HDC dc, const wchar_t *text, RECT rect, HFONT font, COLORREF color,
                  UINT format = DT_SINGLELINE | DT_VCENTER | DT_LEFT) {
  HGDIOBJ oldFont = SelectObject(dc, font);
  SetTextColor(dc, color);
  SetBkMode(dc, TRANSPARENT);
  DrawTextW(dc, text, -1, &rect, format);
  SelectObject(dc, oldFont);
}

void DrawMetric(HDC dc, RECT rect, const wchar_t *label, const wchar_t *value, COLORREF accent) {
  RoundedFill(dc, rect, Rgb(255, 255, 255), Rgb(216, 225, 238), 14);
  RECT accentRect{rect.left + 12, rect.top + 12, rect.left + 17, rect.bottom - 12};
  RoundedFill(dc, accentRect, accent, accent, 5);
  RECT labelRect{rect.left + 28, rect.top + 8, rect.right - 14, rect.top + 30};
  RECT valueRect{rect.left + 28, rect.top + 30, rect.right - 14, rect.bottom - 8};
  DrawTextLine(dc, label, labelRect, g_bodyFont, Rgb(91, 103, 122));
  DrawTextLine(dc, value, valueRect, g_titleFont, accent);
}

void DrawBadge(HDC dc, RECT rect, const wchar_t *text, COLORREF fill, COLORREF border,
               COLORREF color) {
  RoundedFill(dc, rect, fill, border, 16);
  DrawTextLine(dc, text, rect, g_bodyFont, color,
               DT_SINGLELINE | DT_VCENTER | DT_CENTER);
}

void DrawSegment(HDC dc, RECT rect, const wchar_t *label, const wchar_t *value,
                 const wchar_t *hint, COLORREF accent) {
  RECT marker{rect.left, rect.top + 8, rect.left + 4, rect.bottom - 8};
  RoundedFill(dc, marker, accent, accent, 4);
  RECT labelRect{rect.left + 14, rect.top + 4, rect.right - 6, rect.top + 24};
  RECT valueRect{rect.left + 14, rect.top + 22, rect.right - 6, rect.top + 50};
  RECT hintRect{rect.left + 14, rect.top + 49, rect.right - 6, rect.bottom - 2};
  DrawTextLine(dc, label, labelRect, g_bodyFont, Rgb(100, 116, 139));
  DrawTextLine(dc, value, valueRect, g_titleFont, accent);
  DrawTextLine(dc, hint, hintRect, g_bodyFont, Rgb(107, 114, 128));
}

void DrawBar(HDC dc, RECT rect, double value, double excellentMax, double warnMax) {
  RoundedFill(dc, rect, Rgb(232, 238, 248), Rgb(232, 238, 248), 8);
  const double clamped = std::clamp(value / warnMax, 0.0, 1.0);
  RECT fill = rect;
  fill.right = fill.left + static_cast<int>((fill.right - fill.left) * clamped);
  const COLORREF color = value <= excellentMax ? Rgb(5, 150, 105)
                         : value <= warnMax   ? Rgb(217, 119, 6)
                                              : Rgb(220, 38, 38);
  if (fill.right > fill.left) {
    RoundedFill(dc, fill, color, color, 8);
  }
}

void DrawSparkline(HDC dc, RECT rect) {
  RoundedFill(dc, rect, Rgb(248, 250, 252), Rgb(226, 232, 240), 10);
  RECT title{rect.left + 12, rect.top + 8, rect.right - 12, rect.top + 30};
  DrawTextLine(dc, L"延迟火花线", title, g_bodyFont, Rgb(71, 85, 105));

  RECT plot{rect.left + 12, rect.top + 36, rect.right - 12, rect.bottom - 14};
  HPEN gridPen = CreatePen(PS_SOLID, 1, Rgb(226, 232, 240));
  HGDIOBJ oldPen = SelectObject(dc, gridPen);
  for (int i = 0; i < 3; ++i) {
    const int y = plot.top + ((plot.bottom - plot.top) * (i + 1)) / 4;
    MoveToEx(dc, plot.left, y, nullptr);
    LineTo(dc, plot.right, y);
  }
  SelectObject(dc, oldPen);
  DeleteObject(gridPen);

  if (g_metrics.latencyCount <= 1) {
    DrawTextLine(dc, L"输入并上屏后开始采样", plot, g_bodyFont, Rgb(148, 163, 184),
                 DT_SINGLELINE | DT_CENTER | DT_VCENTER);
    return;
  }

  HPEN linePen = CreatePen(PS_SOLID, 2, Rgb(37, 99, 235));
  oldPen = SelectObject(dc, linePen);
  const int count = g_metrics.latencyCount;
  const int start = count == kLatencyWindow ? g_metrics.latencyCursor : 0;
  for (int i = 0; i < count; ++i) {
    const int source = (start + i) % kLatencyWindow;
    const double sample = std::clamp(g_metrics.latencyWindow[source], 0.0, 50.0);
    const int x = plot.left + ((plot.right - plot.left) * i) / std::max(1, count - 1);
    const int y = plot.bottom - static_cast<int>((plot.bottom - plot.top) * (sample / 50.0));
    if (i == 0) {
      MoveToEx(dc, x, y, nullptr);
    } else {
      LineTo(dc, x, y);
    }
  }
  SelectObject(dc, oldPen);
  DeleteObject(linePen);
}

void DrawKeyTrail(HDC dc, RECT rect) {
  RoundedFill(dc, rect, Rgb(248, 250, 252), Rgb(226, 232, 240), 10);
  RECT labelRect{rect.left + 12, rect.top + 8, rect.left + 112, rect.bottom - 8};
  DrawTextLine(dc, L"最近按键", labelRect, g_bodyFont, Rgb(71, 85, 105));
  if (g_metrics.recentKeyLabelCount <= 0) {
    RECT empty{rect.left + 112, rect.top + 8, rect.right - 12, rect.bottom - 8};
    DrawTextLine(dc, L"等待输入", empty, g_bodyFont, Rgb(148, 163, 184));
    return;
  }

  int x = rect.left + 112;
  const int y = rect.top + 9;
  const int count = g_metrics.recentKeyLabelCount;
  const int start = count == kRecentKeyLabels ? g_metrics.recentKeyLabelCursor : 0;
  HGDIOBJ oldFont = SelectObject(dc, g_bodyFont);
  for (int i = 0; i < count; ++i) {
    const int source = (start + i) % kRecentKeyLabels;
    const wchar_t *text = g_metrics.recentKeyLabels[source];
    SIZE size{};
    GetTextExtentPoint32W(dc, text, static_cast<int>(wcslen(text)), &size);
    const int width = std::clamp(static_cast<int>(size.cx) + 22, 44, 104);
    if (x + width > rect.right - 12) {
      break;
    }
    RECT chip{x, y, x + width, rect.bottom - 9};
    DrawBadge(dc, chip, text, Rgb(239, 246, 255), Rgb(191, 219, 254), Rgb(29, 78, 216));
    x += width + 8;
  }
  SelectObject(dc, oldFont);
}

void Paint(HWND hwnd) {
  PAINTSTRUCT ps{};
  HDC paintDc = BeginPaint(hwnd, &ps);
  RECT client{};
  GetClientRect(hwnd, &client);
  HDC dc = CreateCompatibleDC(paintDc);
  HBITMAP bitmap = CreateCompatibleBitmap(paintDc, client.right - client.left,
                                          client.bottom - client.top);
  HGDIOBJ oldBitmap = SelectObject(dc, bitmap);

  HBRUSH bg = CreateSolidBrush(Rgb(243, 246, 251));
  FillRect(dc, &client, bg);
  DeleteObject(bg);

  RECT hero{24, 20, client.right - 24, 132};
  RoundedFill(dc, hero, Rgb(18, 27, 42), Rgb(44, 57, 80), 18);
  RECT heroAccent{hero.left, hero.top, hero.left + 7, hero.bottom};
  FillSolidRect(dc, heroAccent, Rgb(37, 99, 235));
  RECT title{hero.left + 26, hero.top + 14, hero.right - 230, hero.top + 46};
  DrawTextLine(dc, L"shurufa233 电竞输入性能实验室", title, g_titleFont, Rgb(255, 255, 255));
  RECT subtitle{hero.left + 26, hero.top + 50, hero.right - 26, hero.top + 78};
  DrawTextLine(dc, L"真实 Win32 EDIT + TSF 输入链路，验证键盘触发、候选上屏、Ctrl+Shift 共存和低延迟节奏",
               subtitle, g_bodyFont, Rgb(191, 219, 254));
  RECT badges{hero.left + 26, hero.top + 84, hero.right - 26, hero.bottom - 14};
  RECT badge{badges.left, badges.top, badges.left + 116, badges.bottom};
  DrawBadge(dc, badge, L"F5 重置", Rgb(30, 41, 59), Rgb(71, 85, 105), Rgb(226, 232, 240));
  OffsetRect(&badge, 126, 0);
  DrawBadge(dc, badge, L"F6 激活本输入法", Rgb(30, 41, 59), Rgb(71, 85, 105),
            Rgb(226, 232, 240));
  RECT statusBadge{hero.right - 206, hero.top + 18, hero.right - 24, hero.top + 48};
  DrawBadge(dc, statusBadge, g_shurufaActive ? L"shurufa233 active" : L"Microsoft 可共存",
            g_shurufaActive ? Rgb(6, 95, 70) : Rgb(51, 65, 85),
            g_shurufaActive ? Rgb(52, 211, 153) : Rgb(100, 116, 139),
            Rgb(240, 253, 250));

  LARGE_INTEGER now = Now();
  const double elapsed = g_metrics.started ? std::max(0.001, MsSince(g_metrics.startedAt, now) / 1000.0) : 0.0;
  const double wpm = elapsed > 0 ? (static_cast<double>(g_metrics.textLength) / 5.0) / (elapsed / 60.0) : 0.0;
  const double kps = elapsed > 0 ? static_cast<double>(g_metrics.keyDowns) / elapsed : 0.0;
  const double avgLatency = g_metrics.latencySamples > 0
                                ? g_metrics.latencyTotalMs / g_metrics.latencySamples
                                : 0.0;
  const double p95Latency = PercentileLatency(0.95);
  const int compositionCycles = std::max(0, g_metrics.imeEnds);
  const bool latencyExcellent = g_metrics.latencySamples > 0 && p95Latency <= 16.0;
  const bool latencyGood = g_metrics.latencySamples > 0 && p95Latency <= 33.0;

  wchar_t value[64]{};
  const int cardTop = 150;
  const int cardHeight = 74;
  const int gap = 12;
  const int cardWidth = std::max(120, (static_cast<int>(client.right) - 48 - gap * 5) / 6);
  RECT card{24, cardTop, 24 + cardWidth, cardTop + cardHeight};

  StringCchPrintfW(value, ARRAYSIZE(value), L"%.1f", wpm);
  DrawMetric(dc, card, L"WPM", value, Rgb(37, 99, 235));
  OffsetRect(&card, cardWidth + gap, 0);
  StringCchPrintfW(value, ARRAYSIZE(value), L"%.1f", kps);
  DrawMetric(dc, card, L"Keys/s", value, Rgb(5, 150, 105));
  OffsetRect(&card, cardWidth + gap, 0);
  StringCchPrintfW(value, ARRAYSIZE(value), L"%.1f ms", avgLatency);
  DrawMetric(dc, card, L"Avg latency", value, Rgb(124, 58, 237));
  OffsetRect(&card, cardWidth + gap, 0);
  StringCchPrintfW(value, ARRAYSIZE(value), L"%d", g_metrics.imeStarts - g_metrics.imeEnds);
  DrawMetric(dc, card, L"IME state", value, Rgb(217, 119, 6));
  OffsetRect(&card, cardWidth + gap, 0);
  StringCchPrintfW(value, ARRAYSIZE(value), L"%d", g_metrics.textLength);
  DrawMetric(dc, card, L"Chars", value, Rgb(220, 38, 38));
  OffsetRect(&card, cardWidth + gap, 0);
  StringCchPrintfW(value, ARRAYSIZE(value), L"%d", g_metrics.changes);
  DrawMetric(dc, card, L"Text changes", value, Rgb(8, 145, 178));

  RECT perfPanel{24, 244, client.right - 24, 400};
  RoundedFill(dc, perfPanel, Rgb(255, 255, 255), Rgb(211, 219, 232), 16);
  RECT perfTitle{perfPanel.left + 18, perfPanel.top + 10, perfPanel.right - 18, perfPanel.top + 32};
  DrawTextLine(dc, L"电竞性能雷达", perfTitle, g_bodyFont, Rgb(55, 65, 81));
  const int segmentTop = perfPanel.top + 34;
  const int segmentGap = 16;
  const int segmentWidth = std::max(150, (static_cast<int>(perfPanel.right - perfPanel.left) - 36 - segmentGap * 3) / 4);
  RECT segment{perfPanel.left + 18, segmentTop, perfPanel.left + 18 + segmentWidth, perfPanel.bottom - 58};
  StringCchPrintfW(value, ARRAYSIZE(value), L"%.1f ms", p95Latency);
  DrawSegment(dc, segment, L"P95 latency", value,
              g_metrics.latencySamples > 0 ? L"越低越适合高速连击" : L"等待候选上屏样本",
              latencyExcellent ? Rgb(5, 150, 105) : Rgb(217, 119, 6));
  OffsetRect(&segment, segmentWidth + segmentGap, 0);
  StringCchPrintfW(value, ARRAYSIZE(value), L"%.0f keys/s", g_metrics.peakKeysPerSecond);
  DrawSegment(dc, segment, L"Peak burst", value, L"1 秒滑窗峰值", Rgb(37, 99, 235));
  OffsetRect(&segment, segmentWidth + segmentGap, 0);
  StringCchPrintfW(value, ARRAYSIZE(value), L"%d cycles", compositionCycles);
  DrawSegment(dc, segment, L"IME compose", value, L"候选生命周期", Rgb(124, 58, 237));
  OffsetRect(&segment, segmentWidth + segmentGap, 0);
  DrawSegment(dc, segment, L"Stability",
              g_metrics.latencySamples == 0 ? L"Standby" : (latencyGood ? L"Ready" : L"Watch"),
              g_shurufaActive ? L"TSF profile active" : L"F6 激活后再测",
              latencyGood ? Rgb(5, 150, 105) : Rgb(220, 38, 38));

  RECT bar{perfPanel.left + 18, perfPanel.bottom - 48, perfPanel.left + segmentWidth - 8,
           perfPanel.bottom - 36};
  DrawBar(dc, bar, g_metrics.latencySamples > 0 ? p95Latency : 0.0, 16.0, 50.0);
  RECT spark{perfPanel.left + 18 + segmentWidth + segmentGap, perfPanel.bottom - 46,
             perfPanel.right - 18, perfPanel.bottom - 10};
  DrawKeyTrail(dc, spark);

  RECT sparkPanel{24, 414, client.right - 24, 520};
  DrawSparkline(dc, sparkPanel);

  RECT editFrame{24, 536, client.right - 24, client.bottom - 24};
  RoundedFill(dc, editFrame, Rgb(255, 255, 255), Rgb(211, 219, 232), 16);
  RECT editTitle{editFrame.left + 18, editFrame.top + 12, editFrame.right - 18, editFrame.top + 42};
  DrawTextLine(dc, L"原生输入轨道", editTitle, g_titleFont, Rgb(31, 41, 55));
  RECT hint{editFrame.left + 128, editFrame.top + 14, editFrame.right - 18, editFrame.top + 40};
  DrawTextLine(dc, L"建议测试：nihao / shurufa / zan / kaixin / shengluehao / 12345 连续高速输入",
               hint, g_bodyFont, Rgb(100, 116, 139),
               DT_SINGLELINE | DT_VCENTER | DT_RIGHT);
  RECT imeHint{editFrame.left + 18, editFrame.top + 44, editFrame.right - 18, editFrame.top + 70};
  DrawTextLine(dc, g_imeStatus, imeHint, g_bodyFont,
               g_shurufaActive ? Rgb(5, 150, 105) : Rgb(100, 116, 139));
  RECT divider{editFrame.left + 18, editFrame.top + 78, editFrame.right - 18, editFrame.top + 79};
  FillSolidRect(dc, divider, Rgb(229, 235, 245));

  BitBlt(paintDc, client.left, client.top, client.right - client.left, client.bottom - client.top,
         dc, 0, 0, SRCCOPY);
  SelectObject(dc, oldBitmap);
  DeleteObject(bitmap);
  DeleteDC(dc);
  EndPaint(hwnd, &ps);
}

LRESULT CALLBACK EditProc(HWND hwnd, UINT message, WPARAM wparam, LPARAM lparam) {
  switch (message) {
    case WM_KEYDOWN:
      if (wparam == VK_F5 || wparam == VK_F6) {
        HWND parent = GetParent(hwnd);
        if (parent) {
          SendMessageW(parent, WM_KEYDOWN, wparam, lparam);
          return 0;
        }
      }
      if (!g_suppressInputMetrics) {
        RecordKeyDown(wparam, lparam);
      }
      break;
    case WM_CHAR:
      if (!g_suppressInputMetrics) {
        EnsureStarted();
        g_metrics.chars++;
      }
      break;
    case WM_IME_STARTCOMPOSITION:
      if (!g_suppressInputMetrics) {
        EnsureStarted();
        g_metrics.imeStarts++;
      }
      break;
    case WM_IME_ENDCOMPOSITION:
      if (!g_suppressInputMetrics) {
        g_metrics.imeEnds++;
      }
      break;
    default:
      break;
  }
  return CallWindowProcW(g_originalEditProc, hwnd, message, wparam, lparam);
}

void UpdateTextMetrics(HWND hwnd) {
  if (!g_edit) {
    return;
  }
  const int previousLength = g_metrics.textLength;
  const int nextLength = GetWindowTextLengthW(g_edit);
  g_metrics.textLength = nextLength;
  g_metrics.changes++;
  if (nextLength != previousLength && g_metrics.lastKeyAt.QuadPart != 0) {
    LARGE_INTEGER now = Now();
    const double latency = MsSince(g_metrics.lastKeyAt, now);
    if (latency >= 0.0 && latency < 1000.0) {
      RecordLatencySample(latency);
    }
  }
  InvalidateRect(hwnd, nullptr, FALSE);
}

void Layout(HWND hwnd) {
  if (!g_edit) {
    return;
  }
  RECT client{};
  GetClientRect(hwnd, &client);
  MoveWindow(g_edit, 46, kEditTop, std::max(120, static_cast<int>(client.right) - 92),
             std::max(90, static_cast<int>(client.bottom) - kEditTop - 48), TRUE);
}

LRESULT CALLBACK WindowProc(HWND hwnd, UINT message, WPARAM wparam, LPARAM lparam) {
  switch (message) {
    case WM_CREATE: {
      QueryPerformanceFrequency(&g_metrics.frequency);
      g_titleFont = CreateFontW(-22, 0, 0, 0, FW_SEMIBOLD, FALSE, FALSE, FALSE,
                                DEFAULT_CHARSET, OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS,
                                CLEARTYPE_QUALITY, DEFAULT_PITCH | FF_DONTCARE,
                                L"Microsoft YaHei UI");
      g_bodyFont = CreateFontW(-15, 0, 0, 0, FW_NORMAL, FALSE, FALSE, FALSE,
                               DEFAULT_CHARSET, OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS,
                               CLEARTYPE_QUALITY, DEFAULT_PITCH | FF_DONTCARE,
                               L"Microsoft YaHei UI");
      g_editFont = CreateFontW(-26, 0, 0, 0, FW_NORMAL, FALSE, FALSE, FALSE,
                               DEFAULT_CHARSET, OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS,
                               CLEARTYPE_QUALITY, DEFAULT_PITCH | FF_DONTCARE,
                               L"Microsoft YaHei UI");
      g_edit = CreateWindowExW(0, L"EDIT", L"",
                               WS_CHILD | WS_VISIBLE | WS_TABSTOP | ES_LEFT |
                                   ES_MULTILINE | ES_AUTOVSCROLL | WS_VSCROLL,
                               46, kEditTop, 760, 260, hwnd, reinterpret_cast<HMENU>(1),
                               reinterpret_cast<LPCREATESTRUCTW>(lparam)->hInstance, nullptr);
      SendMessageW(g_edit, WM_SETFONT, reinterpret_cast<WPARAM>(g_editFont), TRUE);
      SendMessageW(g_edit, EM_SETMARGINS, EC_LEFTMARGIN | EC_RIGHTMARGIN, MAKELPARAM(14, 14));
      g_originalEditProc = reinterpret_cast<WNDPROC>(
          SetWindowLongPtrW(g_edit, GWLP_WNDPROC, reinterpret_cast<LONG_PTR>(EditProc)));
      g_shurufaActive = IsShurufaActive();
      if (g_shurufaActive) {
        StringCchCopyW(g_imeStatus, ARRAYSIZE(g_imeStatus),
                       L"shurufa233 active in this lab");
      }
      SetTimer(hwnd, kStatsTimer, 100, nullptr);
      SetFocus(g_edit);
      return 0;
    }
    case WM_COMMAND:
      if (reinterpret_cast<HWND>(lparam) == g_edit && HIWORD(wparam) == EN_CHANGE) {
        UpdateTextMetrics(hwnd);
      }
      return 0;
    case WM_SIZE:
      Layout(hwnd);
      return 0;
    case WM_TIMER:
      if (wparam == kStatsTimer) {
        InvalidateRect(hwnd, nullptr, FALSE);
        return 0;
      }
      break;
    case WM_GETMINMAXINFO: {
      auto *info = reinterpret_cast<MINMAXINFO *>(lparam);
      info->ptMinTrackSize.x = 920;
      info->ptMinTrackSize.y = 800;
      return 0;
    }
    case WM_KEYDOWN:
      if (wparam == VK_F5) {
        ResetMetrics(hwnd);
        return 0;
      }
      if (wparam == VK_F6) {
        UpdateImeStatus(hwnd, ActivateShurufaProfile());
        return 0;
      }
      break;
    case WM_PAINT:
      Paint(hwnd);
      return 0;
    case WM_SETFOCUS:
      if (g_edit) {
        SetFocus(g_edit);
      }
      return 0;
    case WM_DESTROY:
      KillTimer(hwnd, kStatsTimer);
      if (g_titleFont) {
        DeleteObject(g_titleFont);
      }
      if (g_bodyFont) {
        DeleteObject(g_bodyFont);
      }
      if (g_editFont) {
        DeleteObject(g_editFont);
      }
      PostQuitMessage(0);
      return 0;
    default:
      return DefWindowProcW(hwnd, message, wparam, lparam);
  }
  return DefWindowProcW(hwnd, message, wparam, lparam);
}

}  // namespace

int WINAPI wWinMain(HINSTANCE instance, HINSTANCE, PWSTR, int show) {
  HANDLE singleton = CreateMutexW(nullptr, FALSE, kSingletonName);
  if (singleton && GetLastError() == ERROR_ALREADY_EXISTS) {
    HWND existing = FindWindowW(kClassName, nullptr);
    if (existing) {
      ShowWindow(existing, SW_RESTORE);
      SetForegroundWindow(existing);
      CloseHandle(singleton);
      return 0;
    }
  }

  HRESULT hr = CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED);
  const bool didCoInit = SUCCEEDED(hr);
  if (FAILED(hr) && hr != RPC_E_CHANGED_MODE) {
    if (singleton) {
      CloseHandle(singleton);
    }
    return 1;
  }

  WNDCLASSW wc{};
  wc.lpfnWndProc = WindowProc;
  wc.hInstance = instance;
  wc.hCursor = LoadCursorW(nullptr, IDC_ARROW);
  wc.hbrBackground = reinterpret_cast<HBRUSH>(COLOR_WINDOW + 1);
  wc.lpszClassName = kClassName;
  RegisterClassW(&wc);

  HWND hwnd = CreateWindowExW(0, kClassName, L"shurufa233 input performance lab",
                              WS_OVERLAPPEDWINDOW | WS_CLIPCHILDREN, CW_USEDEFAULT, CW_USEDEFAULT,
                              1040, 900, nullptr, nullptr, instance, nullptr);
  if (!hwnd) {
    if (didCoInit) {
      CoUninitialize();
    }
    if (singleton) {
      CloseHandle(singleton);
    }
    return 1;
  }
  ShowWindow(hwnd, show);
  UpdateWindow(hwnd);

  MSG msg{};
  while (GetMessageW(&msg, nullptr, 0, 0) > 0) {
    TranslateMessage(&msg);
    DispatchMessageW(&msg);
  }
  if (didCoInit) {
    CoUninitialize();
  }
  if (singleton) {
    CloseHandle(singleton);
  }
  return static_cast<int>(msg.wParam);
}
