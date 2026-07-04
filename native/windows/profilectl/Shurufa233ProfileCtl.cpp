#include <windows.h>
#include <msctf.h>
#include <cstdio>

namespace {

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

int PrintResult(const char *action, HRESULT hr) {
  std::printf("%s=0x%08X\n", action, static_cast<unsigned int>(hr));
  return SUCCEEDED(hr) ? 0 : 1;
}

}  // namespace

int main(int argc, char **argv) {
  const char *command = argc > 1 ? argv[1] : "activate";
  HRESULT hr = CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED);
  const bool didCoInit = SUCCEEDED(hr);
  if (FAILED(hr) && hr != RPC_E_CHANGED_MODE) {
    return PrintResult("coinitialize", hr);
  }

  if (_stricmp(command, "activate") == 0) {
    ITfInputProcessorProfileMgr *mgr = nullptr;
    hr = CoCreateInstance(CLSID_TF_InputProcessorProfiles, nullptr, CLSCTX_INPROC_SERVER,
                          IID_ITfInputProcessorProfileMgr,
                          reinterpret_cast<void **>(&mgr));
    if (SUCCEEDED(hr) && mgr) {
      hr = mgr->ActivateProfile(TF_PROFILETYPE_INPUTPROCESSOR, kLanguage,
                                kClsidTextService, kProfileGuid, nullptr,
                                TF_IPPMF_FORSESSION);
      mgr->Release();
    }
    if (didCoInit) {
      CoUninitialize();
    }
    return PrintResult("activate", hr);
  }

  if (_stricmp(command, "enable") == 0) {
    ITfInputProcessorProfiles *profiles = nullptr;
    hr = CoCreateInstance(CLSID_TF_InputProcessorProfiles, nullptr, CLSCTX_INPROC_SERVER,
                          IID_ITfInputProcessorProfiles,
                          reinterpret_cast<void **>(&profiles));
    if (SUCCEEDED(hr) && profiles) {
      hr = profiles->EnableLanguageProfile(kClsidTextService, kLanguage, kProfileGuid, TRUE);
      profiles->Release();
    }
    if (didCoInit) {
      CoUninitialize();
    }
    return PrintResult("enable", hr);
  }

  if (_stricmp(command, "probe") == 0) {
    IUnknown *unknown = nullptr;
    hr = CoCreateInstance(kClsidTextService, nullptr, CLSCTX_INPROC_SERVER,
                          IID_IUnknown, reinterpret_cast<void **>(&unknown));
    if (unknown) {
      unknown->Release();
    }
    if (didCoInit) {
      CoUninitialize();
    }
    return PrintResult("probe", hr);
  }

  std::fprintf(stderr, "usage: Shurufa233ProfileCtl.exe [activate|enable|probe]\n");
  if (didCoInit) {
    CoUninitialize();
  }
  return 2;
}
