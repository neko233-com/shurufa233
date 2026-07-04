#include <windows.h>
#include <dwmapi.h>
#include <msctf.h>
#include <strsafe.h>
#include <winhttp.h>

#include <cstdint>
#include <cstdio>
#include <string>
#include <vector>

namespace {

// {3D7B8D06-9872-4C31-B77D-3B87327CBF64}
const CLSID kClsidTextService = {
    0x3d7b8d06,
    0x9872,
    0x4c31,
    {0xb7, 0x7d, 0x3b, 0x87, 0x32, 0x7c, 0xbf, 0x64}};

// {B68911A2-4478-491C-A624-978441648E20}
const GUID kProfileGuid = {
    0xb68911a2,
    0x4478,
    0x491c,
    {0xa6, 0x24, 0x97, 0x84, 0x41, 0x64, 0x8e, 0x20}};

constexpr wchar_t kDescription[] = L"shurufa233";
constexpr wchar_t kModel[] = L"Apartment";
constexpr LANGID kLanguage = MAKELANGID(LANG_CHINESE, SUBLANG_CHINESE_SIMPLIFIED);

long g_dllRefCount = 0;
HINSTANCE g_instance = nullptr;
volatile LONG64 g_nextSessionId = 1000;

using CreateSessionFn = uint64_t (*)();
using DestroySessionFn = void (*)(uint64_t);
using InputKeyFastFn = int (*)(uint64_t, char);
using BackspaceFastFn = int (*)(uint64_t);
using CandidateCountFn = int (*)(uint64_t);
using CandidateTextFn = char *(*)(uint64_t, int);
using CandidateReadingFn = char *(*)(uint64_t, int);
using CandidateScoreFn = int (*)(uint64_t, int);
using CandidatePayloadFn = char *(*)(uint64_t, int);
using ClearSessionFn = char *(*)(uint64_t);
using CommitCandidateFn = char *(*)(uint64_t, int);
using FreeFn = void (*)(char *);

struct CoreApi {
  bool initialized = false;
  bool inProcess = false;
  CreateSessionFn createSession = nullptr;
  DestroySessionFn destroySession = nullptr;
  InputKeyFastFn inputKeyFast = nullptr;
  BackspaceFastFn backspaceFast = nullptr;
  CandidateCountFn candidateCount = nullptr;
  CandidateTextFn candidateText = nullptr;
  CandidateReadingFn candidateReading = nullptr;
  CandidateScoreFn candidateScore = nullptr;
  CandidatePayloadFn candidatePayload = nullptr;
  ClearSessionFn clearSession = nullptr;
  CommitCandidateFn commitCandidate = nullptr;
  FreeFn freeValue = nullptr;

