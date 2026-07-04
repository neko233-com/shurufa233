#include <windows.h>
#include <msctf.h>
#include <cstdio>
#include <cwchar>

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

const CLSID kMicrosoftPinyinClsid = {
    0x81d4e9c9,
    0x1d3b,
    0x41bc,
    {0x9e, 0x6c, 0x4b, 0x40, 0xbf, 0x79, 0xe3, 0x5e}};

const GUID kMicrosoftPinyinProfile = {
    0xfa550b04,
    0x5ad7,
    0x411f,
    {0xa5, 0xac, 0xca, 0x03, 0x8e, 0xc5, 0x15, 0xd7}};

constexpr LANGID kLanguage = MAKELANGID(LANG_CHINESE, SUBLANG_CHINESE_SIMPLIFIED);

int PrintResult(const char *action, HRESULT hr) {
  std::printf("%s=0x%08X\n", action, static_cast<unsigned int>(hr));
  return SUCCEEDED(hr) ? 0 : 1;
}

void PrintGuid(const char *name, REFGUID guid) {
  wchar_t value[64]{};
  StringFromGUID2(guid, value, ARRAYSIZE(value));
  std::wprintf(L"%hs=%ls\n", name, value);
}

bool ToWide(const char *value, wchar_t *out, int outCount) {
  if (!value || !out || outCount <= 0) {
    return false;
  }
  const int written = MultiByteToWideChar(CP_UTF8, 0, value, -1, out, outCount);
  return written > 0 && written < outCount;
}

HRESULT ParseGuidArg(const char *value, GUID *guid) {
  wchar_t wide[80]{};
  if (!ToWide(value, wide, ARRAYSIZE(wide))) {
    return E_INVALIDARG;
  }
  return CLSIDFromString(wide, guid);
}

bool ParseLangIdArg(const char *value, LANGID *langid) {
  if (!value || !langid) {
    return false;
  }
  wchar_t wide[16]{};
  if (!ToWide(value, wide, ARRAYSIZE(wide))) {
    return false;
  }
  wchar_t *end = nullptr;
  unsigned long parsed = wcstoul(wide, &end, 0);
  if (end == wide || parsed > 0xffff) {
    return false;
  }
  *langid = static_cast<LANGID>(parsed);
  return true;
}

HRESULT ActivateTip(LANGID langid, REFCLSID clsid, REFGUID profileGuid) {
  HRESULT hr = S_OK;

  ITfInputProcessorProfiles *profiles = nullptr;
  hr = CoCreateInstance(CLSID_TF_InputProcessorProfiles, nullptr, CLSCTX_INPROC_SERVER,
                        IID_ITfInputProcessorProfiles,
                        reinterpret_cast<void **>(&profiles));
  if (SUCCEEDED(hr) && profiles) {
    profiles->EnableLanguageProfile(clsid, langid, profileGuid, TRUE);
    profiles->ChangeCurrentLanguage(langid);
    profiles->ActivateLanguageProfile(clsid, langid, profileGuid);
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
    hr = mgr->ActivateProfile(TF_PROFILETYPE_INPUTPROCESSOR, langid, clsid,
                              profileGuid, nullptr, flags);
    if (FAILED(hr)) {
      hr = mgr->ActivateProfile(TF_PROFILETYPE_INPUTPROCESSOR, langid, clsid,
                                profileGuid, nullptr, TF_IPPMF_FORSESSION);
    }
    mgr->Release();
  }
  return hr;
}

}  // namespace

int main(int argc, char **argv) {
  const char *command = argc > 1 ? argv[1] : "enable";
  HRESULT hr = CoInitializeEx(nullptr, COINIT_APARTMENTTHREADED);
  const bool didCoInit = SUCCEEDED(hr);
  if (FAILED(hr) && hr != RPC_E_CHANGED_MODE) {
    return PrintResult("coinitialize", hr);
  }

  if (_stricmp(command, "activate") == 0) {
    hr = ActivateTip(kLanguage, kClsidTextService, kProfileGuid);
    if (didCoInit) {
      CoUninitialize();
    }
    return PrintResult("activate", hr);
  }

  if (_stricmp(command, "activate-microsoft") == 0) {
    hr = ActivateTip(kLanguage, kMicrosoftPinyinClsid, kMicrosoftPinyinProfile);
    if (didCoInit) {
      CoUninitialize();
    }
    return PrintResult("activate-microsoft", hr);
  }

  if (_stricmp(command, "activate-tip") == 0) {
    LANGID langid = 0;
    GUID clsid{};
    GUID profileGuid{};
    if (argc < 5 || !ParseLangIdArg(argv[2], &langid) ||
        FAILED(ParseGuidArg(argv[3], &clsid)) ||
        FAILED(ParseGuidArg(argv[4], &profileGuid))) {
      if (didCoInit) {
        CoUninitialize();
      }
      std::fprintf(stderr, "usage: Shurufa233ProfileCtl.exe activate-tip <langid> <clsid> <profile>\n");
      return 2;
    }
    hr = ActivateTip(langid, clsid, profileGuid);
    if (didCoInit) {
      CoUninitialize();
    }
    return PrintResult("activate-tip", hr);
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

  if (_stricmp(command, "current") == 0) {
    ITfInputProcessorProfileMgr *mgr = nullptr;
    hr = CoCreateInstance(CLSID_TF_InputProcessorProfiles, nullptr, CLSCTX_INPROC_SERVER,
                          IID_ITfInputProcessorProfileMgr,
                          reinterpret_cast<void **>(&mgr));
    TF_INPUTPROCESSORPROFILE profile{};
    if (SUCCEEDED(hr) && mgr) {
      hr = mgr->GetActiveProfile(GUID_TFCAT_TIP_KEYBOARD, &profile);
      mgr->Release();
    }
    if (SUCCEEDED(hr)) {
      const bool isShurufa = IsEqualGUID(profile.clsid, kClsidTextService) &&
                             IsEqualGUID(profile.guidProfile, kProfileGuid);
      std::printf("current=%s\n", isShurufa ? "shurufa233" : "other");
      std::printf("langid=0x%04X\n", static_cast<unsigned int>(profile.langid));
      PrintGuid("clsid", profile.clsid);
      PrintGuid("profile", profile.guidProfile);
    }
    if (didCoInit) {
      CoUninitialize();
    }
    return PrintResult("current", hr);
  }

  std::fprintf(stderr, "usage: Shurufa233ProfileCtl.exe [enable|activate|activate-microsoft|activate-tip <langid> <clsid> <profile>|current|probe]\n");
  if (didCoInit) {
    CoUninitialize();
  }
  return 2;
}
