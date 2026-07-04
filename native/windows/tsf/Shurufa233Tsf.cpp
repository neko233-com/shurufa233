#include <windows.h>
#include <msctf.h>
#include <strsafe.h>
#include <winhttp.h>

#include <cstdint>
#include <cstdio>
#include <string>

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

using CreateSessionFn = uint64_t (*)();
using DestroySessionFn = void (*)(uint64_t);
using InputKeyFastFn = int (*)(uint64_t, char);
using BackspaceFastFn = int (*)(uint64_t);
using CandidateCountFn = int (*)(uint64_t);
using CandidateTextFn = char *(*)(uint64_t, int);
using CommitCandidateFn = char *(*)(uint64_t, int);
using FreeFn = void (*)(char *);

struct CoreApi {
  bool initialized = false;
  CreateSessionFn createSession = nullptr;
  DestroySessionFn destroySession = nullptr;
  InputKeyFastFn inputKeyFast = nullptr;
  BackspaceFastFn backspaceFast = nullptr;
  CandidateCountFn candidateCount = nullptr;
  CandidateTextFn candidateText = nullptr;
  CommitCandidateFn commitCandidate = nullptr;
  FreeFn freeValue = nullptr;

  bool Ready() const {
    return initialized && createSession && destroySession && inputKeyFast &&
           backspaceFast && candidateCount && candidateText && commitCandidate &&
           freeValue;
  }
};

CoreApi g_core;
HINTERNET g_httpSession = nullptr;
HINTERNET g_httpConnect = nullptr;
CRITICAL_SECTION g_httpLock;
bool g_httpLockReady = false;

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
    return "";
  }
  EnterCriticalSection(&g_httpLock);
  HINTERNET request = WinHttpOpenRequest(g_httpConnect, verb, path.c_str(), nullptr,
                                         WINHTTP_NO_REFERER, WINHTTP_DEFAULT_ACCEPT_TYPES, 0);
  LeaveCriticalSection(&g_httpLock);
  if (!request) {
    return "";
  }

  std::string response;
  BOOL ok = WinHttpSendRequest(request, WINHTTP_NO_ADDITIONAL_HEADERS, 0,
                               WINHTTP_NO_REQUEST_DATA, 0, 0, 0);
  if (ok) {
    ok = WinHttpReceiveResponse(request, nullptr);
  }
  if (ok) {
    DWORD status = 0;
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
  return 1;
}

void HttpDestroySession(uint64_t) {}

int HttpInputKeyFast(uint64_t, char key) {
  wchar_t path[64]{};
  StringCchPrintfW(path, ARRAYSIZE(path), L"/ime/key?key=%c", static_cast<wchar_t>(key));
  HttpRequest(L"POST", path);
  return 0;
}

int HttpBackspaceFast(uint64_t) {
  HttpRequest(L"POST", L"/ime/backspace");
  return 0;
}

int HttpCandidateCount(uint64_t) {
  std::string value = HttpRequest(L"GET", L"/ime/count");
  return value.empty() ? 0 : atoi(value.c_str());
}

char *HttpCandidateText(uint64_t, int) {
  return AllocCString("");
}

char *HttpCommitCandidate(uint64_t, int index) {
  wchar_t path[64]{};
  StringCchPrintfW(path, ARRAYSIZE(path), L"/ime/select?index=%d", index);
  return AllocCString(HttpRequest(L"POST", path));
}

void HttpFree(char *value) {
  CoTaskMemFree(value);
}