  bool Ready() const {
    return initialized && createSession && destroySession && inputKeyFast &&
           backspaceFast && candidateCount && candidateText && candidateReading &&
           candidateScore && clearSession && commitCandidate && freeValue;
  }
};

CoreApi g_core;
HMODULE g_coreModule = nullptr;
HINTERNET g_httpSession = nullptr;
HINTERNET g_httpConnect = nullptr;
CRITICAL_SECTION g_httpLock;
bool g_httpLockReady = false;

void LogLine(const wchar_t *message) {
  wchar_t path[MAX_PATH]{};
  if (!GetEnvironmentVariableW(L"LOCALAPPDATA", path, ARRAYSIZE(path))) {
    GetTempPathW(ARRAYSIZE(path), path);
  }
  StringCchCatW(path, ARRAYSIZE(path), L"\\shurufa233-tsf.log");

  HANDLE file = CreateFileW(path, FILE_APPEND_DATA, FILE_SHARE_READ | FILE_SHARE_WRITE,
                            nullptr, OPEN_ALWAYS, FILE_ATTRIBUTE_NORMAL, nullptr);
  if (file == INVALID_HANDLE_VALUE) {
    return;
  }

  SYSTEMTIME st{};
  GetLocalTime(&st);
  wchar_t line[1024]{};
  StringCchPrintfW(line, ARRAYSIZE(line),
                   L"%04u-%02u-%02u %02u:%02u:%02u.%03u %s\r\n",
                   st.wYear, st.wMonth, st.wDay, st.wHour, st.wMinute, st.wSecond,
                   st.wMilliseconds, message);
  char utf8[2048]{};
  const int len = WideCharToMultiByte(CP_UTF8, 0, line, -1, utf8, sizeof(utf8), nullptr, nullptr);
  DWORD bytes = 0;
  if (len > 1) {
    WriteFile(file, utf8, static_cast<DWORD>(len - 1), &bytes, nullptr);
  }
  CloseHandle(file);
}

void AddDllRef() {
  InterlockedIncrement(&g_dllRefCount);
}

void ReleaseDllRef() {
  InterlockedDecrement(&g_dllRefCount);
}

std::wstring ModuleDir() {
  wchar_t modulePath[MAX_PATH]{};
  GetModuleFileNameW(g_instance, modulePath, ARRAYSIZE(modulePath));
  wchar_t *slash = wcsrchr(modulePath, L'\\');
  if (slash) {
    *(slash + 1) = L'\0';
  }
  return modulePath;
}

std::wstring ModuleFileName() {
  wchar_t modulePath[MAX_PATH]{};
  GetModuleFileNameW(g_instance, modulePath, ARRAYSIZE(modulePath));
  const wchar_t *slash = wcsrchr(modulePath, L'\\');
  return slash ? std::wstring(slash + 1) : std::wstring(modulePath);
}

bool EnsureHttpHandles() {
  if (!g_httpLockReady) {
    InitializeCriticalSection(&g_httpLock);
    g_httpLockReady = true;
  }
  EnterCriticalSection(&g_httpLock);
  if (!g_httpSession) {
    g_httpSession = WinHttpOpen(L"shurufa233-tsf/0.1", WINHTTP_ACCESS_TYPE_NO_PROXY,
                                WINHTTP_NO_PROXY_NAME, WINHTTP_NO_PROXY_BYPASS, 0);
  }
  if (g_httpSession && !g_httpConnect) {
    g_httpConnect = WinHttpConnect(g_httpSession, L"127.0.0.1", 23333, 0);
  }
  const bool ok = g_httpSession && g_httpConnect;
  LeaveCriticalSection(&g_httpLock);
  return ok;
}

std::string HttpRequest(const wchar_t *verb, const std::wstring &path) {
  if (!EnsureHttpHandles()) {
    LogLine(L"HttpRequest skipped: handles unavailable");
    return "";
  }
  EnterCriticalSection(&g_httpLock);
  HINTERNET request = WinHttpOpenRequest(g_httpConnect, verb, path.c_str(), nullptr,
                                         WINHTTP_NO_REFERER, WINHTTP_DEFAULT_ACCEPT_TYPES, 0);
  LeaveCriticalSection(&g_httpLock);
  if (!request) {
    wchar_t message[192]{};
    StringCchPrintfW(message, ARRAYSIZE(message), L"HttpRequest open failed error=%lu path=%s",
                     GetLastError(), path.c_str());
    LogLine(message);
    return "";
  }

  std::string response;
  BOOL ok = WinHttpSendRequest(request, WINHTTP_NO_ADDITIONAL_HEADERS, 0,
                               WINHTTP_NO_REQUEST_DATA, 0, 0, 0);
  if (ok) {
    ok = WinHttpReceiveResponse(request, nullptr);
  }
  DWORD status = 0;
  if (ok) {
    DWORD statusSize = sizeof(status);
    WinHttpQueryHeaders(request, WINHTTP_QUERY_STATUS_CODE | WINHTTP_QUERY_FLAG_NUMBER,
                        WINHTTP_HEADER_NAME_BY_INDEX, &status, &statusSize, WINHTTP_NO_HEADER_INDEX);
    if (status >= 200 && status < 300) {
      DWORD available = 0;
      while (WinHttpQueryDataAvailable(request, &available) && available > 0) {
        std::string chunk(available, '\0');
        DWORD read = 0;
        if (!WinHttpReadData(request, chunk.data(), available, &read) || read == 0) {
          break;
        }
        chunk.resize(read);
        response += chunk;
      }
    }
  }
  if (!ok || status < 200 || status >= 300) {
    wchar_t message[256]{};
    StringCchPrintfW(message, ARRAYSIZE(message),
                     L"HttpRequest failed ok=%d status=%lu error=%lu path=%s",
                     ok ? 1 : 0, status, GetLastError(), path.c_str());
    LogLine(message);
  }
  WinHttpCloseHandle(request);
  return response;
}

char *AllocCString(const std::string &value) {
  char *out = static_cast<char *>(CoTaskMemAlloc(value.size() + 1));
  if (!out) {
    return nullptr;
  }
  memcpy(out, value.data(), value.size());
  out[value.size()] = '\0';
  return out;
}

uint64_t HttpCreateSession() {
  return static_cast<uint64_t>(InterlockedIncrement64(&g_nextSessionId));
}

void HttpDestroySession(uint64_t) {}

int HttpInputKeyFast(uint64_t session, char key) {
  wchar_t path[64]{};
  StringCchPrintfW(path, ARRAYSIZE(path), L"/ime/key?key=%c&session=%llu",
                   static_cast<wchar_t>(key), session);
  HttpRequest(L"POST", path);
  return -1;
}

int HttpBackspaceFast(uint64_t session) {
  wchar_t path[64]{};
  StringCchPrintfW(path, ARRAYSIZE(path), L"/ime/backspace?session=%llu", session);
  HttpRequest(L"POST", path);
  return -1;
}

int HttpCandidateCount(uint64_t session) {
  wchar_t path[64]{};
  StringCchPrintfW(path, ARRAYSIZE(path), L"/ime/count?session=%llu", session);
  std::string value = HttpRequest(L"GET", path);
  return value.empty() ? 0 : atoi(value.c_str());
}

char *HttpCandidateText(uint64_t, int) {
  return AllocCString("");
}

char *HttpCandidateReading(uint64_t, int) {
  return AllocCString("");
}

int HttpCandidateScore(uint64_t, int) {
  return 0;
}

char *HttpClearSessionValue(uint64_t session) {
  wchar_t path[64]{};
  StringCchPrintfW(path, ARRAYSIZE(path), L"/ime/clear?session=%llu", session);
  return AllocCString(HttpRequest(L"POST", path));
}

char *HttpCommitCandidate(uint64_t session, int index) {
  wchar_t path[64]{};
  StringCchPrintfW(path, ARRAYSIZE(path), L"/ime/select?index=%d&session=%llu", index, session);
  return AllocCString(HttpRequest(L"POST", path));
}

void HttpFree(char *value) {
  CoTaskMemFree(value);
}

template <typename Fn>
Fn LoadCoreProc(HMODULE module, const char *name) {
  return reinterpret_cast<Fn>(GetProcAddress(module, name));
}

bool TryLoadInProcessCore() {
  if (g_coreModule) {
    return true;
  }

  std::vector<std::wstring> corePaths;
  const std::wstring moduleDir = ModuleDir();
  const std::wstring moduleName = ModuleFileName();
  constexpr wchar_t tsfPrefix[] = L"Shurufa233Tsf-";
  if (moduleName.rfind(tsfPrefix, 0) == 0 && moduleName.size() > wcslen(tsfPrefix)) {
    corePaths.push_back(moduleDir + L"shurufa_core-" + moduleName.substr(wcslen(tsfPrefix)));
  }
  corePaths.push_back(moduleDir + L"shurufa_core.dll");

  HMODULE module = nullptr;
  std::wstring loadedPath;
  for (const std::wstring &corePath : corePaths) {
    module = LoadLibraryW(corePath.c_str());
    if (module) {
      loadedPath = corePath;
      break;
    }
    wchar_t message[256]{};
    StringCchPrintfW(message, ARRAYSIZE(message),
                     L"In-process core unavailable error=%lu path=%s",
                     GetLastError(), corePath.c_str());
    LogLine(message);
  }
  if (!module) {
    return false;
  }

  CoreApi api{};
  api.initialized = true;
  api.inProcess = true;
  api.createSession = LoadCoreProc<CreateSessionFn>(module, "ShurufaCreateSession");
  api.destroySession = LoadCoreProc<DestroySessionFn>(module, "ShurufaDestroySession");
  api.inputKeyFast = LoadCoreProc<InputKeyFastFn>(module, "ShurufaInputKeyFast");
  api.backspaceFast = LoadCoreProc<BackspaceFastFn>(module, "ShurufaBackspaceFast");
  api.candidateCount = LoadCoreProc<CandidateCountFn>(module, "ShurufaCandidateCount");
  api.candidateText = LoadCoreProc<CandidateTextFn>(module, "ShurufaCandidateText");
  api.candidateReading = LoadCoreProc<CandidateReadingFn>(module, "ShurufaCandidateReading");
  api.candidateScore = LoadCoreProc<CandidateScoreFn>(module, "ShurufaCandidateScore");
  api.candidatePayload = LoadCoreProc<CandidatePayloadFn>(module, "ShurufaCandidatePayload");
  api.clearSession = LoadCoreProc<ClearSessionFn>(module, "ShurufaClear");
  api.commitCandidate = LoadCoreProc<CommitCandidateFn>(module, "ShurufaCommitCandidate");
  api.freeValue = LoadCoreProc<FreeFn>(module, "ShurufaFree");
  if (!api.Ready()) {
    LogLine(L"In-process core missing required exports; falling back to daemon IPC");
    FreeLibrary(module);
    return false;
  }

  g_coreModule = module;
  g_core = api;
  wchar_t message[256]{};
  StringCchPrintfW(message, ARRAYSIZE(message), L"In-process Go core loaded path=%s",
                   loadedPath.c_str());
  LogLine(message);
  return true;
}

void UseHttpCoreFallback() {
  g_core.initialized = true;
  g_core.inProcess = false;
  g_core.createSession = HttpCreateSession;
  g_core.destroySession = HttpDestroySession;
  g_core.inputKeyFast = HttpInputKeyFast;
  g_core.backspaceFast = HttpBackspaceFast;
  g_core.candidateCount = HttpCandidateCount;
  g_core.candidateText = HttpCandidateText;
  g_core.candidateReading = HttpCandidateReading;
  g_core.candidateScore = HttpCandidateScore;
  g_core.candidatePayload = nullptr;
  g_core.clearSession = HttpClearSessionValue;
  g_core.commitCandidate = HttpCommitCandidate;
  g_core.freeValue = HttpFree;
}

bool EnsureCoreLoaded() {
  if (g_core.Ready()) {
    return true;
  }
  if (TryLoadInProcessCore()) {
    return true;
  }
  UseHttpCoreFallback();
  LogLine(L"Using daemon HTTP core fallback");
  return g_core.Ready();
}

std::wstring Utf8ToWide(const char *value) {
  if (!value || !*value) {
    return L"";
  }
  const int len = MultiByteToWideChar(CP_UTF8, 0, value, -1, nullptr, 0);
  if (len <= 1) {
    return L"";
  }
  std::wstring wide(static_cast<size_t>(len - 1), L'\0');
  MultiByteToWideChar(CP_UTF8, 0, value, -1, wide.data(), len);
  return wide;
}

class CandidateWindow {
 public:
  struct CandidateView {
    int index = 0;
    std::wstring text;
    std::wstring reading;
    int score = 0;
  };

