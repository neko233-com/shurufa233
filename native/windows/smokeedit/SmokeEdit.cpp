#define NOMINMAX
#include <windows.h>
#include <strsafe.h>

#include <algorithm>

namespace {

constexpr wchar_t kClassName[] = L"Shurufa233SmokeEditWindow";
constexpr UINT_PTR kStatsTimer = 1;
constexpr int kEditTop = 300;

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
  bool started = false;
};

HWND g_edit = nullptr;
HFONT g_titleFont = nullptr;
HFONT g_bodyFont = nullptr;
HFONT g_editFont = nullptr;
WNDPROC g_originalEditProc = nullptr;
Metrics g_metrics{};

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

void ResetMetrics(HWND hwnd) {
  g_metrics = Metrics{};
  QueryPerformanceFrequency(&g_metrics.frequency);
  if (g_edit) {
    SetWindowTextW(g_edit, L"");
    SetFocus(g_edit);
  }
  InvalidateRect(hwnd, nullptr, TRUE);
}

void EnsureStarted() {
  if (!g_metrics.started) {
    g_metrics.startedAt = Now();
    g_metrics.started = true;
  }
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

void DrawTextLine(HDC dc, const wchar_t *text, RECT rect, HFONT font, COLORREF color,
                  UINT format = DT_SINGLELINE | DT_VCENTER | DT_LEFT) {
  HGDIOBJ oldFont = SelectObject(dc, font);
  SetTextColor(dc, color);
  SetBkMode(dc, TRANSPARENT);
  DrawTextW(dc, text, -1, &rect, format);
  SelectObject(dc, oldFont);
}

void DrawMetric(HDC dc, RECT rect, const wchar_t *label, const wchar_t *value, COLORREF accent) {
  RoundedFill(dc, rect, Rgb(255, 255, 255), Rgb(218, 225, 235), 14);
  RECT labelRect{rect.left + 14, rect.top + 8, rect.right - 14, rect.top + 28};
  RECT valueRect{rect.left + 14, rect.top + 28, rect.right - 14, rect.bottom - 8};
  DrawTextLine(dc, label, labelRect, g_bodyFont, Rgb(91, 103, 122));
  DrawTextLine(dc, value, valueRect, g_titleFont, accent);
}

void Paint(HWND hwnd) {
  PAINTSTRUCT ps{};
  HDC dc = BeginPaint(hwnd, &ps);
  RECT client{};
  GetClientRect(hwnd, &client);

  HBRUSH bg = CreateSolidBrush(Rgb(246, 248, 252));
  FillRect(dc, &client, bg);
  DeleteObject(bg);

  RECT hero{24, 20, client.right - 24, 104};
  RoundedFill(dc, hero, Rgb(20, 29, 43), Rgb(42, 55, 77), 18);
  RECT title{hero.left + 22, hero.top + 12, hero.right - 22, hero.top + 42};
  DrawTextLine(dc, L"shurufa233 输入性能验证", title, g_titleFont, Rgb(255, 255, 255));
  RECT subtitle{hero.left + 22, hero.top + 48, hero.right - 22, hero.bottom - 12};
  DrawTextLine(dc, L"Native Win32 EDIT / TSF path / Ctrl+Shift coexistence / low-latency typing smoke",
               subtitle, g_bodyFont, Rgb(170, 231, 218));

  LARGE_INTEGER now = Now();
  const double elapsed = g_metrics.started ? std::max(0.001, MsSince(g_metrics.startedAt, now) / 1000.0) : 0.0;
  const double wpm = elapsed > 0 ? (static_cast<double>(g_metrics.textLength) / 5.0) / (elapsed / 60.0) : 0.0;
  const double kps = elapsed > 0 ? static_cast<double>(g_metrics.keyDowns) / elapsed : 0.0;
  const double avgLatency = g_metrics.latencySamples > 0
                                ? g_metrics.latencyTotalMs / g_metrics.latencySamples
                                : 0.0;

  wchar_t value[64]{};
  const int cardTop = 120;
  const int cardHeight = 70;
  const int gap = 12;
  const int cardWidth = std::max(130, (static_cast<int>(client.right) - 48 - gap * 4) / 5);
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
  DrawMetric(dc, card, L"IME composing", value, Rgb(217, 119, 6));
  OffsetRect(&card, cardWidth + gap, 0);
  StringCchPrintfW(value, ARRAYSIZE(value), L"%d", g_metrics.textLength);
  DrawMetric(dc, card, L"Chars", value, Rgb(220, 38, 38));

  RECT editFrame{24, 222, client.right - 24, client.bottom - 24};
  RoundedFill(dc, editFrame, Rgb(255, 255, 255), Rgb(211, 219, 232), 16);
  RECT editTitle{editFrame.left + 18, editFrame.top + 12, editFrame.right - 18, editFrame.top + 42};
  DrawTextLine(dc, L"输入区", editTitle, g_titleFont, Rgb(31, 41, 55));
  RECT hint{editFrame.left + 100, editFrame.top + 14, editFrame.right - 18, editFrame.top + 40};
  DrawTextLine(dc, L"F5 reset", hint, g_bodyFont, Rgb(100, 116, 139),
               DT_SINGLELINE | DT_VCENTER | DT_RIGHT);

  EndPaint(hwnd, &ps);
}

LRESULT CALLBACK EditProc(HWND hwnd, UINT message, WPARAM wparam, LPARAM lparam) {
  switch (message) {
    case WM_KEYDOWN:
      EnsureStarted();
      g_metrics.keyDowns++;
      g_metrics.lastKeyAt = Now();
      break;
    case WM_CHAR:
      EnsureStarted();
      g_metrics.chars++;
      break;
    case WM_IME_STARTCOMPOSITION:
      EnsureStarted();
      g_metrics.imeStarts++;
      break;
    case WM_IME_ENDCOMPOSITION:
      g_metrics.imeEnds++;
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
      g_metrics.latencyTotalMs += latency;
      g_metrics.latencySamples++;
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
    case WM_KEYDOWN:
      if (wparam == VK_F5) {
        ResetMetrics(hwnd);
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
  WNDCLASSW wc{};
  wc.lpfnWndProc = WindowProc;
  wc.hInstance = instance;
  wc.hCursor = LoadCursorW(nullptr, IDC_ARROW);
  wc.hbrBackground = reinterpret_cast<HBRUSH>(COLOR_WINDOW + 1);
  wc.lpszClassName = kClassName;
  RegisterClassW(&wc);

  HWND hwnd = CreateWindowExW(0, kClassName, L"shurufa233 input performance lab",
                              WS_OVERLAPPEDWINDOW | WS_CLIPCHILDREN, CW_USEDEFAULT, CW_USEDEFAULT,
                              980, 700, nullptr, nullptr, instance, nullptr);
  if (!hwnd) {
    return 1;
  }
  ShowWindow(hwnd, show);
  UpdateWindow(hwnd);

  MSG msg{};
  while (GetMessageW(&msg, nullptr, 0, 0) > 0) {
    TranslateMessage(&msg);
    DispatchMessageW(&msg);
  }
  return static_cast<int>(msg.wParam);
}
