#include <windows.h>

namespace {

constexpr wchar_t kClassName[] = L"Shurufa233SmokeEditWindow";

LRESULT CALLBACK WindowProc(HWND hwnd, UINT message, WPARAM wparam, LPARAM lparam) {
  static HWND edit = nullptr;
  switch (message) {
    case WM_CREATE: {
      edit = CreateWindowExW(WS_EX_CLIENTEDGE, L"EDIT", L"",
                             WS_CHILD | WS_VISIBLE | WS_TABSTOP | ES_LEFT |
                                 ES_MULTILINE | ES_AUTOVSCROLL | WS_VSCROLL,
                             12, 12, 760, 420, hwnd, nullptr,
                             reinterpret_cast<LPCREATESTRUCTW>(lparam)->hInstance, nullptr);
      HFONT font = CreateFontW(-24, 0, 0, 0, FW_NORMAL, FALSE, FALSE, FALSE,
                               DEFAULT_CHARSET, OUT_DEFAULT_PRECIS, CLIP_DEFAULT_PRECIS,
                               CLEARTYPE_QUALITY, DEFAULT_PITCH | FF_DONTCARE,
                               L"Microsoft YaHei UI");
      SendMessageW(edit, WM_SETFONT, reinterpret_cast<WPARAM>(font), TRUE);
      SetFocus(edit);
      return 0;
    }
    case WM_SIZE:
      if (edit) {
        MoveWindow(edit, 12, 12, LOWORD(lparam) - 24, HIWORD(lparam) - 24, TRUE);
      }
      return 0;
    case WM_SETFOCUS:
      if (edit) {
        SetFocus(edit);
      }
      return 0;
    case WM_DESTROY:
      PostQuitMessage(0);
      return 0;
    default:
      return DefWindowProcW(hwnd, message, wparam, lparam);
  }
}

}  // namespace

int WINAPI wWinMain(HINSTANCE instance, HINSTANCE, PWSTR, int show) {
  WNDCLASSW wc{};
  wc.lpfnWndProc = WindowProc;
  wc.hInstance = instance;
  wc.hCursor = LoadCursorW(nullptr, IDC_IBEAM);
  wc.lpszClassName = kClassName;
  RegisterClassW(&wc);

  HWND hwnd = CreateWindowExW(0, kClassName, L"shurufa233 smoke edit",
                              WS_OVERLAPPEDWINDOW, CW_USEDEFAULT, CW_USEDEFAULT,
                              820, 520, nullptr, nullptr, instance, nullptr);
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