  ~CandidateWindow() {
    ResetFont();
  }

  void Show(const std::string &payload, int selectedIndex) {
    candidates_ = ParseCandidates(payload);
    if (candidates_.empty()) {
      Hide();
      return;
    }
    if (hwnd_) {
      KillTimer(hwnd_, kStatusTimerId);
    }
    selectedIndex_ = max(0, min(selectedIndex, static_cast<int>(candidates_.size()) - 1));
    EnsureWindow();
    RefreshSkin();

    POINT anchor = CaretAnchor();
    const int width = MeasureWindowWidth();
    const int height = CandidateWindowHeight();
    const POINT origin = FitToWorkArea(anchor, width, height);
    SetWindowPos(hwnd_, HWND_TOPMOST, origin.x, origin.y, width, height,
                 SWP_NOACTIVATE | SWP_SHOWWINDOW);
    InvalidateRect(hwnd_, nullptr, TRUE);
  }

  void Hide() {
    if (hwnd_) {
      KillTimer(hwnd_, kStatusTimerId);
      ShowWindow(hwnd_, SW_HIDE);
    }
  }

  void ShowStatus(const wchar_t *text) {
    EnsureWindow();
    RefreshSkin();
    statusText_ = text ? text : L"";
    candidates_.clear();
    POINT anchor = CaretAnchor();
    const int width = MeasureStatusWidth();
    const int height = max(42, fontSize_ + 28);
    const POINT origin = FitToWorkArea(anchor, width, height);
    SetWindowPos(hwnd_, HWND_TOPMOST, origin.x, origin.y, width, height,
                 SWP_NOACTIVATE | SWP_SHOWWINDOW);
    SetTimer(hwnd_, kStatusTimerId, 850, nullptr);
    InvalidateRect(hwnd_, nullptr, TRUE);
  }

 private:
  static LRESULT CALLBACK WindowProc(HWND hwnd, UINT message, WPARAM wparam, LPARAM lparam) {
    CandidateWindow *self = reinterpret_cast<CandidateWindow *>(GetWindowLongPtrW(hwnd, GWLP_USERDATA));
    if (message == WM_NCCREATE) {
      auto *create = reinterpret_cast<CREATESTRUCTW *>(lparam);
      self = reinterpret_cast<CandidateWindow *>(create->lpCreateParams);
      SetWindowLongPtrW(hwnd, GWLP_USERDATA, reinterpret_cast<LONG_PTR>(self));
    }
    if (self && message == WM_PAINT) {
      self->Paint(hwnd);
      return 0;
    }
    if (self && message == WM_TIMER && wparam == kStatusTimerId) {
      self->Hide();
      return 0;
    }
    if (message == WM_ERASEBKGND) {
      return 1;
    }
    return DefWindowProcW(hwnd, message, wparam, lparam);
  }

  void EnsureWindow() {
    if (hwnd_) {
      return;
    }
    static bool registered = false;
    if (!registered) {
      WNDCLASSW wc{};
      wc.style = CS_DROPSHADOW;
      wc.lpfnWndProc = CandidateWindow::WindowProc;
      wc.hInstance = g_instance;
      wc.hCursor = LoadCursorW(nullptr, IDC_ARROW);
      wc.lpszClassName = L"Shurufa233CandidateWindow";
      RegisterClassW(&wc);
      registered = true;
    }
    hwnd_ = CreateWindowExW(WS_EX_TOOLWINDOW | WS_EX_TOPMOST | WS_EX_NOACTIVATE,
                            L"Shurufa233CandidateWindow", L"", WS_POPUP,
                            CW_USEDEFAULT, CW_USEDEFAULT, 320, 42, nullptr, nullptr,
                            g_instance, this);
    if (hwnd_) {
      DWM_WINDOW_CORNER_PREFERENCE corner = DWMWCP_ROUND;
      DwmSetWindowAttribute(hwnd_, DWMWA_WINDOW_CORNER_PREFERENCE, &corner, sizeof(corner));
    }
  }

  POINT CaretAnchor() const {
    GUITHREADINFO info{};
    info.cbSize = sizeof(info);
    if (GetGUIThreadInfo(0, &info) && !IsRectEmpty(&info.rcCaret)) {
      POINT anchor{info.rcCaret.left, info.rcCaret.bottom};
      if (info.hwndCaret && ClientToScreen(info.hwndCaret, &anchor)) {
        return anchor;
      }
      return anchor;
    }
    POINT cursor{};
    GetCursorPos(&cursor);
    return cursor;
  }

  void Paint(HWND hwnd) {
    PAINTSTRUCT ps{};
    HDC paintDc = BeginPaint(hwnd, &ps);
    RECT rect{};
    GetClientRect(hwnd, &rect);
    const int width = max(1, rect.right - rect.left);
    const int height = max(1, rect.bottom - rect.top);

    HDC dc = CreateCompatibleDC(paintDc);
    HBITMAP bitmap = dc ? CreateCompatibleBitmap(paintDc, width, height) : nullptr;
    HGDIOBJ oldBitmap = nullptr;
    if (dc && bitmap) {
      oldBitmap = SelectObject(dc, bitmap);
    } else {
      if (dc) {
        DeleteDC(dc);
      }
      dc = paintDc;
    }

    HBRUSH bg = CreateSolidBrush(surface_);
    FillRect(dc, &rect, bg);
    DeleteObject(bg);

    HPEN border = CreatePen(PS_SOLID, 1, border_);
    HGDIOBJ oldPen = SelectObject(dc, border);
    HGDIOBJ oldBrush = SelectObject(dc, GetStockObject(HOLLOW_BRUSH));
    RoundRect(dc, rect.left, rect.top, rect.right, rect.bottom, 12, 12);
    SelectObject(dc, oldBrush);
    SelectObject(dc, oldPen);
    DeleteObject(border);

    HFONT font = EnsureFont();
    HGDIOBJ oldFont = SelectObject(dc, font);
    SetBkMode(dc, TRANSPARENT);
    if (!statusText_.empty()) {
      DrawStatus(dc, rect);
    } else {
      DrawCandidates(dc, rect);
    }
    SelectObject(dc, oldFont);
    if (dc != paintDc) {
      BitBlt(paintDc, 0, 0, width, height, dc, 0, 0, SRCCOPY);
      SelectObject(dc, oldBitmap);
      DeleteObject(bitmap);
      DeleteDC(dc);
    }
    EndPaint(hwnd, &ps);
  }