bool EnsureCoreLoaded() {
  if (g_core.Ready()) {
    return true;
  }
  g_core.initialized = true;
  g_core.createSession = HttpCreateSession;
  g_core.destroySession = HttpDestroySession;
  g_core.inputKeyFast = HttpInputKeyFast;
  g_core.backspaceFast = HttpBackspaceFast;
  g_core.candidateCount = HttpCandidateCount;
  g_core.candidateText = HttpCandidateText;
  g_core.commitCandidate = HttpCommitCandidate;
  g_core.freeValue = HttpFree;
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
  void Show(const std::wstring &text) {
    if (text.empty()) {
      Hide();
      return;
    }
    EnsureWindow();
    RefreshSkin();
    text_ = text;

    POINT anchor = CaretAnchor();
    const int width = max(280, min(720, 44 + static_cast<int>(text_.size()) * (fontSize_ + 2)));
    const int height = max(42, fontSize_ + 28);
    SetWindowPos(hwnd_, HWND_TOPMOST, anchor.x, anchor.y + 8, width, height,
                 SWP_NOACTIVATE | SWP_SHOWWINDOW);
    InvalidateRect(hwnd_, nullptr, TRUE);
  }

  void Hide() {
    if (hwnd_) {
      ShowWindow(hwnd_, SW_HIDE);
    }
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
  }

  POINT CaretAnchor() const {
    GUITHREADINFO info{};
    info.cbSize = sizeof(info);
    if (GetGUIThreadInfo(0, &info) && !IsRectEmpty(&info.rcCaret)) {
      return POINT{info.rcCaret.left, info.rcCaret.bottom};
    }
    POINT cursor{};
    GetCursorPos(&cursor);
    return cursor;
  }

  void Paint(HWND hwnd) {
    PAINTSTRUCT ps{};
    HDC dc = BeginPaint(hwnd, &ps);
    RECT rect{};
    GetClientRect(hwnd, &rect);

    HBRUSH bg = CreateSolidBrush(RGB(255, 255, 255));
    FillRect(dc, &rect, bg);
    DeleteObject(bg);

    RECT accentRect = rect;
    accentRect.right = accentRect.left + 4;
    HBRUSH accent = CreateSolidBrush(accent_);
    FillRect(dc, &accentRect, accent);
    DeleteObject(accent);

    HPEN border = CreatePen(PS_SOLID, 1, RGB(210, 215, 224));
    HGDIOBJ oldPen = SelectObject(dc, border);
    HGDIOBJ oldBrush = SelectObject(dc, GetStockObject(HOLLOW_BRUSH));
    Rectangle(dc, rect.left, rect.top, rect.right, rect.bottom);
    SelectObject(dc, oldBrush);
    SelectObject(dc, oldPen);
    DeleteObject(border);

    HFONT font = CreateFontW(-fontSize_, 0, 0, 0, FW_NORMAL, FALSE, FALSE, FALSE,
                             DEFAULT_CHARSET, OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS,
                             CLEARTYPE_QUALITY, DEFAULT_PITCH | FF_DONTCARE,
                             fontFamily_.c_str());
    HGDIOBJ oldFont = SelectObject(dc, font);
    SetBkMode(dc, TRANSPARENT);
    SetTextColor(dc, RGB(17, 24, 39));
    RECT textRect = rect;
    textRect.left += 14;
    textRect.right -= 14;
    DrawTextW(dc, text_.c_str(), static_cast<int>(text_.size()), &textRect,
              DT_SINGLELINE | DT_VCENTER | DT_LEFT | DT_END_ELLIPSIS);
    SelectObject(dc, oldFont);
    DeleteObject(font);
    EndPaint(hwnd, &ps);
  }

  HWND hwnd_ = nullptr;
  std::wstring text_;
  std::wstring fontFamily_ = L"Microsoft YaHei UI";
  int fontSize_ = 18;
  COLORREF accent_ = RGB(37, 99, 235);

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

  void RefreshSkin() {
    const std::string skin = HttpRequest(L"GET", L"/ime/skin");
    if (skin.empty()) {
      return;
    }
    size_t first = skin.find('|');
    size_t second = first == std::string::npos ? std::string::npos : skin.find('|', first + 1);
    size_t third = second == std::string::npos ? std::string::npos : skin.find('|', second + 1);
    if (first == std::string::npos || second == std::string::npos || third == std::string::npos) {
      return;
    }
    fontFamily_ = Utf8ToWide(skin.substr(0, first).c_str());
    fontSize_ = max(13, min(28, atoi(skin.substr(first + 1, second - first - 1).c_str()) + 3));
    accent_ = ParseColor(skin.substr(second + 1, third - second - 1));
  }
};

