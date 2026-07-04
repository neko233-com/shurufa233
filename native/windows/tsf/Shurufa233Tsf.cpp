#include <windows.h>
#include <msctf.h>
#include <strsafe.h>

// Thin Windows TSF glue placeholder.
//
// This file intentionally contains no pinyin or ranking logic. It is the native
// edge that will translate TSF key events into calls to the Go C ABI exported
// by core/abi:
//
//   ShurufaCreateSession()
//   ShurufaInputKey(session, key)
//   ShurufaBackspace(session)
//   ShurufaSelect(session, index)
//   ShurufaFree(json)
//
// The complete TSF implementation still needs ITfTextInputProcessor,
// ITfKeyEventSink, ITfCompositionSink, and a candidate UI window. Keeping this
// file thin protects the cross-platform engine from Windows COM lifecycle code.

namespace {
constexpr wchar_t kDescription[] = L"shurufa233";
constexpr wchar_t kCLSID[] = L"{3D7B8D06-9872-4C31-B77D-3B87327CBF64}";

long g_refCount = 0;
HINSTANCE g_instance = nullptr;

HRESULT RegisterServer() {
  wchar_t modulePath[MAX_PATH]{};
  if (!GetModuleFileNameW(g_instance, modulePath, ARRAYSIZE(modulePath))) {
    return HRESULT_FROM_WIN32(GetLastError());
  }

  wchar_t clsidKey[256]{};
  HRESULT hr = StringCchPrintfW(clsidKey, ARRAYSIZE(clsidKey),
                                L"Software\\Classes\\CLSID\\%s", kCLSID);
  if (FAILED(hr)) {
    return hr;
  }

  HKEY clsid = nullptr;
  LSTATUS status = RegCreateKeyExW(HKEY_CURRENT_USER, clsidKey, 0, nullptr, 0,
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
    const wchar_t apartment[] = L"Apartment";
    RegSetValueExW(inproc, L"ThreadingModel", 0, REG_SZ,
                   reinterpret_cast<const BYTE *>(apartment),
                   sizeof(apartment));
    RegCloseKey(inproc);
  }
  RegCloseKey(clsid);
  return HRESULT_FROM_WIN32(status);
}

HRESULT UnregisterServer() {
  wchar_t clsidKey[256]{};
  HRESULT hr = StringCchPrintfW(clsidKey, ARRAYSIZE(clsidKey),
                                L"Software\\Classes\\CLSID\\%s", kCLSID);
  if (FAILED(hr)) {
    return hr;
  }
  LSTATUS status = RegDeleteTreeW(HKEY_CURRENT_USER, clsidKey);
  if (status == ERROR_FILE_NOT_FOUND) {
    return S_OK;
  }
  return HRESULT_FROM_WIN32(status);
}
}  // namespace

BOOL APIENTRY DllMain(HINSTANCE instance, DWORD reason, LPVOID) {
  if (reason == DLL_PROCESS_ATTACH) {
    g_instance = instance;
    DisableThreadLibraryCalls(instance);
  }
  return TRUE;
}

STDAPI DllCanUnloadNow() {
  return g_refCount == 0 ? S_OK : S_FALSE;
}

STDAPI DllGetClassObject(REFCLSID, REFIID, void **) {
  return E_NOTIMPL;
}

STDAPI DllRegisterServer() {
  return RegisterServer();
}

STDAPI DllUnregisterServer() {
  return UnregisterServer();
}