  HWND hwnd_ = nullptr;
  static constexpr UINT_PTR kStatusTimerId = 1;
  std::vector<CandidateView> candidates_;
  std::wstring statusText_;
  int selectedIndex_ = 0;
  std::wstring fontFamily_ = L"Microsoft YaHei UI";
  int fontSize_ = 18;
  COLORREF accent_ = RGB(37, 99, 235);
  COLORREF surface_ = RGB(255, 255, 255);
  COLORREF text_ = RGB(17, 24, 39);
  COLORREF mutedText_ = RGB(100, 116, 139);
  COLORREF border_ = RGB(209, 213, 219);
  COLORREF highlightText_ = RGB(255, 255, 255);
  std::string theme_ = "system";
  DWORD lastSkinRefreshTick_ = 0;
  HFONT font_ = nullptr;
  std::wstring fontFamilyKey_;
  int fontSizeKey_ = 0;

  int CandidateWindowHeight() const {
    return max(48, fontSize_ + 34);
  }

  HFONT EnsureFont() {
    if (font_ && fontFamilyKey_ == fontFamily_ && fontSizeKey_ == fontSize_) {
      return font_;
    }
    ResetFont();
    fontFamilyKey_ = fontFamily_;
    fontSizeKey_ = fontSize_;
    font_ = CreateFontW(-fontSize_, 0, 0, 0, FW_NORMAL, FALSE, FALSE, FALSE,
                        DEFAULT_CHARSET, OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS,
                        CLEARTYPE_QUALITY, DEFAULT_PITCH | FF_DONTCARE,
                        fontFamily_.c_str());
    return font_;
  }

  void ResetFont() {
    if (font_) {
      DeleteObject(font_);
      font_ = nullptr;
    }
    fontFamilyKey_.clear();
    fontSizeKey_ = 0;
  }

  int TextWidth(HDC dc, const std::wstring &value) const {
    if (value.empty()) {
      return 0;
    }
    SIZE size{};
    if (!GetTextExtentPoint32W(dc, value.c_str(), static_cast<int>(value.size()), &size)) {
      return static_cast<int>(value.size()) * fontSize_;
    }
    return size.cx;
  }

  int CandidateItemWidth(HDC dc, const CandidateView &candidate, bool selected) const {
    const int textWidth = TextWidth(dc, candidate.text);
    const int readingWidth = selected ? TextWidth(dc, candidate.reading) : 0;
    const int padding = selected && readingWidth > 0 ? 58 : 44;
    return max(62, min(260, padding + textWidth + readingWidth));
  }

  int MeasureWindowWidth() {
    HDC dc = GetDC(hwnd_);
    HGDIOBJ oldFont = SelectObject(dc, EnsureFont());
    int width = 20;
    for (size_t i = 0; i < candidates_.size() && i < 7; ++i) {
      width += CandidateItemWidth(dc, candidates_[i], static_cast<int>(i) == selectedIndex_) + 6;
    }
    SelectObject(dc, oldFont);
    ReleaseDC(hwnd_, dc);
    return max(180, min(780, width));
  }

  int MeasureStatusWidth() {
    HDC dc = GetDC(hwnd_);
    HGDIOBJ oldFont = SelectObject(dc, EnsureFont());
    const int width = max(92, min(180, TextWidth(dc, statusText_) + 40));
    SelectObject(dc, oldFont);
    ReleaseDC(hwnd_, dc);
    return width;
  }

  POINT FitToWorkArea(POINT anchor, int width, int height) const {
    RECT work{};
    HMONITOR monitor = MonitorFromPoint(anchor, MONITOR_DEFAULTTONEAREST);
    MONITORINFO info{};
    info.cbSize = sizeof(info);
    if (monitor && GetMonitorInfoW(monitor, &info)) {
      work = info.rcWork;
    } else {
      SystemParametersInfoW(SPI_GETWORKAREA, 0, &work, 0);
    }
    POINT origin{anchor.x, anchor.y + 8};
    origin.x = max(work.left + 8, min(origin.x, work.right - width - 8));
    if (origin.y + height > work.bottom - 8) {
      origin.y = anchor.y - height - 8;
    }
    origin.y = max(work.top + 8, min(origin.y, work.bottom - height - 8));
    return origin;
  }

  static COLORREF MixColor(COLORREF left, COLORREF right, int rightPercent) {
    rightPercent = max(0, min(100, rightPercent));
    const int leftPercent = 100 - rightPercent;
    const int red = (GetRValue(left) * leftPercent + GetRValue(right) * rightPercent) / 100;
    const int green = (GetGValue(left) * leftPercent + GetGValue(right) * rightPercent) / 100;
    const int blue = (GetBValue(left) * leftPercent + GetBValue(right) * rightPercent) / 100;
    return RGB(red, green, blue);
  }

  static COLORREF ParseColor(const std::string &value) {
    if (value.size() != 7 || value[0] != '#') {
      return RGB(37, 99, 235);
    }
    unsigned int rgb = 0;
    if (sscanf_s(value.c_str() + 1, "%x", &rgb) != 1) {
      return RGB(37, 99, 235);
    }
    return RGB((rgb >> 16) & 0xff, (rgb >> 8) & 0xff, rgb & 0xff);
  }

  std::vector<CandidateView> ParseCandidates(const std::string &payload) const {
    std::vector<CandidateView> parsed;
    size_t lineStart = 0;
    while (lineStart < payload.size()) {
      size_t lineEnd = payload.find('\n', lineStart);
      if (lineEnd == std::string::npos) {
        lineEnd = payload.size();
      }
      std::string line = payload.substr(lineStart, lineEnd - lineStart);
      if (!line.empty()) {
        size_t first = line.find('\t');
        size_t second = first == std::string::npos ? std::string::npos : line.find('\t', first + 1);
        size_t third = second == std::string::npos ? std::string::npos : line.find('\t', second + 1);
        if (first != std::string::npos && second != std::string::npos && third != std::string::npos) {
          CandidateView item;
          item.index = atoi(line.substr(0, first).c_str());
          item.text = Utf8ToWide(line.substr(first + 1, second - first - 1).c_str());
          item.reading = Utf8ToWide(line.substr(second + 1, third - second - 1).c_str());
          item.score = atoi(line.substr(third + 1).c_str());
          parsed.push_back(item);
        }
      }
      lineStart = lineEnd + 1;
    }
    return parsed;
  }