bool IsAsciiLetter(WPARAM key) {
  return (key >= L'A' && key <= L'Z') || (key >= L'a' && key <= L'z');
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
    if (!threadMgr || !EnsureCoreLoaded()) {
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
    return hr;
  }

  STDMETHODIMP Deactivate() override {
    candidateWindow_.Hide();
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

    if (IsAsciiLetter(key)) {
      char ch = static_cast<char>(key);
      if (ch >= 'A' && ch <= 'Z') {
        ch = static_cast<char>(ch - 'A' + 'a');
      }
      g_core.inputKeyFast(session_, ch);
      UpdateCandidateWindow();
      *eaten = TRUE;
      return S_OK;
    }

    if (key == VK_BACK) {
      g_core.backspaceFast(session_);
      UpdateCandidateWindow();
      *eaten = TRUE;
      return S_OK;
    }

    if (key == VK_SPACE || key == VK_RETURN || (key >= L'1' && key <= L'9')) {
      CommitCandidate(context, CandidateIndexFromKey(key));
      candidateWindow_.Hide();
      *eaten = TRUE;
      return S_OK;
    }

    return S_OK;
  }

  STDMETHODIMP OnTestKeyUp(ITfContext *, WPARAM, LPARAM, BOOL *eaten) override {
    if (!eaten) {
      return E_INVALIDARG;
    }
    *eaten = FALSE;
    return S_OK;
  }

  STDMETHODIMP OnKeyUp(ITfContext *, WPARAM, LPARAM, BOOL *eaten) override {
    if (!eaten) {
      return E_INVALIDARG;
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
    if (IsAsciiLetter(key) || key == VK_BACK) {
      return true;
    }
    if (key == VK_SPACE || key == VK_RETURN) {
      return g_core.candidateCount(session_) > 0;
    }
    if (key >= L'1' && key <= L'9') {
      return g_core.candidateCount(session_) > CandidateIndexFromKey(key);
    }
    return false;
  }

  void CommitCandidate(ITfContext *context, int index) {
    if (!context || !session_) {
      return;
    }
    char *committed = g_core.commitCandidate(session_, index);
    std::wstring text = Utf8ToWide(committed);
    if (committed) {
      g_core.freeValue(committed);
    }
    if (text.empty()) {
      return;
    }

    EditSession *session = new EditSession(context, text);
    HRESULT editResult = E_FAIL;
    HRESULT hr = context->RequestEditSession(
        clientId_, session, TF_ES_READWRITE | TF_ES_SYNC, &editResult);
    session->Release();
    if (hr == TF_E_SYNCHRONOUS || FAILED(hr)) {
      EditSession *asyncSession = new EditSession(context, text);
      HRESULT ignored = E_FAIL;
      context->RequestEditSession(clientId_, asyncSession, TF_ES_READWRITE | TF_ES_ASYNC, &ignored);
      asyncSession->Release();
    }
  }

  void UpdateCandidateWindow() {
    const std::string candidates = HttpRequest(L"GET", L"/ime/candidates");
    candidateWindow_.Show(Utf8ToWide(candidates.c_str()));
  }

  long refCount_ = 1;
  ITfThreadMgr *threadMgr_ = nullptr;
  ITfKeystrokeMgr *keyMgr_ = nullptr;
  TfClientId clientId_ = TF_CLIENTID_NULL;
  uint64_t session_ = 0;
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
  HRESULT hr = WriteComRegistration(HKEY_LOCAL_MACHINE);
  if (hr == HRESULT_FROM_WIN32(ERROR_ACCESS_DENIED)) {
    hr = WriteComRegistration(HKEY_CURRENT_USER);
  }
  return hr;
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
  } else if (reason == DLL_PROCESS_DETACH) {
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
  if (clsid != kClsidTextService) {
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