  void DrawCandidates(HDC dc, const RECT &rect) {
    statusText_.clear();
    int x = rect.left + 14;
    const int y = rect.top + 8;
    const int itemHeight = max(32, fontSize_ + 18);
    for (size_t i = 0; i < candidates_.size() && i < 7; ++i) {
      const CandidateView &candidate = candidates_[i];
      const bool selected = static_cast<int>(i) == selectedIndex_;
      const int readingWidth = selected ? TextWidth(dc, candidate.reading) : 0;
      const int itemWidth = CandidateItemWidth(dc, candidate, selected);
      RECT itemRect{x, y, x + itemWidth, y + itemHeight};

      if (selected) {
        HBRUSH selected = CreateSolidBrush(accent_);
        HPEN selectedPen = CreatePen(PS_SOLID, 1, accent_);
        HGDIOBJ oldBrush = SelectObject(dc, selected);
        HGDIOBJ oldPen = SelectObject(dc, selectedPen);
        RoundRect(dc, itemRect.left, itemRect.top, itemRect.right, itemRect.bottom, 10, 10);
        SelectObject(dc, oldPen);
        SelectObject(dc, oldBrush);
        DeleteObject(selectedPen);
        DeleteObject(selected);
      }

      wchar_t number[8]{};
      StringCchPrintfW(number, ARRAYSIZE(number), L"%d", candidate.index);
      SetTextColor(dc, selected ? highlightText_ : mutedText_);
      RECT numberRect{itemRect.left + 10, itemRect.top, itemRect.left + 28, itemRect.bottom};
      DrawTextW(dc, number, -1, &numberRect, DT_SINGLELINE | DT_VCENTER | DT_LEFT);

      SetTextColor(dc, selected ? highlightText_ : text_);
      RECT textRect{itemRect.left + 28, itemRect.top, itemRect.right - 8, itemRect.bottom};
      if (selected && readingWidth > 0) {
        textRect.right = max(textRect.left + 24, itemRect.right - readingWidth - 14);
      }
      DrawTextW(dc, candidate.text.c_str(), static_cast<int>(candidate.text.size()), &textRect,
                DT_SINGLELINE | DT_VCENTER | DT_LEFT | DT_END_ELLIPSIS);

      if (selected && !candidate.reading.empty()) {
        SetTextColor(dc, MixColor(highlightText_, accent_, 22));
        RECT readingRect{max(itemRect.left + 72, itemRect.right - readingWidth - 8), itemRect.top,
                         itemRect.right - 8, itemRect.bottom};
        DrawTextW(dc, candidate.reading.c_str(), static_cast<int>(candidate.reading.size()),
                  &readingRect, DT_SINGLELINE | DT_VCENTER | DT_RIGHT | DT_END_ELLIPSIS);
      }
      x += itemWidth + 6;
      if (x > rect.right - 40) {
        break;
      }
    }
  }

  void DrawStatus(HDC dc, const RECT &rect) {
    RECT badge{rect.left + 10, rect.top + 7, rect.right - 10, rect.bottom - 7};
    HBRUSH selected = CreateSolidBrush(accent_);
    HPEN selectedPen = CreatePen(PS_SOLID, 1, accent_);
    HGDIOBJ oldBrush = SelectObject(dc, selected);
    HGDIOBJ oldPen = SelectObject(dc, selectedPen);
    RoundRect(dc, badge.left, badge.top, badge.right, badge.bottom, 10, 10);
    SelectObject(dc, oldPen);
    SelectObject(dc, oldBrush);
    DeleteObject(selectedPen);
    DeleteObject(selected);

    SetTextColor(dc, highlightText_);
    DrawTextW(dc, statusText_.c_str(), static_cast<int>(statusText_.size()), &badge,
              DT_SINGLELINE | DT_VCENTER | DT_CENTER);
  }

  void RefreshSkin() {
    const DWORD now = GetTickCount();
    if (lastSkinRefreshTick_ != 0 && now - lastSkinRefreshTick_ < 2000) {
      return;
    }
    lastSkinRefreshTick_ = now;
    const std::string skin = HttpRequest(L"GET", L"/ime/skin");
    if (skin.empty()) {
      return;
    }
    size_t first = skin.find('|');
    size_t second = first == std::string::npos ? std::string::npos : skin.find('|', first + 1);
    size_t third = second == std::string::npos ? std::string::npos : skin.find('|', second + 1);
    size_t fourth = third == std::string::npos ? std::string::npos : skin.find('|', third + 1);
    size_t fifth = fourth == std::string::npos ? std::string::npos : skin.find('|', fourth + 1);
    size_t sixth = fifth == std::string::npos ? std::string::npos : skin.find('|', fifth + 1);
    size_t seventh = sixth == std::string::npos ? std::string::npos : skin.find('|', sixth + 1);
    size_t eighth = seventh == std::string::npos ? std::string::npos : skin.find('|', seventh + 1);
    if (first == std::string::npos || second == std::string::npos || third == std::string::npos ||
        fourth == std::string::npos || fifth == std::string::npos || sixth == std::string::npos ||
        seventh == std::string::npos || eighth == std::string::npos) {
      return;
    }
    std::wstring nextFontFamily = Utf8ToWide(skin.substr(0, first).c_str());
    if (nextFontFamily.empty()) {
      nextFontFamily = L"Microsoft YaHei UI";
    }
    const int nextFontSize = max(13, min(28, atoi(skin.substr(first + 1, second - first - 1).c_str()) + 3));
    if (nextFontFamily != fontFamily_ || nextFontSize != fontSize_) {
      fontFamily_ = nextFontFamily;
      fontSize_ = nextFontSize;
      ResetFont();
    }
    accent_ = ParseColor(skin.substr(second + 1, third - second - 1));
    surface_ = ParseColor(skin.substr(third + 1, fourth - third - 1));
    text_ = ParseColor(skin.substr(fourth + 1, fifth - fourth - 1));
    mutedText_ = ParseColor(skin.substr(fifth + 1, sixth - fifth - 1));
    border_ = ParseColor(skin.substr(sixth + 1, seventh - sixth - 1));
    highlightText_ = ParseColor(skin.substr(seventh + 1, eighth - seventh - 1));
    theme_ = skin.substr(eighth + 1);
  }
};

bool IsAsciiLetter(WPARAM key) {
  return (key >= L'A' && key <= L'Z') || (key >= L'a' && key <= L'z');
}

bool IsShiftKey(WPARAM key) {
  return key == VK_SHIFT || key == VK_LSHIFT || key == VK_RSHIFT;
}

int CandidateIndexFromKey(WPARAM key) {
  if (key >= L'1' && key <= L'9') {
    return static_cast<int>(key - L'1');
  }
  return 0;
}

class EditSession final : public ITfEditSession {
 public:
  EditSession(ITfContext *context, std::wstring text) : context_(context), text_(std::move(text)) {
    AddDllRef();
    if (context_) {
      context_->AddRef();
    }
  }

  ~EditSession() {
    if (context_) {
      context_->Release();
    }
    ReleaseDllRef();
  }

  STDMETHODIMP QueryInterface(REFIID riid, void **out) override {
    if (!out) {
      return E_INVALIDARG;
    }
    *out = nullptr;
    if (riid == IID_IUnknown || riid == IID_ITfEditSession) {
      *out = static_cast<ITfEditSession *>(this);
      AddRef();
      return S_OK;
    }
    return E_NOINTERFACE;
  }

  STDMETHODIMP_(ULONG) AddRef() override {
    return InterlockedIncrement(&refCount_);
  }

  STDMETHODIMP_(ULONG) Release() override {
    const ULONG count = InterlockedDecrement(&refCount_);
    if (count == 0) {
      delete this;
    }
    return count;
  }

  STDMETHODIMP DoEditSession(TfEditCookie ec) override {
    if (!context_ || text_.empty()) {
      return S_OK;
    }

    TF_SELECTION selection{};
    ULONG fetched = 0;
    HRESULT hr = context_->GetSelection(ec, TF_DEFAULT_SELECTION, 1, &selection, &fetched);
    if (FAILED(hr) || fetched == 0 || !selection.range) {
      return hr;
    }

    hr = selection.range->SetText(ec, 0, text_.c_str(), static_cast<LONG>(text_.size()));
    selection.range->Release();
    return hr;
  }

 private:
  long refCount_ = 1;
  ITfContext *context_ = nullptr;
  std::wstring text_;
};

class TextService final : public ITfTextInputProcessorEx, public ITfKeyEventSink {
 public:
  TextService() {
    AddDllRef();
  }

  ~TextService() {
    Deactivate();
    if (session_ && g_core.destroySession) {
      g_core.destroySession(session_);
      session_ = 0;
    }
    ReleaseDllRef();
  }

  STDMETHODIMP QueryInterface(REFIID riid, void **out) override {
    if (!out) {
      return E_INVALIDARG;
    }
    *out = nullptr;
    if (riid == IID_IUnknown || riid == IID_ITfTextInputProcessor) {
      *out = static_cast<ITfTextInputProcessor *>(this);
    } else if (riid == IID_ITfTextInputProcessorEx) {
      *out = static_cast<ITfTextInputProcessorEx *>(this);
    } else if (riid == IID_ITfKeyEventSink) {
      *out = static_cast<ITfKeyEventSink *>(this);
    } else {
      return E_NOINTERFACE;
    }
    AddRef();
    return S_OK;
  }

  STDMETHODIMP_(ULONG) AddRef() override {
    return InterlockedIncrement(&refCount_);
  }

  STDMETHODIMP_(ULONG) Release() override {
    const ULONG count = InterlockedDecrement(&refCount_);
    if (count == 0) {
      delete this;
    }
    return count;
  }

  STDMETHODIMP Activate(ITfThreadMgr *threadMgr, TfClientId clientId) override {
    return ActivateEx(threadMgr, clientId, 0);
  }

  STDMETHODIMP ActivateEx(ITfThreadMgr *threadMgr, TfClientId clientId, DWORD) override {
    LogLine(L"ActivateEx called");
    if (!threadMgr || !EnsureCoreLoaded()) {
      LogLine(L"ActivateEx failed: threadMgr/core unavailable");
      return E_FAIL;
    }
    threadMgr_ = threadMgr;
    threadMgr_->AddRef();
    clientId_ = clientId;
    session_ = g_core.createSession();

    HRESULT hr = threadMgr_->QueryInterface(IID_ITfKeystrokeMgr, reinterpret_cast<void **>(&keyMgr_));
    if (SUCCEEDED(hr) && keyMgr_) {
      hr = keyMgr_->AdviseKeyEventSink(clientId_, static_cast<ITfKeyEventSink *>(this), TRUE);
    }
    wchar_t message[160]{};
    StringCchPrintfW(message, ARRAYSIZE(message), L"ActivateEx session=%llu advise_hr=0x%08X",
                     session_, static_cast<unsigned int>(hr));
    LogLine(message);
    return hr;
  }

  STDMETHODIMP Deactivate() override {
    LogLine(L"Deactivate called");
    candidateWindow_.Hide();
    cachedCandidateCount_ = 0;
    if (session_ && g_core.Ready()) {
      g_core.destroySession(session_);
      session_ = 0;
    }
    if (keyMgr_) {
      keyMgr_->UnadviseKeyEventSink(clientId_);
      keyMgr_->Release();
      keyMgr_ = nullptr;
    }
    if (threadMgr_) {
      threadMgr_->Release();
      threadMgr_ = nullptr;
    }
    clientId_ = TF_CLIENTID_NULL;
    return S_OK;
  }

  STDMETHODIMP OnSetFocus(BOOL) override {
    return S_OK;
  }

  STDMETHODIMP OnTestKeyDown(ITfContext *, WPARAM key, LPARAM, BOOL *eaten) override {
    if (!eaten) {
      return E_INVALIDARG;
    }
    *eaten = ShouldEatKey(key);
    return S_OK;
  }

  STDMETHODIMP OnKeyDown(ITfContext *context, WPARAM key, LPARAM, BOOL *eaten) override {
    if (!eaten) {
      return E_INVALIDARG;
    }
    *eaten = FALSE;
    if (!session_ || !ShouldEatKey(key)) {
      return S_OK;
    }

    if (IsShiftKey(key)) {
      if (!shiftDown_) {
        shiftDown_ = true;
        ToggleAsciiMode();
      }
      *eaten = TRUE;
      return S_OK;
    }

    if (IsAsciiLetter(key)) {
      char ch = static_cast<char>(key);
      if (ch >= 'A' && ch <= 'Z') {
        ch = static_cast<char>(ch - 'A' + 'a');
      }
      selectedIndex_ = 0;
      const int count = g_core.inputKeyFast(session_, ch);
      UpdateCandidateWindow(count);
      *eaten = TRUE;
      return S_OK;
    }

    if (key == VK_BACK) {
      selectedIndex_ = 0;
      const int count = g_core.backspaceFast(session_);
      UpdateCandidateWindow(count);
      *eaten = TRUE;
      return S_OK;
    }

    if (key == VK_ESCAPE) {
      selectedIndex_ = 0;
      ClearSession();
      candidateWindow_.Hide();
      cachedCandidateCount_ = 0;
      *eaten = TRUE;
      return S_OK;
    }

    if (key == VK_RIGHT || key == VK_DOWN || key == VK_TAB) {
      MoveSelection(1);
      *eaten = TRUE;
      return S_OK;
    }

    if (key == VK_LEFT || key == VK_UP) {
      MoveSelection(-1);
      *eaten = TRUE;
      return S_OK;
    }

    if (key == VK_SPACE || key == VK_RETURN || (key >= L'1' && key <= L'9')) {
      const int index = (key >= L'1' && key <= L'9') ? CandidateIndexFromKey(key) : selectedIndex_;
      CommitCandidate(context, index);
      selectedIndex_ = 0;
      candidateWindow_.Hide();
      cachedCandidateCount_ = 0;
      *eaten = TRUE;
      return S_OK;
    }

    return S_OK;
  }

  STDMETHODIMP OnTestKeyUp(ITfContext *, WPARAM key, LPARAM, BOOL *eaten) override {
    if (!eaten) {
      return E_INVALIDARG;
    }
    *eaten = IsShiftKey(key);
    return S_OK;
  }

  STDMETHODIMP OnKeyUp(ITfContext *, WPARAM key, LPARAM, BOOL *eaten) override {
    if (!eaten) {
      return E_INVALIDARG;
    }
    if (IsShiftKey(key)) {
      shiftDown_ = false;
      *eaten = TRUE;
      return S_OK;
    }
    *eaten = FALSE;
    return S_OK;
  }

  STDMETHODIMP OnPreservedKey(ITfContext *, REFGUID, BOOL *eaten) override {
    if (!eaten) {
      return E_INVALIDARG;
    }
    *eaten = FALSE;
    return S_OK;
  }

 private:
  bool ShouldEatKey(WPARAM key) const {
    if (!session_) {
      return false;
    }
    if (IsShiftKey(key)) {
      return true;
    }
    if (asciiMode_) {
      return cachedCandidateCount_ > 0 && (key == VK_ESCAPE || key == VK_SPACE || key == VK_RETURN);
    }
    if (IsAsciiLetter(key) || key == VK_BACK) {
      return true;
    }
    if (key == VK_ESCAPE) {
      return cachedCandidateCount_ > 0;
    }
    if (key == VK_RIGHT || key == VK_DOWN || key == VK_TAB || key == VK_LEFT || key == VK_UP) {
      return cachedCandidateCount_ > 0;
    }
    if (key == VK_SPACE || key == VK_RETURN) {
      return cachedCandidateCount_ > 0;
    }
    if (key >= L'1' && key <= L'9') {
      return cachedCandidateCount_ > CandidateIndexFromKey(key);
    }
    return false;
  }

  void CommitCandidate(ITfContext *context, int index) {
    if (!context || !session_) {
      LogLine(L"CommitCandidate skipped: no context/session");
      return;
    }
    char *committed = g_core.commitCandidate(session_, index);
    std::wstring text = Utf8ToWide(committed);
    if (committed) {
      g_core.freeValue(committed);
    }
    if (text.empty()) {
      LogLine(L"CommitCandidate skipped: empty committed text");
      return;
    }

    EditSession *session = new EditSession(context, text);
    HRESULT editResult = E_FAIL;
    HRESULT hr = context->RequestEditSession(
        clientId_, session, TF_ES_READWRITE | TF_ES_SYNC, &editResult);
    session->Release();
    wchar_t message[192]{};
    StringCchPrintfW(message, ARRAYSIZE(message),
                     L"CommitCandidate text_len=%zu request_hr=0x%08X edit_hr=0x%08X",
                     text.size(), static_cast<unsigned int>(hr), static_cast<unsigned int>(editResult));
    LogLine(message);
    if (hr == TF_E_SYNCHRONOUS || FAILED(hr)) {
      EditSession *asyncSession = new EditSession(context, text);
      HRESULT ignored = E_FAIL;
      context->RequestEditSession(clientId_, asyncSession, TF_ES_READWRITE | TF_ES_ASYNC, &ignored);
      asyncSession->Release();
    }
  }

  std::string BuildCandidatePayloadFromCore(int count) {
    if (g_core.candidatePayload) {
      char *payload = g_core.candidatePayload(session_, min(count, 9));
      std::string result = payload ? payload : "";
      if (payload) {
        g_core.freeValue(payload);
      }
      if (!result.empty()) {
        return result;
      }
    }

    std::string payload;
    for (int i = 0; i < count && i < 9; ++i) {
      char *text = g_core.candidateText(session_, i);
      char *reading = g_core.candidateReading(session_, i);
      const int score = g_core.candidateScore(session_, i);

      char prefix[32]{};
      sprintf_s(prefix, "%d\t", i + 1);
      if (!payload.empty()) {
        payload += "\n";
      }
      payload += prefix;
      payload += text ? text : "";
      payload += "\t";
      payload += reading ? reading : "";
      payload += "\t";
      char scoreText[32]{};
      sprintf_s(scoreText, "%d", score);
      payload += scoreText;

      if (text) {
        g_core.freeValue(text);
      }
      if (reading) {
        g_core.freeValue(reading);
      }
    }
    return payload;
  }

  void UpdateCandidateWindow(int knownCount = -1) {
    const int count = knownCount >= 0 ? knownCount : g_core.candidateCount(session_);
    cachedCandidateCount_ = count;
    if (count <= 0) {
      candidateWindow_.Hide();
      return;
    }
    if (count > 0 && selectedIndex_ >= count) {
      selectedIndex_ = count - 1;
    }
    std::string candidates;
    if (g_core.inProcess) {
      candidates = BuildCandidatePayloadFromCore(count);
    } else {
      wchar_t path[80]{};
      StringCchPrintfW(path, ARRAYSIZE(path), L"/ime/candidates?session=%llu", session_);
      candidates = HttpRequest(L"GET", path);
    }
    if (candidates.empty()) {
      wchar_t message[128]{};
      StringCchPrintfW(message, ARRAYSIZE(message), L"UpdateCandidateWindow empty session=%llu", session_);
      LogLine(message);
      candidateWindow_.Hide();
      cachedCandidateCount_ = 0;
      return;
    }
    candidateWindow_.Show(candidates, selectedIndex_);
  }

  void MoveSelection(int delta) {
    const int count = cachedCandidateCount_;
    if (count <= 0) {
      return;
    }
    selectedIndex_ = (selectedIndex_ + delta + count) % count;
    UpdateCandidateWindow();
  }

  void ClearSession() {
    if (!session_ || !g_core.clearSession) {
      return;
    }
    char *cleared = g_core.clearSession(session_);
    if (cleared) {
      g_core.freeValue(cleared);
    }
    cachedCandidateCount_ = 0;
  }

  void ToggleAsciiMode() {
    asciiMode_ = !asciiMode_;
    selectedIndex_ = 0;
    ClearSession();
    wchar_t message[96]{};
    StringCchPrintfW(message, ARRAYSIZE(message), L"ToggleAsciiMode ascii=%d", asciiMode_ ? 1 : 0);
    LogLine(message);
    candidateWindow_.ShowStatus(asciiMode_ ? L"EN" : L"中");
  }

  long refCount_ = 1;
  ITfThreadMgr *threadMgr_ = nullptr;
  ITfKeystrokeMgr *keyMgr_ = nullptr;
  TfClientId clientId_ = TF_CLIENTID_NULL;
  uint64_t session_ = 0;
  int selectedIndex_ = 0;
  int cachedCandidateCount_ = 0;
  bool asciiMode_ = false;
  bool shiftDown_ = false;
  CandidateWindow candidateWindow_;
};

class ClassFactory final : public IClassFactory {
 public:
  ClassFactory() {
    AddDllRef();
  }

  ~ClassFactory() {
    ReleaseDllRef();
  }

  STDMETHODIMP QueryInterface(REFIID riid, void **out) override {
    if (!out) {
      return E_INVALIDARG;
    }
    *out = nullptr;
    if (riid == IID_IUnknown || riid == IID_IClassFactory) {
      *out = static_cast<IClassFactory *>(this);
      AddRef();
      return S_OK;
    }
    return E_NOINTERFACE;
  }

  STDMETHODIMP_(ULONG) AddRef() override {
    return InterlockedIncrement(&refCount_);
  }

  STDMETHODIMP_(ULONG) Release() override {
    const ULONG count = InterlockedDecrement(&refCount_);
    if (count == 0) {
      delete this;
    }
    return count;
  }

  STDMETHODIMP CreateInstance(IUnknown *outer, REFIID riid, void **out) override {
    LogLine(L"ClassFactory CreateInstance called");
    if (outer) {
      return CLASS_E_NOAGGREGATION;
    }
    TextService *service = new TextService();
    HRESULT hr = service->QueryInterface(riid, out);
    service->Release();
    return hr;
  }

  STDMETHODIMP LockServer(BOOL lock) override {
    if (lock) {
      AddDllRef();
    } else {
      ReleaseDllRef();
    }
    return S_OK;
  }

 private:
  long refCount_ = 1;
};

std::wstring GuidToString(REFGUID guid) {
  wchar_t text[64]{};
  StringFromGUID2(guid, text, ARRAYSIZE(text));
  return text;
}

HRESULT WriteComRegistration(HKEY root) {
  wchar_t modulePath[MAX_PATH]{};
  if (!GetModuleFileNameW(g_instance, modulePath, ARRAYSIZE(modulePath))) {
    return HRESULT_FROM_WIN32(GetLastError());
  }

  const std::wstring clsidKey =
      L"Software\\Classes\\CLSID\\" + GuidToString(kClsidTextService);
  HKEY clsid = nullptr;
  LSTATUS status = RegCreateKeyExW(root, clsidKey.c_str(), 0, nullptr, 0,
                                   KEY_WRITE, nullptr, &clsid, nullptr);
  if (status != ERROR_SUCCESS) {
    return HRESULT_FROM_WIN32(status);
  }

  RegSetValueExW(clsid, nullptr, 0, REG_SZ,
                 reinterpret_cast<const BYTE *>(kDescription),
                 sizeof(kDescription));

  HKEY inproc = nullptr;
  status = RegCreateKeyExW(clsid, L"InprocServer32", 0, nullptr, 0, KEY_WRITE,
                           nullptr, &inproc, nullptr);
  if (status == ERROR_SUCCESS) {
    RegSetValueExW(inproc, nullptr, 0, REG_SZ,
                   reinterpret_cast<const BYTE *>(modulePath),
                   static_cast<DWORD>((wcslen(modulePath) + 1) * sizeof(wchar_t)));
    RegSetValueExW(inproc, L"ThreadingModel", 0, REG_SZ,
                   reinterpret_cast<const BYTE *>(kModel), sizeof(kModel));
    RegCloseKey(inproc);
  }
  RegCloseKey(clsid);
  return HRESULT_FROM_WIN32(status);
}

HRESULT RegisterComServer() {
  HRESULT userHr = WriteComRegistration(HKEY_CURRENT_USER);
  HRESULT machineHr = WriteComRegistration(HKEY_LOCAL_MACHINE);
  if (SUCCEEDED(machineHr)) {
    return SUCCEEDED(userHr) ? S_OK : userHr;
  }
  if (machineHr == HRESULT_FROM_WIN32(ERROR_ACCESS_DENIED)) {
    return userHr;
  }
  return machineHr;
}

HRESULT UnregisterComServer() {
  const std::wstring clsidKey =
      L"Software\\Classes\\CLSID\\" + GuidToString(kClsidTextService);
  LSTATUS userStatus = RegDeleteTreeW(HKEY_CURRENT_USER, clsidKey.c_str());
  LSTATUS machineStatus = RegDeleteTreeW(HKEY_LOCAL_MACHINE, clsidKey.c_str());
  LSTATUS status = machineStatus == ERROR_SUCCESS ? machineStatus : userStatus;
  if (status == ERROR_FILE_NOT_FOUND || status == ERROR_ACCESS_DENIED) {
    return S_OK;
  }
  return HRESULT_FROM_WIN32(status);
}

HRESULT RegisterTextServiceProfile() {
  HRESULT hr = CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED);
  bool didCoInit = SUCCEEDED(hr);
  if (hr == RPC_E_CHANGED_MODE) {
    didCoInit = false;
  } else if (FAILED(hr)) {
    return hr;
  }

  ITfInputProcessorProfiles *profiles = nullptr;
  hr = CoCreateInstance(CLSID_TF_InputProcessorProfiles, nullptr, CLSCTX_INPROC_SERVER,
                        IID_ITfInputProcessorProfiles,
                        reinterpret_cast<void **>(&profiles));
  if (SUCCEEDED(hr) && profiles) {
    hr = profiles->Register(kClsidTextService);
    if (SUCCEEDED(hr)) {
      hr = profiles->AddLanguageProfile(
          kClsidTextService, kLanguage, kProfileGuid,
          const_cast<WCHAR *>(kDescription), static_cast<ULONG>(wcslen(kDescription)),
          nullptr, 0, 0);
    }
    if (SUCCEEDED(hr)) {
      hr = profiles->EnableLanguageProfile(kClsidTextService, kLanguage, kProfileGuid, TRUE);
    }
    profiles->Release();
  }

  ITfCategoryMgr *categoryMgr = nullptr;
  if (SUCCEEDED(CoCreateInstance(CLSID_TF_CategoryMgr, nullptr, CLSCTX_INPROC_SERVER,
                                 IID_ITfCategoryMgr,
                                 reinterpret_cast<void **>(&categoryMgr))) &&
      categoryMgr) {
    categoryMgr->RegisterCategory(kClsidTextService, GUID_TFCAT_TIP_KEYBOARD,
                                  kClsidTextService);
    categoryMgr->Release();
  }

  if (didCoInit) {
    CoUninitialize();
  }
  return hr;
}

HRESULT UnregisterTextServiceProfile() {
  HRESULT hr = CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED);
  bool didCoInit = SUCCEEDED(hr);
  if (hr == RPC_E_CHANGED_MODE) {
    didCoInit = false;
  } else if (FAILED(hr)) {
    return hr;
  }

  ITfInputProcessorProfiles *profiles = nullptr;
  hr = CoCreateInstance(CLSID_TF_InputProcessorProfiles, nullptr, CLSCTX_INPROC_SERVER,
                        IID_ITfInputProcessorProfiles,
                        reinterpret_cast<void **>(&profiles));
  if (SUCCEEDED(hr) && profiles) {
    profiles->RemoveLanguageProfile(kClsidTextService, kLanguage, kProfileGuid);
    hr = profiles->Unregister(kClsidTextService);
    profiles->Release();
  }

  if (didCoInit) {
    CoUninitialize();
  }
  return hr;
}

}  // namespace

BOOL APIENTRY DllMain(HINSTANCE instance, DWORD reason, LPVOID) {
  if (reason == DLL_PROCESS_ATTACH) {
    g_instance = instance;
    DisableThreadLibraryCalls(instance);
    LogLine(L"DllMain PROCESS_ATTACH");
  } else if (reason == DLL_PROCESS_DETACH) {
    LogLine(L"DllMain PROCESS_DETACH");
    if (g_httpConnect) {
      WinHttpCloseHandle(g_httpConnect);
      g_httpConnect = nullptr;
    }
    if (g_httpSession) {
      WinHttpCloseHandle(g_httpSession);
      g_httpSession = nullptr;
    }
    if (g_httpLockReady) {
      DeleteCriticalSection(&g_httpLock);
      g_httpLockReady = false;
    }
    g_core = {};
  }
  return TRUE;
}

STDAPI DllCanUnloadNow() {
  return g_dllRefCount == 0 ? S_OK : S_FALSE;
}

STDAPI DllGetClassObject(REFCLSID clsid, REFIID riid, void **out) {
  LogLine(L"DllGetClassObject called");
  if (clsid != kClsidTextService) {
    LogLine(L"DllGetClassObject class not available");
    return CLASS_E_CLASSNOTAVAILABLE;
  }
  ClassFactory *factory = new ClassFactory();
  HRESULT hr = factory->QueryInterface(riid, out);
  factory->Release();
  return hr;
}

STDAPI DllRegisterServer() {
  HRESULT hr = RegisterComServer();
  if (FAILED(hr)) {
    return hr;
  }
  return RegisterTextServiceProfile();
}

STDAPI DllUnregisterServer() {
  UnregisterTextServiceProfile();
  return UnregisterComServer();
}
