#include <windows.h>
#include <windowsx.h>
#include <dwmapi.h>
#include <msctf.h>
#include <strsafe.h>
#include <winhttp.h>

#include <cstdint>
#include <cctype>
#include <cstdio>
#include <cstring>
#include <fstream>
#include <iterator>
#include <string>
#include <unordered_map>
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
constexpr int kDefaultCandidatesPerPage = 7;
constexpr int kMinCandidatesPerPage = 3;
constexpr int kMaxCandidatesPerPage = 9;
constexpr DWORD kSkinConfigPollMs = 250;
constexpr DWORD kHttpSkinPollMs = 2000;

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
using CandidatePayloadRangeFn = char *(*)(uint64_t, int, int);
using ClearSessionFn = char *(*)(uint64_t);
using SetModeFn = char *(*)(uint64_t, const char *);
using ToggleModeFn = char *(*)(uint64_t);
using ModeFn = char *(*)(uint64_t);
using CommitCandidateFn = char *(*)(uint64_t, int);
using CommitCandidateCharFn = char *(*)(uint64_t, int, const char *);
using RejectCandidateFn = char *(*)(uint64_t, int);
using PinCandidateFn = char *(*)(uint64_t, int);
using FreeFn = void (*)(char *);
using CoreStringFn = char *(*)();
using SessionStringFn = char *(*)(uint64_t);
using CandidatePayloadV2Fn = char *(*)(uint64_t, int, int);
using AssociateFn = char *(*)(uint64_t, const char *);
using CandidateActionFn = char *(*)(uint64_t, const char *);
using KeyEventJsonFn = char *(*)(uint64_t, const char *);
using ReverseLookupFn = char *(*)(uint64_t, const char *);
using ApplyConfigJsonFn = char *(*)(char *);
using ImportUserScoresJsonFn = char *(*)(uint64_t, char *);
using ImportUserPhrasesJsonFn = char *(*)(uint64_t, char *);
using ImportUserRejectsJsonFn = char *(*)(uint64_t, char *);
using ImportUserPinsJsonFn = char *(*)(uint64_t, char *);
using CommitTextFn = char *(*)(uint64_t, char *, char *);
using AgentComposeFn = char *(*)(char *, char *);
using ExecuteCommandFn = char *(*)(uint64_t, const char *, const char *);
using SessionPayloadFn = char *(*)(uint64_t, const char *);

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
  CandidatePayloadRangeFn candidatePayloadRange = nullptr;
  ClearSessionFn clearSession = nullptr;
  SetModeFn setMode = nullptr;
  ToggleModeFn toggleMode = nullptr;
  ModeFn mode = nullptr;
  CommitCandidateFn commitCandidate = nullptr;
  CommitCandidateCharFn commitCandidateChar = nullptr;
  RejectCandidateFn rejectCandidate = nullptr;
  PinCandidateFn pinCandidate = nullptr;
  FreeFn freeValue = nullptr;
  CoreStringFn abiVersion = nullptr;
  CoreStringFn capabilities = nullptr;
  SessionStringFn stateJson = nullptr;
  CandidatePayloadV2Fn candidatePayloadV2 = nullptr;
  AssociateFn associate = nullptr;
  CandidateActionFn candidateAction = nullptr;
  KeyEventJsonFn keyEventJson = nullptr;
  ReverseLookupFn reverseLookup = nullptr;
  CoreStringFn dictionarySourcesJson = nullptr;
  CoreStringFn configJson = nullptr;
  ApplyConfigJsonFn applyConfigJson = nullptr;
  CoreStringFn reloadConfig = nullptr;
  CoreStringFn reloadDictionaries = nullptr;
  CoreStringFn dictionaryManifestJson = nullptr;
  SessionStringFn recognizerPatternsJson = nullptr;
  SessionPayloadFn applyAppRulesJson = nullptr;
  SessionStringFn userScoresJson = nullptr;
  ImportUserScoresJsonFn importUserScoresJson = nullptr;
  SessionStringFn userPhrasesJson = nullptr;
  ImportUserPhrasesJsonFn importUserPhrasesJson = nullptr;
  SessionPayloadFn deleteUserPhraseJson = nullptr;
  SessionStringFn userRejectsJson = nullptr;
  ImportUserRejectsJsonFn importUserRejectsJson = nullptr;
  SessionPayloadFn deleteUserRejectJson = nullptr;
  SessionStringFn userPinsJson = nullptr;
  ImportUserPinsJsonFn importUserPinsJson = nullptr;
  SessionPayloadFn deleteUserPinJson = nullptr;
  CoreStringFn agentConfigJson = nullptr;
  ApplyConfigJsonFn applyAgentConfigJson = nullptr;
  CoreStringFn syncConfigJson = nullptr;
  ApplyConfigJsonFn applySyncConfigJson = nullptr;
  SessionPayloadFn exportProfileSyncJson = nullptr;
  SessionPayloadFn importProfileSyncJson = nullptr;
  CommitTextFn commitText = nullptr;
  AgentComposeFn agentCompose = nullptr;
  ExecuteCommandFn executeCommand = nullptr;

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

bool IsDebugLogEnabled() {
  static const bool enabled = []() {
    wchar_t value[16]{};
    const DWORD len = GetEnvironmentVariableW(L"SHURUFA233_TSF_DEBUG", value, ARRAYSIZE(value));
    return len > 0 && value[0] != L'\0' && value[0] != L'0';
  }();
  return enabled;
}

void LogDebugLine(const wchar_t *message) {
  if (IsDebugLogEnabled()) {
    LogLine(message);
  }
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

char *HttpSetModeValue(uint64_t session, const char *mode) {
  const bool english = mode && _stricmp(mode, "en") == 0;
  wchar_t path[80]{};
  StringCchPrintfW(path, ARRAYSIZE(path), L"/ime/mode?session=%llu&mode=%s", session,
                   english ? L"en" : L"zh");
  return AllocCString(HttpRequest(L"POST", path));
}

char *HttpToggleModeValue(uint64_t session) {
  wchar_t path[80]{};
  StringCchPrintfW(path, ARRAYSIZE(path), L"/ime/mode?session=%llu&toggle=1", session);
  return AllocCString(HttpRequest(L"POST", path));
}

char *HttpModeValue(uint64_t session) {
  wchar_t path[80]{};
  StringCchPrintfW(path, ARRAYSIZE(path), L"/ime/mode?session=%llu", session);
  const std::string state = HttpRequest(L"GET", path);
  return AllocCString(state.find("\"mode\":\"en\"") != std::string::npos ? "en" : "zh");
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
  api.candidatePayloadRange =
      LoadCoreProc<CandidatePayloadRangeFn>(module, "ShurufaCandidatePayloadRange");
  api.clearSession = LoadCoreProc<ClearSessionFn>(module, "ShurufaClear");
  api.setMode = LoadCoreProc<SetModeFn>(module, "ShurufaSetMode");
  api.toggleMode = LoadCoreProc<ToggleModeFn>(module, "ShurufaToggleMode");
  api.mode = LoadCoreProc<ModeFn>(module, "ShurufaMode");
  api.commitCandidate = LoadCoreProc<CommitCandidateFn>(module, "ShurufaCommitCandidate");
  api.commitCandidateChar =
      LoadCoreProc<CommitCandidateCharFn>(module, "ShurufaCommitCandidateChar");
  api.rejectCandidate = LoadCoreProc<RejectCandidateFn>(module, "ShurufaRejectCandidate");
  api.pinCandidate = LoadCoreProc<PinCandidateFn>(module, "ShurufaPinCandidate");
  api.freeValue = LoadCoreProc<FreeFn>(module, "ShurufaFree");
  api.abiVersion = LoadCoreProc<CoreStringFn>(module, "ShurufaAbiVersion");
  api.capabilities = LoadCoreProc<CoreStringFn>(module, "ShurufaCapabilities");
  api.stateJson = LoadCoreProc<SessionStringFn>(module, "ShurufaState");
  api.candidatePayloadV2 = LoadCoreProc<CandidatePayloadV2Fn>(module, "ShurufaCandidatePayloadV2");
  api.associate = LoadCoreProc<AssociateFn>(module, "ShurufaAssociate");
  api.candidateAction = LoadCoreProc<CandidateActionFn>(module, "ShurufaCandidateAction");
  api.keyEventJson = LoadCoreProc<KeyEventJsonFn>(module, "ShurufaKeyEventJSON");
  api.reverseLookup = LoadCoreProc<ReverseLookupFn>(module, "ShurufaReverseLookupJSON");
  api.dictionarySourcesJson = LoadCoreProc<CoreStringFn>(module, "ShurufaDictionarySourcesJSON");
  api.configJson = LoadCoreProc<CoreStringFn>(module, "ShurufaConfigJSON");
  api.applyConfigJson = LoadCoreProc<ApplyConfigJsonFn>(module, "ShurufaApplyConfigJSON");
  api.reloadConfig = LoadCoreProc<CoreStringFn>(module, "ShurufaReloadConfig");
  api.reloadDictionaries = LoadCoreProc<CoreStringFn>(module, "ShurufaReloadDictionaries");
  api.dictionaryManifestJson =
      LoadCoreProc<CoreStringFn>(module, "ShurufaDictionaryManifestJSON");
  api.recognizerPatternsJson = LoadCoreProc<SessionStringFn>(module, "ShurufaRecognizerPatternsJSON");
  api.applyAppRulesJson = LoadCoreProc<SessionPayloadFn>(module, "ShurufaApplyAppRulesJSON");
  api.userScoresJson = LoadCoreProc<SessionStringFn>(module, "ShurufaUserScoresJSON");
  api.importUserScoresJson =
      LoadCoreProc<ImportUserScoresJsonFn>(module, "ShurufaImportUserScoresJSON");
  api.userPhrasesJson = LoadCoreProc<SessionStringFn>(module, "ShurufaUserPhrasesJSON");
  api.importUserPhrasesJson =
      LoadCoreProc<ImportUserPhrasesJsonFn>(module, "ShurufaImportUserPhrasesJSON");
  api.deleteUserPhraseJson = LoadCoreProc<SessionPayloadFn>(module, "ShurufaDeleteUserPhraseJSON");
  api.userRejectsJson = LoadCoreProc<SessionStringFn>(module, "ShurufaUserRejectsJSON");
  api.importUserRejectsJson =
      LoadCoreProc<ImportUserRejectsJsonFn>(module, "ShurufaImportUserRejectsJSON");
  api.deleteUserRejectJson = LoadCoreProc<SessionPayloadFn>(module, "ShurufaDeleteUserRejectJSON");
  api.userPinsJson = LoadCoreProc<SessionStringFn>(module, "ShurufaUserPinsJSON");
  api.importUserPinsJson =
      LoadCoreProc<ImportUserPinsJsonFn>(module, "ShurufaImportUserPinsJSON");
  api.deleteUserPinJson = LoadCoreProc<SessionPayloadFn>(module, "ShurufaDeleteUserPinJSON");
  api.agentConfigJson = LoadCoreProc<CoreStringFn>(module, "ShurufaAgentConfigJSON");
  api.applyAgentConfigJson = LoadCoreProc<ApplyConfigJsonFn>(module, "ShurufaApplyAgentConfigJSON");
  api.syncConfigJson = LoadCoreProc<CoreStringFn>(module, "ShurufaSyncConfigJSON");
  api.applySyncConfigJson = LoadCoreProc<ApplyConfigJsonFn>(module, "ShurufaApplySyncConfigJSON");
  api.exportProfileSyncJson = LoadCoreProc<SessionPayloadFn>(module, "ShurufaExportProfileSyncJSON");
  api.importProfileSyncJson = LoadCoreProc<SessionPayloadFn>(module, "ShurufaImportProfileSyncJSON");
  api.commitText = LoadCoreProc<CommitTextFn>(module, "ShurufaCommitText");
  api.agentCompose = LoadCoreProc<AgentComposeFn>(module, "ShurufaAgentCompose");
  api.executeCommand = LoadCoreProc<ExecuteCommandFn>(module, "ShurufaExecuteCommand");
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
  g_core.candidatePayloadRange = nullptr;
  g_core.clearSession = HttpClearSessionValue;
  g_core.setMode = HttpSetModeValue;
  g_core.toggleMode = HttpToggleModeValue;
  g_core.mode = HttpModeValue;
  g_core.commitCandidate = HttpCommitCandidate;
  g_core.commitCandidateChar = nullptr;
  g_core.rejectCandidate = nullptr;
  g_core.pinCandidate = nullptr;
  g_core.freeValue = HttpFree;
  g_core.abiVersion = nullptr;
  g_core.capabilities = nullptr;
  g_core.stateJson = nullptr;
  g_core.candidatePayloadV2 = nullptr;
  g_core.associate = nullptr;
  g_core.candidateAction = nullptr;
  g_core.keyEventJson = nullptr;
  g_core.reverseLookup = nullptr;
  g_core.dictionarySourcesJson = nullptr;
  g_core.configJson = nullptr;
  g_core.applyConfigJson = nullptr;
  g_core.reloadConfig = nullptr;
  g_core.reloadDictionaries = nullptr;
  g_core.dictionaryManifestJson = nullptr;
  g_core.recognizerPatternsJson = nullptr;
  g_core.applyAppRulesJson = nullptr;
  g_core.userScoresJson = nullptr;
  g_core.importUserScoresJson = nullptr;
  g_core.userPhrasesJson = nullptr;
  g_core.importUserPhrasesJson = nullptr;
  g_core.deleteUserPhraseJson = nullptr;
  g_core.userRejectsJson = nullptr;
  g_core.importUserRejectsJson = nullptr;
  g_core.deleteUserRejectJson = nullptr;
  g_core.userPinsJson = nullptr;
  g_core.importUserPinsJson = nullptr;
  g_core.deleteUserPinJson = nullptr;
  g_core.agentConfigJson = nullptr;
  g_core.applyAgentConfigJson = nullptr;
  g_core.syncConfigJson = nullptr;
  g_core.applySyncConfigJson = nullptr;
  g_core.exportProfileSyncJson = nullptr;
  g_core.importProfileSyncJson = nullptr;
  g_core.commitText = nullptr;
  g_core.agentCompose = nullptr;
  g_core.executeCommand = nullptr;
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

bool ModePayloadIsEnglish(const char *value) {
  if (!value || !*value) {
    return false;
  }
  const std::string payload(value);
  return payload == "en" || payload.find("\"mode\":\"en\"") != std::string::npos;
}

std::string JsonEscape(const std::string &value) {
  std::string out;
  out.reserve(value.size() + 8);
  for (char ch : value) {
    switch (ch) {
      case '\\':
        out += "\\\\";
        break;
      case '"':
        out += "\\\"";
        break;
      case '\b':
        out += "\\b";
        break;
      case '\f':
        out += "\\f";
        break;
      case '\n':
        out += "\\n";
        break;
      case '\r':
        out += "\\r";
        break;
      case '\t':
        out += "\\t";
        break;
      default:
        out.push_back(ch);
        break;
    }
  }
  return out;
}

bool JsonBoolFieldValue(const std::string &json, const char *field, bool fallback) {
  const std::string key = std::string("\"") + field + "\"";
  size_t pos = json.find(key);
  if (pos == std::string::npos) {
    return fallback;
  }
  pos = json.find(':', pos + key.size());
  if (pos == std::string::npos) {
    return fallback;
  }
  size_t start = pos + 1;
  while (start < json.size() && isspace(static_cast<unsigned char>(json[start]))) {
    ++start;
  }
  size_t end = start;
  while (end < json.size() && json[end] != ',' && json[end] != '}' &&
         json[end] != '\n' && json[end] != '\r') {
    ++end;
  }
  std::string value = json.substr(start, end - start);
  std::string normalized;
  normalized.reserve(value.size());
  for (char ch : value) {
    if (!isspace(static_cast<unsigned char>(ch))) {
      normalized.push_back(static_cast<char>(tolower(static_cast<unsigned char>(ch))));
    }
  }
  if (normalized == "true" || normalized == "1" || normalized == "yes" || normalized == "on") {
    return true;
  }
  if (normalized == "false" || normalized == "0" || normalized == "no" || normalized == "off") {
    return false;
  }
  return fallback;
}

std::vector<std::string> SplitTabFields(const std::string &line) {
  std::vector<std::string> fields;
  size_t start = 0;
  while (start <= line.size()) {
    size_t end = line.find('\t', start);
    if (end == std::string::npos) {
      fields.push_back(line.substr(start));
      break;
    }
    fields.push_back(line.substr(start, end - start));
    start = end + 1;
  }
  return fields;
}

bool PayloadBoolField(const std::string &value) {
  std::string normalized;
  normalized.reserve(value.size());
  for (char ch : value) {
    normalized.push_back(static_cast<char>(tolower(static_cast<unsigned char>(ch))));
  }
  return normalized == "true" || normalized == "1" || normalized == "yes" || normalized == "on";
}

bool IsWindowsInputExperienceWindow(HWND hwnd) {
  if (!IsWindowVisible(hwnd)) {
    return false;
  }
  wchar_t className[128]{};
  wchar_t title[256]{};
  GetClassNameW(hwnd, className, ARRAYSIZE(className));
  GetWindowTextW(hwnd, title, ARRAYSIZE(title));
  if (lstrcmpiW(className, L"Windows.UI.Core.CoreWindow") != 0) {
    return false;
  }
  const std::wstring titleText(title);
  return titleText.find(L"Windows 输入体验") != std::wstring::npos ||
         titleText.find(L"Windows Input Experience") != std::wstring::npos;
}

BOOL CALLBACK HideWindowsInputExperienceProc(HWND hwnd, LPARAM) {
  if (IsWindowsInputExperienceWindow(hwnd)) {
    ShowWindow(hwnd, SW_HIDE);
  }
  return TRUE;
}

void HideWindowsInputExperienceResidue() {
  EnumWindows(HideWindowsInputExperienceProc, 0);
}

class CandidateWindow {
 public:
  using CandidateClickHandler = void (*)(void *, int);
  using CandidateSelectHandler = void (*)(void *, int);
  using CandidatePageHandler = void (*)(void *, int);
  using CandidateMenuHandler = void (*)(void *, int, HWND, POINT);

  struct CandidateView {
    int index = 0;
    std::wstring text;
    std::wstring reading;
    std::wstring kind;
    std::wstring source;
    std::wstring comment;
    int score = 0;
    bool pinned = false;
  };

  ~CandidateWindow() {
    ResetFont();
  }

  void SetClickHandler(void *owner, CandidateClickHandler handler) {
    clickOwner_ = owner;
    clickHandler_ = handler;
  }

  void SetSelectHandler(void *owner, CandidateSelectHandler handler) {
    selectOwner_ = owner;
    selectHandler_ = handler;
  }

  void SetPageHandler(void *owner, CandidatePageHandler handler) {
    pageOwner_ = owner;
    pageHandler_ = handler;
  }

  void SetMenuHandler(void *owner, CandidateMenuHandler handler) {
    menuOwner_ = owner;
    menuHandler_ = handler;
  }

  void Show(const std::string &payload, int selectedIndex, int pageStart, int totalCount,
            int pageSize,
            const std::wstring &compositionText) {
    candidates_ = ParseCandidates(payload);
    if (candidates_.empty()) {
      Hide();
      return;
    }
    if (hwnd_) {
      KillTimer(hwnd_, kStatusTimerId);
    }
    selectedIndex_ = max(0, min(selectedIndex, static_cast<int>(candidates_.size()) - 1));
    pageStart_ = max(0, pageStart);
    pageSize_ = max(kMinCandidatesPerPage, min(kMaxCandidatesPerPage, pageSize));
    totalCount_ = max(static_cast<int>(candidates_.size()), totalCount);
    composing_ = compositionText.empty() ? CompositionText() : compositionText;
    EnsureWindow();
    RefreshDpi();
    RefreshSkin();

    POINT anchor = CaretAnchor();
    const int width = MeasureWindowWidth();
    const int height = CandidateWindowHeight();
    const POINT origin = FitToWorkArea(anchor, width, height);
    ArmHoverGuard();
    HideWindowsInputExperienceResidue();
    SetWindowPos(hwnd_, HWND_TOPMOST, origin.x, origin.y, width, height,
                 SWP_NOACTIVATE | SWP_SHOWWINDOW);
    HideWindowsInputExperienceResidue();
    InvalidateRect(hwnd_, nullptr, TRUE);
  }

  void Hide() {
    if (hwnd_) {
      KillTimer(hwnd_, kStatusTimerId);
      ShowWindow(hwnd_, SW_HIDE);
    }
    composing_.clear();
    candidateHits_.clear();
    pageHits_.clear();
    hoverGuardArmed_ = false;
  }

  void ShowStatus(const wchar_t *text) {
    EnsureWindow();
    RefreshDpi();
    RefreshSkin();
    statusText_ = text ? text : L"";
    candidates_.clear();
    candidateHits_.clear();
    pageHits_.clear();
    composing_.clear();
    pageStart_ = 0;
    totalCount_ = 0;
    POINT anchor = CaretAnchor();
    const int width = MeasureStatusWidth();
    const int height = max(Scale(42), ScaledFontSize() + Scale(28));
    const POINT origin = FitToWorkArea(anchor, width, height);
    HideWindowsInputExperienceResidue();
    SetWindowPos(hwnd_, HWND_TOPMOST, origin.x, origin.y, width, height,
                 SWP_NOACTIVATE | SWP_SHOWWINDOW);
    HideWindowsInputExperienceResidue();
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
    if (self && message == WM_DPICHANGED) {
      self->RefreshDpi(LOWORD(wparam));
      InvalidateRect(hwnd, nullptr, TRUE);
      return 0;
    }
    if (self && message == WM_TIMER && wparam == kStatusTimerId) {
      self->Hide();
      return 0;
    }
    if (self && message == WM_LBUTTONDOWN) {
      const POINT point{GET_X_LPARAM(lparam), GET_Y_LPARAM(lparam)};
      const int absoluteIndex = self->HitTestCandidate(point);
      if (absoluteIndex >= 0 && self->clickHandler_) {
        self->clickHandler_(self->clickOwner_, absoluteIndex);
      } else if (self->pageHandler_) {
        const int pageDelta = self->HitTestPage(point);
        if (pageDelta != 0) {
          self->pageHandler_(self->pageOwner_, pageDelta);
        }
      }
      return 0;
    }
    if (self && (message == WM_RBUTTONUP || message == WM_CONTEXTMENU)) {
      POINT point{};
      if (message == WM_CONTEXTMENU && lparam != -1) {
        point = POINT{GET_X_LPARAM(lparam), GET_Y_LPARAM(lparam)};
        ScreenToClient(hwnd, &point);
      } else {
        point = POINT{GET_X_LPARAM(lparam), GET_Y_LPARAM(lparam)};
      }
      const int absoluteIndex = self->HitTestCandidate(point);
      if (absoluteIndex >= 0 && self->menuHandler_) {
        self->SelectCandidate(absoluteIndex);
        POINT screenPoint = point;
        ClientToScreen(hwnd, &screenPoint);
        self->menuHandler_(self->menuOwner_, absoluteIndex, hwnd, screenPoint);
      }
      return 0;
    }
    if (self && message == WM_MOUSEMOVE) {
      const POINT point{GET_X_LPARAM(lparam), GET_Y_LPARAM(lparam)};
      if (self->ShouldIgnoreHoverMove(point)) {
        return 0;
      }
      const int absoluteIndex = self->HitTestCandidate(point);
      if (absoluteIndex >= 0) {
        self->SelectCandidate(absoluteIndex);
      }
      return 0;
    }
    if (self && message == WM_MOUSEWHEEL) {
      if (self->pageHandler_) {
        const int delta = GET_WHEEL_DELTA_WPARAM(wparam);
        self->pageHandler_(self->pageOwner_, delta < 0 ? 1 : -1);
      }
      return 0;
    }
    if (self && message == WM_SETCURSOR) {
      POINT cursor{};
      GetCursorPos(&cursor);
      ScreenToClient(hwnd, &cursor);
      if (self->HitTestCandidate(cursor) >= 0 || self->HitTestPage(cursor) != 0) {
        SetCursor(LoadCursorW(nullptr, IDC_HAND));
        return TRUE;
      }
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
      RefreshDpi();
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

    HBRUSH bg = CreateSolidBrush(PreeditSurfaceColor());
    FillRect(dc, &rect, bg);
    DeleteObject(bg);

    if (statusText_.empty() && !candidates_.empty()) {
      RECT candidateBand{rect.left, CandidateBandTop(), rect.right, rect.bottom};
      HBRUSH candidateBg = CreateSolidBrush(CandidateBandColor());
      FillRect(dc, &candidateBand, candidateBg);
      DeleteObject(candidateBg);

      HPEN separator = CreatePen(PS_SOLID, max(1, Scale(1)), MixColor(border_, CandidateBandColor(), 28));
      HGDIOBJ oldSeparator = SelectObject(dc, separator);
      MoveToEx(dc, rect.left + Scale(14), candidateBand.top, nullptr);
      LineTo(dc, rect.right - Scale(14), candidateBand.top);
      SelectObject(dc, oldSeparator);
      DeleteObject(separator);
    }

    HPEN border = CreatePen(PS_SOLID, max(1, Scale(1)),
                            statusText_.empty() ? PreeditBorderColor() : border_);
    HGDIOBJ oldPen = SelectObject(dc, border);
    HGDIOBJ oldBrush = SelectObject(dc, GetStockObject(HOLLOW_BRUSH));
    const int windowRadius = Scale(12);
    RoundRect(dc, rect.left, rect.top, rect.right, rect.bottom, windowRadius, windowRadius);
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
  struct CandidateHit {
    RECT rect{};
    int absoluteIndex = -1;
  };
  struct PageHit {
    RECT rect{};
    int delta = 0;
  };
  static constexpr UINT_PTR kStatusTimerId = 1;
  std::vector<CandidateView> candidates_;
  std::vector<CandidateHit> candidateHits_;
  std::vector<PageHit> pageHits_;
  void *clickOwner_ = nullptr;
  CandidateClickHandler clickHandler_ = nullptr;
  void *selectOwner_ = nullptr;
  CandidateSelectHandler selectHandler_ = nullptr;
  void *pageOwner_ = nullptr;
  CandidatePageHandler pageHandler_ = nullptr;
  void *menuOwner_ = nullptr;
  CandidateMenuHandler menuHandler_ = nullptr;
  std::wstring composing_;
  std::wstring statusText_;
  int selectedIndex_ = 0;
  int pageStart_ = 0;
  int totalCount_ = 0;
  int pageSize_ = kDefaultCandidatesPerPage;
  POINT hoverGuardScreenPoint_{};
  bool hoverGuardArmed_ = false;
  std::wstring fontFamily_ = L"Microsoft YaHei UI";
  int fontSize_ = 18;
  COLORREF accent_ = RGB(37, 99, 235);
  COLORREF surface_ = RGB(255, 255, 255);
  COLORREF text_ = RGB(17, 24, 39);
  COLORREF mutedText_ = RGB(100, 116, 139);
  COLORREF border_ = RGB(209, 213, 219);
  COLORREF highlightText_ = RGB(255, 255, 255);
  std::string theme_ = "system";
  std::string layout_ = "horizontal";
  bool showComments_ = true;
  std::wstring skinConfigPath_;
  bool skinConfigPathResolved_ = false;
  DWORD lastLocalSkinCheckTick_ = 0;
  DWORD lastHttpSkinRefreshTick_ = 0;
  FILETIME lastSkinConfigWriteTime_{};
  bool hasSkinConfigWriteTime_ = false;
  HFONT font_ = nullptr;
  std::wstring fontFamilyKey_;
  int fontSizeKey_ = 0;
  UINT fontDpiKey_ = 0;
  UINT dpi_ = 96;

  int CandidateWindowHeight() const {
    if (IsVerticalLayout()) {
      const int rows = max(1, min(static_cast<int>(candidates_.size()), pageSize_));
      const int itemHeight = max(Scale(34), ScaledFontSize() + Scale(20));
      return max(Scale(112), CandidateBandTop() + Scale(8) + rows * (itemHeight + Scale(7)) + Scale(10));
    }
    return max(Scale(82), ScaledFontSize() * 2 + Scale(56));
  }

  int CandidateBandTop() const {
    return ScaledFontSize() + Scale(24);
  }

  bool IsVerticalLayout() const {
    return layout_ == "vertical";
  }

  static std::string NormalizeCandidateLayout(const std::string &layout) {
    std::string value;
    value.reserve(layout.size());
    for (char ch : layout) {
      value.push_back(static_cast<char>(tolower(static_cast<unsigned char>(ch))));
    }
    if (value == "vertical" || value == "rime") {
      return "vertical";
    }
    if (value == "auto") {
      return "auto";
    }
    return "horizontal";
  }

  UINT ReadCurrentDpi() const {
    if (hwnd_) {
      const UINT dpi = GetDpiForWindow(hwnd_);
      if (dpi > 0) {
        return dpi;
      }
    }
    HDC dc = GetDC(nullptr);
    const int dpi = dc ? GetDeviceCaps(dc, LOGPIXELSX) : 96;
    if (dc) {
      ReleaseDC(nullptr, dc);
    }
    return dpi > 0 ? static_cast<UINT>(dpi) : 96;
  }

  void RefreshDpi(UINT dpi = 0) {
    const UINT nextDpi = dpi > 0 ? dpi : ReadCurrentDpi();
    const UINT normalizedDpi = nextDpi > 0 ? nextDpi : 96;
    if (dpi_ == normalizedDpi) {
      return;
    }
    dpi_ = normalizedDpi;
    ResetFont();
  }

  int Scale(int value) const {
    return MulDiv(value, static_cast<int>(dpi_), 96);
  }

  int ScaledFontSize() const {
    return max(1, Scale(fontSize_));
  }

  HFONT EnsureFont() {
    const UINT dpi = dpi_;
    if (font_ && fontFamilyKey_ == fontFamily_ && fontSizeKey_ == fontSize_ &&
        fontDpiKey_ == dpi) {
      return font_;
    }
    ResetFont();
    fontFamilyKey_ = fontFamily_;
    fontSizeKey_ = fontSize_;
    fontDpiKey_ = dpi;
    font_ = CreateFontW(-ScaledFontSize(), 0, 0, 0, FW_NORMAL, FALSE, FALSE, FALSE,
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
    fontDpiKey_ = 0;
  }

  int TextWidth(HDC dc, const std::wstring &value) const {
    if (value.empty()) {
      return 0;
    }
    SIZE size{};
    if (!GetTextExtentPoint32W(dc, value.c_str(), static_cast<int>(value.size()), &size)) {
      return static_cast<int>(value.size()) * ScaledFontSize();
    }
    return size.cx;
  }

  int CandidateBadgeWidth(HDC dc, const std::wstring &kindLabel) const {
    if (kindLabel.empty()) {
      return 0;
    }
    return max(Scale(30), TextWidth(dc, kindLabel) + Scale(12));
  }

  int CandidateItemWidth(HDC dc, const CandidateView &candidate, bool selected) const {
    const int textWidth = TextWidth(dc, candidate.text);
    const int commentWidth = !showComments_ || candidate.comment.empty() ? 0 : min(Scale(88), TextWidth(dc, candidate.comment) + Scale(10));
    const std::wstring kindLabel = CandidateKindLabel(candidate.kind);
    const int kindWidth = kindLabel.empty() ? 0 : CandidateBadgeWidth(dc, kindLabel) + Scale(10);
    return max(Scale(66), min(Scale(320), Scale(48) + textWidth + commentWidth + kindWidth));
  }

  bool HasPageControls() const {
    return totalCount_ > static_cast<int>(candidates_.size());
  }

  int PageControlsWidth() const {
    return HasPageControls() ? Scale(168) : 0;
  }

  int MeasureWindowWidth() {
    HDC dc = GetDC(hwnd_);
    HGDIOBJ oldFont = SelectObject(dc, EnsureFont());
    int width = max(Scale(260), TextWidth(dc, composing_) + Scale(44));
    if (IsVerticalLayout()) {
      for (size_t i = 0; i < candidates_.size() && i < static_cast<size_t>(pageSize_); ++i) {
        width = max(width, CandidateItemWidth(dc, candidates_[i], static_cast<int>(i) == selectedIndex_) + Scale(38) + PageControlsWidth());
      }
    } else {
      for (size_t i = 0; i < candidates_.size() && i < static_cast<size_t>(pageSize_); ++i) {
        width += CandidateItemWidth(dc, candidates_[i], static_cast<int>(i) == selectedIndex_) + Scale(6);
      }
      width += PageControlsWidth();
    }
    SelectObject(dc, oldFont);
    ReleaseDC(hwnd_, dc);
    return max(Scale(180), min(IsVerticalLayout() ? Scale(460) : Scale(780), width));
  }

  int MeasureStatusWidth() {
    HDC dc = GetDC(hwnd_);
    HGDIOBJ oldFont = SelectObject(dc, EnsureFont());
    const int width = max(Scale(92), min(Scale(180), TextWidth(dc, statusText_) + Scale(40)));
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
    const int margin = Scale(8);
    POINT origin{anchor.x, anchor.y + margin};
    origin.x = max(work.left + margin, min(origin.x, work.right - width - margin));
    if (origin.y + height > work.bottom - margin) {
      origin.y = anchor.y - height - margin;
    }
    origin.y = max(work.top + margin, min(origin.y, work.bottom - height - margin));
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

  static int ColorLuminance(COLORREF color) {
    return (GetRValue(color) * 299 + GetGValue(color) * 587 + GetBValue(color) * 114) / 1000;
  }

  bool IsDarkSkin() const {
    return theme_ == "dark" || ColorLuminance(surface_) < 96;
  }

  COLORREF PreeditSurfaceColor() const {
    return IsDarkSkin() ? RGB(36, 36, 36) : RGB(250, 250, 250);
  }

  COLORREF PreeditTextColor() const {
    return IsDarkSkin() ? RGB(245, 245, 245) : RGB(30, 30, 30);
  }

  COLORREF PreeditBorderColor() const {
    return IsDarkSkin() ? RGB(78, 78, 78) : RGB(218, 220, 224);
  }

  COLORREF PreeditUnderlineColor() const {
    return IsDarkSkin() ? RGB(150, 150, 150) : RGB(95, 99, 104);
  }

  COLORREF CandidateBandColor() const {
    const COLORREF base = IsDarkSkin() ? RGB(30, 32, 36) : RGB(255, 255, 255);
    return MixColor(surface_, base, 18);
  }

  COLORREF CandidateIdleColor() const {
    return IsDarkSkin() ? MixColor(surface_, RGB(255, 255, 255), 5)
                            : MixColor(surface_, accent_, 4);
  }

  COLORREF CandidateIdleBorderColor() const {
    return IsDarkSkin() ? MixColor(border_, RGB(255, 255, 255), 8)
                            : MixColor(border_, surface_, 36);
  }

  COLORREF CandidateAccentEdgeColor() const {
    return MixColor(accent_, RGB(255, 255, 255), IsDarkSkin() ? 18 : 10);
  }

  std::wstring SkinConfigPath() {
    if (skinConfigPathResolved_) {
      return skinConfigPath_;
    }
    skinConfigPathResolved_ = true;
    wchar_t appData[MAX_PATH]{};
    const DWORD len = GetEnvironmentVariableW(L"APPDATA", appData, ARRAYSIZE(appData));
    if (len == 0 || len >= ARRAYSIZE(appData)) {
      return L"";
    }
    skinConfigPath_ = std::wstring(appData) + L"\\shurufa233\\config.json";
    return skinConfigPath_;
  }

  static bool SameFileTime(const FILETIME &left, const FILETIME &right) {
    return left.dwLowDateTime == right.dwLowDateTime &&
           left.dwHighDateTime == right.dwHighDateTime;
  }

  static std::string ReadUtf8File(const std::wstring &path) {
    std::ifstream file(path, std::ios::binary);
    if (!file) {
      return "";
    }
    return std::string(std::istreambuf_iterator<char>(file),
                       std::istreambuf_iterator<char>());
  }

  static std::string JsonStringField(const std::string &json, const char *field) {
    const std::string key = std::string("\"") + field + "\"";
    size_t pos = json.find(key);
    if (pos == std::string::npos) {
      return "";
    }
    pos = json.find(':', pos + key.size());
    if (pos == std::string::npos) {
      return "";
    }
    pos = json.find('"', pos + 1);
    if (pos == std::string::npos) {
      return "";
    }
    std::string out;
    bool escaped = false;
    for (size_t i = pos + 1; i < json.size(); ++i) {
      const char ch = json[i];
      if (escaped) {
        out.push_back(ch);
        escaped = false;
        continue;
      }
      if (ch == '\\') {
        escaped = true;
        continue;
      }
      if (ch == '"') {
        return out;
      }
      out.push_back(ch);
    }
    return "";
  }

  static int JsonIntField(const std::string &json, const char *field) {
    const std::string key = std::string("\"") + field + "\"";
    size_t pos = json.find(key);
    if (pos == std::string::npos) {
      return 0;
    }
    pos = json.find(':', pos + key.size());
    if (pos == std::string::npos) {
      return 0;
    }
    while (pos + 1 < json.size() && isspace(static_cast<unsigned char>(json[pos + 1]))) {
      ++pos;
    }
    return atoi(json.c_str() + pos + 1);
  }

  static bool ParseBoolLike(const std::string &value, bool fallback) {
    std::string normalized;
    normalized.reserve(value.size());
    for (char ch : value) {
      if (!isspace(static_cast<unsigned char>(ch))) {
        normalized.push_back(static_cast<char>(tolower(static_cast<unsigned char>(ch))));
      }
    }
    if (normalized == "1" || normalized == "true" || normalized == "yes" || normalized == "on") {
      return true;
    }
    if (normalized == "0" || normalized == "false" || normalized == "no" || normalized == "off") {
      return false;
    }
    return fallback;
  }

  static bool JsonBoolField(const std::string &json, const char *field, bool fallback) {
    const std::string key = std::string("\"") + field + "\"";
    size_t pos = json.find(key);
    if (pos == std::string::npos) {
      return fallback;
    }
    pos = json.find(':', pos + key.size());
    if (pos == std::string::npos) {
      return fallback;
    }
    size_t start = pos + 1;
    while (start < json.size() && isspace(static_cast<unsigned char>(json[start]))) {
      ++start;
    }
    size_t end = start;
    while (end < json.size() && json[end] != ',' && json[end] != '}' && json[end] != '\n' && json[end] != '\r') {
      ++end;
    }
    return ParseBoolLike(json.substr(start, end - start), fallback);
  }

  bool ApplySkinPayload(const std::string &skin) {
    size_t first = skin.find('|');
    size_t second = first == std::string::npos ? std::string::npos : skin.find('|', first + 1);
    size_t third = second == std::string::npos ? std::string::npos : skin.find('|', second + 1);
    size_t fourth = third == std::string::npos ? std::string::npos : skin.find('|', third + 1);
    size_t fifth = fourth == std::string::npos ? std::string::npos : skin.find('|', fourth + 1);
    size_t sixth = fifth == std::string::npos ? std::string::npos : skin.find('|', fifth + 1);
    size_t seventh = sixth == std::string::npos ? std::string::npos : skin.find('|', sixth + 1);
    size_t eighth = seventh == std::string::npos ? std::string::npos : skin.find('|', seventh + 1);
    size_t ninth = eighth == std::string::npos ? std::string::npos : skin.find('|', eighth + 1);
    size_t tenth = ninth == std::string::npos ? std::string::npos : skin.find('|', ninth + 1);
    size_t eleventh = tenth == std::string::npos ? std::string::npos : skin.find('|', tenth + 1);
    if (first == std::string::npos || second == std::string::npos || third == std::string::npos ||
        fourth == std::string::npos || fifth == std::string::npos || sixth == std::string::npos ||
        seventh == std::string::npos || eighth == std::string::npos) {
      return false;
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
    theme_ = skin.substr(eighth + 1, ninth == std::string::npos ? std::string::npos : ninth - eighth - 1);
    if (ninth != std::string::npos) {
      pageSize_ = max(kMinCandidatesPerPage,
                      min(kMaxCandidatesPerPage, atoi(skin.substr(ninth + 1, tenth == std::string::npos ? std::string::npos : tenth - ninth - 1).c_str())));
    }
    if (tenth != std::string::npos) {
      layout_ = NormalizeCandidateLayout(skin.substr(tenth + 1, eleventh == std::string::npos ? std::string::npos : eleventh - tenth - 1));
    }
    if (eleventh != std::string::npos) {
      showComments_ = ParseBoolLike(skin.substr(eleventh + 1), true);
    }
    return true;
  }

  bool RefreshSkinFromConfigFile() {
    const std::wstring path = SkinConfigPath();
    if (path.empty()) {
      return false;
    }
    const DWORD now = GetTickCount();
    if (lastLocalSkinCheckTick_ != 0 && now - lastLocalSkinCheckTick_ < kSkinConfigPollMs) {
      return hasSkinConfigWriteTime_;
    }
    lastLocalSkinCheckTick_ = now;
    WIN32_FILE_ATTRIBUTE_DATA attrs{};
    if (!GetFileAttributesExW(path.c_str(), GetFileExInfoStandard, &attrs)) {
      return false;
    }
    if (hasSkinConfigWriteTime_ &&
        SameFileTime(lastSkinConfigWriteTime_, attrs.ftLastWriteTime)) {
      return true;
    }
    const std::string json = ReadUtf8File(path);
    if (json.empty()) {
      return false;
    }
    const std::string fontFamily = JsonStringField(json, "fontFamily");
    const int fontSize = JsonIntField(json, "fontSize");
    const std::string accent = JsonStringField(json, "accent");
    const std::string surface = JsonStringField(json, "surface");
    const std::string text = JsonStringField(json, "text");
    const std::string mutedText = JsonStringField(json, "mutedText");
    const std::string border = JsonStringField(json, "border");
    const std::string highlightText = JsonStringField(json, "highlightText");
    const std::string theme = JsonStringField(json, "theme");
    int pageSize = JsonIntField(json, "candidatePageSize");
    if (pageSize <= 0) {
      pageSize = kDefaultCandidatesPerPage;
    }
    const std::string layout = JsonStringField(json, "candidateLayout");
    const bool showComments = JsonBoolField(json, "showCandidateComments", true);
    if (fontFamily.empty() || fontSize <= 0 || accent.empty() || surface.empty() ||
        text.empty() || mutedText.empty() || border.empty() || highlightText.empty()) {
      return false;
    }
    std::string payload = fontFamily + "|" + std::to_string(fontSize) + "|" + accent + "|" +
                          surface + "|" + text + "|" + mutedText + "|" + border + "|" +
                          highlightText + "|" + theme + "|" + std::to_string(pageSize) + "|" +
                          NormalizeCandidateLayout(layout) + "|" +
                          (showComments ? "true" : "false");
    if (!ApplySkinPayload(payload)) {
      return false;
    }
    lastSkinConfigWriteTime_ = attrs.ftLastWriteTime;
    hasSkinConfigWriteTime_ = true;
    return true;
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
        const std::vector<std::string> fields = SplitTabFields(line);
        if (fields.size() >= 4) {
          CandidateView item;
          item.index = atoi(fields[0].c_str());
          item.text = Utf8ToWide(fields[1].c_str());
          item.reading = Utf8ToWide(fields[2].c_str());
          item.score = atoi(fields[3].c_str());
          if (fields.size() > 4) {
            item.kind = Utf8ToWide(fields[4].c_str());
          }
          if (fields.size() > 5) {
            item.source = Utf8ToWide(fields[5].c_str());
          }
          if (fields.size() > 6) {
            item.comment = Utf8ToWide(fields[6].c_str());
          }
          if (fields.size() > 7) {
            item.pinned = PayloadBoolField(fields[7]);
          }
          parsed.push_back(item);
        }
      }
      lineStart = lineEnd + 1;
    }
    return parsed;
  }

  std::wstring CompositionText() const {
    if (!candidates_.empty() && selectedIndex_ >= 0 &&
        selectedIndex_ < static_cast<int>(candidates_.size()) &&
        !candidates_[selectedIndex_].reading.empty()) {
      return candidates_[selectedIndex_].reading;
    }
    for (const CandidateView &candidate : candidates_) {
      if (!candidate.reading.empty()) {
        return candidate.reading;
      }
    }
    return L"";
  }

  std::wstring CandidateKindLabel(const std::wstring &kind) const {
    if (kind == L"emoji") {
      return L"表情";
    }
    if (kind == L"kaomoji") {
      return L"颜";
    }
    if (kind == L"symbol") {
      return L"符";
    }
    if (kind == L"phrase") {
      return L"短";
    }
    if (kind == L"agent") {
      return L"AI";
    }
    if (kind == L"dynamic") {
      return L"时";
    }
    return L"";
  }

  COLORREF CandidateBadgeFillColor(bool selected) const {
    if (selected) {
      return MixColor(accent_, highlightText_, ColorLuminance(accent_) < 120 ? 18 : 24);
    }
    return IsDarkSkin() ? MixColor(CandidateBandColor(), accent_, 20)
                        : MixColor(CandidateBandColor(), accent_, 8);
  }

  COLORREF CandidateBadgeBorderColor(bool selected) const {
    if (selected) {
      return MixColor(accent_, highlightText_, ColorLuminance(accent_) < 120 ? 24 : 32);
    }
    return IsDarkSkin() ? MixColor(border_, accent_, 36)
                        : MixColor(border_, accent_, 20);
  }

  COLORREF CandidateBadgeTextColor(bool selected) const {
    if (selected) {
      return highlightText_;
    }
    return IsDarkSkin() ? MixColor(mutedText_, accent_, 24) : mutedText_;
  }

  void DrawComposition(HDC dc, const RECT &rect) {
    if (composing_.empty()) {
      return;
    }
    SetTextColor(dc, PreeditTextColor());
    RECT composeRect{rect.left + Scale(16), rect.top + Scale(7), rect.right - Scale(16),
                     rect.top + ScaledFontSize() + Scale(16)};
    DrawTextW(dc, composing_.c_str(), static_cast<int>(composing_.size()), &composeRect,
              DT_SINGLELINE | DT_VCENTER | DT_LEFT | DT_END_ELLIPSIS);

    HPEN accentPen = CreatePen(PS_SOLID, max(1, Scale(2)), PreeditUnderlineColor());
    HGDIOBJ oldPen = SelectObject(dc, accentPen);
    const int underlineY = composeRect.bottom - Scale(2);
    MoveToEx(dc, composeRect.left, underlineY, nullptr);
    LineTo(dc, min(rect.right - Scale(16),
                   composeRect.left + max(Scale(28), TextWidth(dc, composing_))),
           underlineY);
    SelectObject(dc, oldPen);
    DeleteObject(accentPen);
  }

  void DrawRoundedRect(HDC dc, const RECT &rect, COLORREF fill, COLORREF stroke,
                       int radius) const {
    HBRUSH brush = CreateSolidBrush(fill);
    HPEN pen = CreatePen(PS_SOLID, max(1, Scale(1)), stroke);
    HGDIOBJ oldBrush = SelectObject(dc, brush);
    HGDIOBJ oldPen = SelectObject(dc, pen);
    const int scaledRadius = Scale(radius);
    RoundRect(dc, rect.left, rect.top, rect.right, rect.bottom, scaledRadius, scaledRadius);
    SelectObject(dc, oldPen);
    SelectObject(dc, oldBrush);
    DeleteObject(pen);
    DeleteObject(brush);
  }

  void DrawChevron(HDC dc, const RECT &rect, int delta, COLORREF color) const {
    const int midX = (rect.left + rect.right) / 2;
    const int midY = (rect.top + rect.bottom) / 2;
    const int halfWidth = Scale(4);
    const int halfHeight = Scale(6);
    POINT points[3]{};
    if (delta < 0) {
      points[0] = POINT{midX + halfWidth / 2, midY - halfHeight};
      points[1] = POINT{midX - halfWidth, midY};
      points[2] = POINT{midX + halfWidth / 2, midY + halfHeight};
    } else {
      points[0] = POINT{midX - halfWidth / 2, midY - halfHeight};
      points[1] = POINT{midX + halfWidth, midY};
      points[2] = POINT{midX - halfWidth / 2, midY + halfHeight};
    }
    HPEN pen = CreatePen(PS_SOLID, max(1, Scale(2)), color);
    HGDIOBJ oldPen = SelectObject(dc, pen);
    Polyline(dc, points, 3);
    SelectObject(dc, oldPen);
    DeleteObject(pen);
  }

  void DrawPageButton(HDC dc, const RECT &rect, int delta) const {
    DrawRoundedRect(dc, rect, CandidateIdleColor(), CandidateIdleBorderColor(), 10);
    DrawChevron(dc, rect, delta, mutedText_);
  }

  void DrawCandidateItem(HDC dc, const CandidateView &candidate, bool selected,
                         const RECT &itemRect) {
    if (selected) {
      RECT shadowRect{itemRect.left + Scale(1), itemRect.top + Scale(2),
                      itemRect.right + Scale(1), itemRect.bottom + Scale(2)};
      DrawRoundedRect(dc, shadowRect, MixColor(CandidateBandColor(), accent_, 18),
                      MixColor(CandidateBandColor(), accent_, 18), 12);
      DrawRoundedRect(dc, itemRect, accent_, CandidateAccentEdgeColor(), 12);
    } else {
      DrawRoundedRect(dc, itemRect, CandidateIdleColor(), CandidateIdleBorderColor(), 12);
    }

    wchar_t number[8]{};
    StringCchPrintfW(number, ARRAYSIZE(number), L"%d", candidate.index);
    SetTextColor(dc, selected ? highlightText_ : mutedText_);
    RECT numberRect{itemRect.left + Scale(10), itemRect.top, itemRect.left + Scale(30), itemRect.bottom};
    DrawTextW(dc, number, -1, &numberRect, DT_SINGLELINE | DT_VCENTER | DT_LEFT);

    SetTextColor(dc, selected ? highlightText_ : text_);
    RECT textRect{itemRect.left + Scale(30), itemRect.top, itemRect.right - Scale(10), itemRect.bottom};
    const std::wstring kindLabel = CandidateKindLabel(candidate.kind);
    int rightEdge = itemRect.right - Scale(8);
    if (!kindLabel.empty()) {
      const int badgeWidth = CandidateBadgeWidth(dc, kindLabel);
      rightEdge = itemRect.right - badgeWidth - Scale(8);
      textRect.right = max(textRect.left + Scale(24), rightEdge);
      RECT badgeRect{rightEdge, itemRect.top + Scale(7),
                     itemRect.right - Scale(7), itemRect.bottom - Scale(7)};
      DrawRoundedRect(dc, badgeRect, CandidateBadgeFillColor(selected),
                      CandidateBadgeBorderColor(selected), 9);
      SetTextColor(dc, CandidateBadgeTextColor(selected));
      DrawTextW(dc, kindLabel.c_str(), static_cast<int>(kindLabel.size()), &badgeRect,
                DT_SINGLELINE | DT_VCENTER | DT_CENTER | DT_END_ELLIPSIS);
      SetTextColor(dc, selected ? highlightText_ : text_);
    }
    if (showComments_ && !candidate.comment.empty()) {
      const int commentWidth = min(Scale(88), max(Scale(30), TextWidth(dc, candidate.comment) + Scale(8)));
      RECT commentRect{max(textRect.left + Scale(30), rightEdge - commentWidth), itemRect.top,
                       rightEdge - Scale(4), itemRect.bottom};
      textRect.right = max(textRect.left + Scale(24), commentRect.left - Scale(6));
      SetTextColor(dc, selected ? MixColor(highlightText_, accent_, 14) : mutedText_);
      DrawTextW(dc, candidate.comment.c_str(), static_cast<int>(candidate.comment.size()),
                &commentRect, DT_SINGLELINE | DT_VCENTER | DT_RIGHT | DT_END_ELLIPSIS);
      SetTextColor(dc, selected ? highlightText_ : text_);
    }
    DrawTextW(dc, candidate.text.c_str(), static_cast<int>(candidate.text.size()), &textRect,
              DT_SINGLELINE | DT_VCENTER | DT_LEFT | DT_END_ELLIPSIS);
  }

  void DrawCandidates(HDC dc, const RECT &rect) {
    statusText_.clear();
    candidateHits_.clear();
    pageHits_.clear();
    DrawComposition(dc, rect);
    int x = rect.left + Scale(15);
    const int y = CandidateBandTop() + Scale(8);
    const int itemHeight = max(Scale(34), ScaledFontSize() + Scale(20));
    const int candidateRight = HasPageControls() ? rect.right - PageControlsWidth() - Scale(8)
                                                 : rect.right - Scale(14);
    for (size_t i = 0; i < candidates_.size() && i < static_cast<size_t>(pageSize_); ++i) {
      const CandidateView &candidate = candidates_[i];
      const bool selected = static_cast<int>(i) == selectedIndex_;
      const int itemWidth = IsVerticalLayout()
                                ? candidateRight - x
                                : CandidateItemWidth(dc, candidate, selected);
      RECT itemRect{x, y, x + itemWidth, y + itemHeight};
      if (IsVerticalLayout()) {
        itemRect.top = y + static_cast<int>(i) * (itemHeight + Scale(7));
        itemRect.bottom = itemRect.top + itemHeight;
      }
      if (itemRect.left >= candidateRight - Scale(56)) {
        break;
      }
      if (itemRect.right > candidateRight) {
        itemRect.right = candidateRight;
      }
      candidateHits_.push_back(CandidateHit{itemRect, pageStart_ + static_cast<int>(i)});
      DrawCandidateItem(dc, candidate, selected, itemRect);
      if (IsVerticalLayout()) {
        continue;
      }
      x += itemWidth + Scale(7);
      if (x > candidateRight - Scale(56)) {
        break;
      }
    }
    DrawPageIndicator(dc, rect);
  }

  int HitTestCandidate(POINT point) const {
    for (const CandidateHit &hit : candidateHits_) {
      if (hit.absoluteIndex >= 0 && PtInRect(&hit.rect, point)) {
        return hit.absoluteIndex;
      }
    }
    return -1;
  }

  int HitTestPage(POINT point) const {
    for (const PageHit &hit : pageHits_) {
      if (hit.delta != 0 && PtInRect(&hit.rect, point)) {
        return hit.delta;
      }
    }
    return 0;
  }

  void ArmHoverGuard() {
    hoverGuardArmed_ = GetCursorPos(&hoverGuardScreenPoint_) != FALSE;
  }

  bool ShouldIgnoreHoverMove(POINT clientPoint) {
    if (!hoverGuardArmed_ || !hwnd_) {
      return false;
    }
    POINT screenPoint = clientPoint;
    ClientToScreen(hwnd_, &screenPoint);
    const int dx = abs(screenPoint.x - hoverGuardScreenPoint_.x);
    const int dy = abs(screenPoint.y - hoverGuardScreenPoint_.y);
    if (dx <= 2 && dy <= 2) {
      return true;
    }
    hoverGuardArmed_ = false;
    return false;
  }

  void SelectCandidate(int absoluteIndex) {
    const int relativeIndex = absoluteIndex - pageStart_;
    if (relativeIndex < 0 || relativeIndex >= static_cast<int>(candidates_.size()) ||
        relativeIndex == selectedIndex_) {
      return;
    }
    selectedIndex_ = relativeIndex;
    composing_ = CompositionText();
    if (selectHandler_) {
      selectHandler_(selectOwner_, absoluteIndex);
    }
    if (hwnd_) {
      InvalidateRect(hwnd_, nullptr, FALSE);
    }
  }

  void DrawPageIndicator(HDC dc, const RECT &rect) {
    if (!HasPageControls()) {
      return;
    }
    const int first = pageStart_ + 1;
    const int last = min(pageStart_ + static_cast<int>(candidates_.size()), totalCount_);
    wchar_t label[32]{};
    StringCchPrintfW(label, ARRAYSIZE(label), L"%d-%d/%d", first, last, totalCount_);
    const int centerY = IsVerticalLayout()
                            ? CandidateBandTop() + max(Scale(34), (rect.bottom - CandidateBandTop()) / 2)
                            : CandidateBandTop() + Scale(8) +
                                  max(Scale(34), ScaledFontSize() + Scale(20)) / 2;
    RECT prevRect{rect.right - Scale(156), centerY - Scale(15),
                  rect.right - Scale(128), centerY + Scale(15)};
    RECT nextRect{rect.right - Scale(42), centerY - Scale(15),
                  rect.right - Scale(14), centerY + Scale(15)};
    DrawPageButton(dc, prevRect, -1);
    DrawPageButton(dc, nextRect, 1);
    pageHits_.push_back(PageHit{prevRect, -1});
    pageHits_.push_back(PageHit{nextRect, 1});
    SetTextColor(dc, mutedText_);
    RECT labelRect{prevRect.right + Scale(6), centerY - Scale(15),
                   nextRect.left - Scale(6), centerY + Scale(15)};
    DrawTextW(dc, label, -1, &labelRect, DT_SINGLELINE | DT_VCENTER | DT_CENTER);
  }

  void DrawStatus(HDC dc, const RECT &rect) {
    RECT badge{rect.left + Scale(10), rect.top + Scale(7),
               rect.right - Scale(10), rect.bottom - Scale(7)};
    HBRUSH selected = CreateSolidBrush(accent_);
    HPEN selectedPen = CreatePen(PS_SOLID, max(1, Scale(1)), accent_);
    HGDIOBJ oldBrush = SelectObject(dc, selected);
    HGDIOBJ oldPen = SelectObject(dc, selectedPen);
    const int badgeRadius = Scale(10);
    RoundRect(dc, badge.left, badge.top, badge.right, badge.bottom, badgeRadius, badgeRadius);
    SelectObject(dc, oldPen);
    SelectObject(dc, oldBrush);
    DeleteObject(selectedPen);
    DeleteObject(selected);

    SetTextColor(dc, highlightText_);
    DrawTextW(dc, statusText_.c_str(), static_cast<int>(statusText_.size()), &badge,
              DT_SINGLELINE | DT_VCENTER | DT_CENTER);
  }

  void RefreshSkin() {
    if (RefreshSkinFromConfigFile()) {
      return;
    }
    const DWORD now = GetTickCount();
    if (lastHttpSkinRefreshTick_ != 0 && now - lastHttpSkinRefreshTick_ < kHttpSkinPollMs) {
      return;
    }
    lastHttpSkinRefreshTick_ = now;
    const std::string skin = HttpRequest(L"GET", L"/ime/skin");
    if (skin.empty()) {
      return;
    }
    ApplySkinPayload(skin);
  }
};

bool IsAsciiLetter(WPARAM key) {
  return (key >= L'A' && key <= L'Z') || (key >= L'a' && key <= L'z');
}

bool IsShiftKey(WPARAM key) {
  return key == VK_SHIFT || key == VK_LSHIFT || key == VK_RSHIFT;
}

bool IsShiftPressed() {
  return (GetKeyState(VK_SHIFT) & 0x8000) != 0;
}

bool IsControlPressed() {
  return (GetKeyState(VK_CONTROL) & 0x8000) != 0;
}

bool IsAltPressed() {
  return (GetKeyState(VK_MENU) & 0x8000) != 0;
}

bool HasSystemModifier() {
  return IsControlPressed() || IsAltPressed();
}

bool IsChinesePunctuationKey(WPARAM key) {
  const bool shifted = IsShiftPressed();
  switch (key) {
    case VK_OEM_COMMA:
    case L',':
    case VK_OEM_PERIOD:
    case L'.':
    case VK_OEM_1:
    case L';':
    case VK_OEM_2:
    case L'/':
    case VK_OEM_4:
    case L'[':
    case VK_OEM_6:
    case L']':
    case VK_OEM_7:
    case L'\'':
    case VK_OEM_MINUS:
    case L'-':
      return true;
    case L'<':
    case L'>':
    case L':':
    case L'?':
    case L'{':
    case L'}':
    case L'"':
      return true;
    case L'1':
    case L'6':
    case L'9':
    case L'0':
      return shifted;
    default:
      return false;
  }
}

char RecognizerAsciiCharForKey(WPARAM key) {
  const bool shifted = IsShiftPressed();
  if (!shifted && key >= L'A' && key <= L'Z') {
    return static_cast<char>(key - L'A' + 'a');
  }
  if (!shifted && key >= L'a' && key <= L'z') {
    return static_cast<char>(key);
  }
  if (!shifted && key >= L'0' && key <= L'9') {
    return static_cast<char>(key);
  }
  if (key >= VK_NUMPAD0 && key <= VK_NUMPAD9) {
    return static_cast<char>('0' + (key - VK_NUMPAD0));
  }
  if (!shifted) {
    switch (key) {
      case VK_OEM_PERIOD:
      case L'.':
        return '.';
      case VK_OEM_2:
      case L'/':
        return '/';
      case VK_OEM_MINUS:
      case L'-':
        return '-';
      case VK_OEM_PLUS:
      case L'=':
        return '=';
      case VK_OEM_1:
      case L';':
        return ';';
      case VK_OEM_7:
      case L'\'':
        return '\'';
      case VK_OEM_COMMA:
      case L',':
        return ',';
      default:
        return 0;
    }
  }
  switch (key) {
    case L'2':
      return '@';
    case L'5':
      return '%';
    case L'6':
      return '^';
    case L'7':
      return '&';
    case VK_OEM_PLUS:
    case L'=':
      return '+';
    case VK_OEM_1:
    case L';':
      return ':';
    case VK_OEM_2:
    case L'/':
      return '?';
    case VK_OEM_MINUS:
    case L'-':
      return '_';
    default:
      return 0;
  }
}

bool IsRecognizerContinuingChar(char ch) {
  if (ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9') {
    return true;
  }
  switch (ch) {
    case ';':
    case '\'':
    case '/':
    case '`':
    case '@':
    case '.':
    case '-':
    case '_':
    case ':':
    case '?':
    case '&':
    case '=':
    case '%':
    case '+':
      return true;
    default:
      return false;
  }
}

int CandidateIndexFromKey(WPARAM key) {
  if (key >= L'1' && key <= L'9') {
    return static_cast<int>(key - L'1');
  }
  if (key >= VK_NUMPAD1 && key <= VK_NUMPAD9) {
    return static_cast<int>(key - VK_NUMPAD1);
  }
  return -1;
}

bool IsCandidateNumberKey(WPARAM key) {
  return CandidateIndexFromKey(key) >= 0;
}

int CandidateQuickSelectIndexFromKey(WPARAM key) {
  if (IsShiftPressed()) {
    return -1;
  }
  if (key == VK_OEM_1 || key == L';') {
    return 1;
  }
  if (key == VK_OEM_7 || key == L'\'') {
    return 2;
  }
  return -1;
}

bool IsCommaPeriodPageKey(WPARAM key) {
  return !IsShiftPressed() && (key == VK_OEM_COMMA || key == L',' || key == VK_OEM_PERIOD || key == L'.');
}

bool IsPageKey(WPARAM key) {
  if (!IsShiftPressed() && (key == VK_OEM_4 || key == L'[' || key == VK_OEM_6 || key == L']')) {
    return true;
  }
  return key == VK_NEXT || key == VK_PRIOR || key == VK_OEM_MINUS || key == VK_OEM_PLUS ||
         key == L'-' || key == L'=';
}

bool IsBracketPageKey(WPARAM key) {
  return !IsShiftPressed() && (key == VK_OEM_4 || key == L'[' || key == VK_OEM_6 || key == L']');
}

int CandidatePageDeltaForKey(WPARAM key) {
  if (!IsShiftPressed() && (key == VK_OEM_4 || key == L'[')) {
    return -1;
  }
  if (!IsShiftPressed() && (key == VK_OEM_6 || key == L']')) {
    return 1;
  }
  if (key == VK_PRIOR || key == VK_OEM_MINUS || key == L'-') {
    return -1;
  }
  if (key == VK_NEXT || key == VK_OEM_PLUS || key == L'=') {
    return 1;
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
    candidateWindow_.SetClickHandler(this, &TextService::OnCandidateClickedThunk);
    candidateWindow_.SetSelectHandler(this, &TextService::OnCandidateSelectedThunk);
    candidateWindow_.SetPageHandler(this, &TextService::OnCandidatePageThunk);
    candidateWindow_.SetMenuHandler(this, &TextService::OnCandidateMenuThunk);
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
    LogDebugLine(L"ActivateEx called");
    if (!threadMgr || !EnsureCoreLoaded()) {
      LogLine(L"ActivateEx failed: threadMgr/core unavailable");
      return E_FAIL;
    }
    threadMgr_ = threadMgr;
    threadMgr_->AddRef();
    clientId_ = clientId;
    session_ = g_core.createSession();
    RefreshAsciiModeFromCore();
    RefreshTypingConfigFromConfig();

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
    LogDebugLine(L"Deactivate called");
    candidateWindow_.Hide();
    cachedCandidateCount_ = 0;
    compositionLength_ = 0;
    ResetPunctuationState();
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
    ForgetLastContext();
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
    if (HasSystemModifier()) {
      *eaten = FALSE;
      return S_OK;
    }
    if (shiftDown_ && !IsShiftKey(key)) {
      shiftToggleCandidate_ = false;
    }
    RefreshTypingConfigFromConfig();
    *eaten = ShouldEatKey(key);
    return S_OK;
  }

  STDMETHODIMP OnKeyDown(ITfContext *context, WPARAM key, LPARAM, BOOL *eaten) override {
    if (!eaten) {
      return E_INVALIDARG;
    }
    *eaten = FALSE;
    if (HasSystemModifier()) {
      return S_OK;
    }
    RefreshTypingConfigFromConfig();
    if (!session_ || !ShouldEatKey(key)) {
      return S_OK;
    }
    RememberContext(context);

    if (shiftToggleMode_ && IsShiftKey(key)) {
      if (!shiftDown_) {
        shiftDown_ = true;
        shiftToggleCandidate_ = rawBuffer_.empty() && cachedCandidateCount_ == 0;
      }
      *eaten = TRUE;
      return S_OK;
    }

    if (shiftDown_) {
      shiftToggleCandidate_ = false;
    }

    if (IsAsciiLetter(key)) {
      char ch = static_cast<char>(key);
      if (ch >= 'A' && ch <= 'Z') {
        ch = static_cast<char>(ch - 'A' + 'a');
      }
      selectedIndex_ = 0;
      pageOffset_ = 0;
      compositionLength_++;
      rawBuffer_.push_back(ch);
      const int count = g_core.inputKeyFast(session_, ch);
      UpdateCandidateWindow(count);
      *eaten = TRUE;
      return S_OK;
    }

    if (IsMicrosoftDoublePinyinSemicolonKey(key)) {
      selectedIndex_ = 0;
      pageOffset_ = 0;
      compositionLength_++;
      rawBuffer_.push_back(';');
      const int count = g_core.inputKeyFast(session_, ';');
      UpdateCandidateWindow(count);
      *eaten = TRUE;
      return S_OK;
    }

    const char recognizerChar = RecognizerAsciiCharForKey(key);
    if (!asciiMode_ && ShouldUseRecognizerLiteralChar(recognizerChar)) {
      InputRecognizerLiteralChar(recognizerChar);
      *eaten = TRUE;
      return S_OK;
    }

    if (key == VK_BACK) {
      selectedIndex_ = 0;
      pageOffset_ = 0;
      const int count = g_core.backspaceFast(session_);
      if (!rawBuffer_.empty()) {
        rawBuffer_.pop_back();
      }
      compositionLength_ = static_cast<int>(rawBuffer_.size());
      UpdateCandidateWindow(count);
      *eaten = TRUE;
      return S_OK;
    }

    if (key == VK_ESCAPE) {
      selectedIndex_ = 0;
      ClearSession();
      candidateWindow_.Hide();
      cachedCandidateCount_ = 0;
      compositionLength_ = 0;
      *eaten = TRUE;
      return S_OK;
    }

    if (key == VK_RIGHT || key == VK_DOWN || key == VK_TAB) {
      MoveSelection(key == VK_TAB && IsShiftPressed() ? -1 : 1);
      *eaten = TRUE;
      return S_OK;
    }

    if (key == VK_LEFT || key == VK_UP) {
      MoveSelection(-1);
      *eaten = TRUE;
      return S_OK;
    }

    if (key == VK_HOME || key == VK_END) {
      MoveSelectionTo(key == VK_HOME ? 0 : cachedCandidateCount_ - 1);
      *eaten = TRUE;
      return S_OK;
    }

    if (IsConfiguredPageKey(key) && cachedCandidateCount_ > CandidatePageSize()) {
      MovePage(ConfiguredPageDeltaForKey(key));
      *eaten = TRUE;
      return S_OK;
    }

    const int quickIndex = ConfiguredQuickSelectIndexFromKey(key);
    if (quickIndex >= 0 && cachedCandidateCount_ > pageOffset_ + quickIndex) {
      CommitCandidate(context, pageOffset_ + quickIndex);
      selectedIndex_ = 0;
      pageOffset_ = 0;
      candidateWindow_.Hide();
      cachedCandidateCount_ = 0;
      compositionLength_ = 0;
      *eaten = TRUE;
      return S_OK;
    }

    const std::wstring punctuation = ChinesePunctuationForKey(key);
    if (!asciiMode_ && !punctuation.empty()) {
      if (!rawBuffer_.empty() && IsRecognizerLiteralCurrent()) {
        std::wstring suffix;
        if (recognizerChar != 0) {
          char ascii[2]{recognizerChar, 0};
          suffix = Utf8ToWide(ascii);
        } else {
          suffix = punctuation;
        }
        CommitRawBuffer(context, suffix);
      } else if (cachedCandidateCount_ > 0) {
        CommitCandidate(context, selectedIndex_);
        selectedIndex_ = 0;
        pageOffset_ = 0;
        candidateWindow_.Hide();
        cachedCandidateCount_ = 0;
        CommitText(context, punctuation);
      } else if (!rawBuffer_.empty()) {
        CommitRawBuffer(context, punctuation);
      } else {
        CommitText(context, punctuation);
      }
      *eaten = TRUE;
      return S_OK;
    }

    if ((key == VK_SPACE || key == VK_RETURN) && cachedCandidateCount_ <= 0 && !rawBuffer_.empty()) {
      CommitRawBuffer(context, key == VK_SPACE ? L" " : L"");
      candidateWindow_.Hide();
      *eaten = TRUE;
      return S_OK;
    }

    if (key == VK_SPACE || key == VK_RETURN || IsCandidateNumberKey(key)) {
      const int index =
          IsCandidateNumberKey(key) ? pageOffset_ + CandidateIndexFromKey(key) : selectedIndex_;
      CommitCandidate(context, index);
      selectedIndex_ = 0;
      pageOffset_ = 0;
      candidateWindow_.Hide();
      cachedCandidateCount_ = 0;
      compositionLength_ = 0;
      *eaten = TRUE;
      return S_OK;
    }

    return S_OK;
  }

  STDMETHODIMP OnTestKeyUp(ITfContext *, WPARAM key, LPARAM, BOOL *eaten) override {
    if (!eaten) {
      return E_INVALIDARG;
    }
    *eaten = shiftToggleMode_ && IsShiftKey(key) && shiftDown_ && shiftToggleCandidate_ && !HasSystemModifier();
    return S_OK;
  }

  STDMETHODIMP OnKeyUp(ITfContext *, WPARAM key, LPARAM, BOOL *eaten) override {
    if (!eaten) {
      return E_INVALIDARG;
    }
    if (shiftToggleMode_ && IsShiftKey(key)) {
      const bool shouldToggle = shiftDown_ && shiftToggleCandidate_ && !HasSystemModifier();
      if (shouldToggle) {
        ToggleAsciiMode();
      }
      shiftDown_ = false;
      shiftToggleCandidate_ = false;
      *eaten = shouldToggle ? TRUE : FALSE;
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
  static void OnCandidateClickedThunk(void *owner, int absoluteIndex) {
    if (!owner) {
      return;
    }
    static_cast<TextService *>(owner)->OnCandidateClicked(absoluteIndex);
  }

  static void OnCandidateSelectedThunk(void *owner, int absoluteIndex) {
    if (!owner) {
      return;
    }
    static_cast<TextService *>(owner)->OnCandidateSelected(absoluteIndex);
  }

  static void OnCandidatePageThunk(void *owner, int delta) {
    if (!owner) {
      return;
    }
    static_cast<TextService *>(owner)->OnCandidatePage(delta);
  }

  static void OnCandidateMenuThunk(void *owner, int absoluteIndex, HWND menuHwnd, POINT screenPoint) {
    if (!owner) {
      return;
    }
    static_cast<TextService *>(owner)->OnCandidateMenu(absoluteIndex, menuHwnd, screenPoint);
  }

  void OnCandidateClicked(int absoluteIndex) {
    if (!lastContext_ || !session_ || absoluteIndex < 0 ||
        absoluteIndex >= cachedCandidateCount_) {
      return;
    }
    CommitCandidate(lastContext_, absoluteIndex);
    selectedIndex_ = 0;
    pageOffset_ = 0;
    candidateWindow_.Hide();
    cachedCandidateCount_ = 0;
    compositionLength_ = 0;
  }

  void OnCandidateSelected(int absoluteIndex) {
    if (!session_ || absoluteIndex < 0 || absoluteIndex >= cachedCandidateCount_) {
      return;
    }
    selectedIndex_ = absoluteIndex;
    pageOffset_ = (selectedIndex_ / CandidatePageSize()) * CandidatePageSize();
  }

  void OnCandidatePage(int delta) {
    if (!session_ || cachedCandidateCount_ <= CandidatePageSize() || delta == 0) {
      return;
    }
    MovePage(delta);
  }

  void OnCandidateMenu(int absoluteIndex, HWND menuHwnd, POINT screenPoint) {
    if (!session_ || !lastContext_ || absoluteIndex < 0 ||
        absoluteIndex >= cachedCandidateCount_) {
      return;
    }
    selectedIndex_ = absoluteIndex;
    pageOffset_ = (selectedIndex_ / CandidatePageSize()) * CandidatePageSize();

    HMENU menu = CreatePopupMenu();
    if (!menu) {
      return;
    }

    constexpr UINT_PTR kCommit = 1001;
    constexpr UINT_PTR kPin = 1002;
    constexpr UINT_PTR kForget = 1003;
    constexpr UINT_PTR kFirstChar = 1004;
    constexpr UINT_PTR kLastChar = 1005;

    AppendMenuW(menu, MF_STRING, kCommit, L"上屏");
    AppendMenuW(menu, MF_SEPARATOR, 0, nullptr);
    const bool canPin = g_core.pinCandidate || g_core.candidateAction || g_core.executeCommand;
    AppendMenuW(menu, MF_STRING | (canPin ? 0 : MF_GRAYED), kPin, L"固定到前排");
    const bool canForget = g_core.rejectCandidate || g_core.candidateAction || g_core.executeCommand;
    AppendMenuW(menu, MF_STRING | (canForget ? 0 : MF_GRAYED), kForget, L"隐藏候选");
    if (g_core.commitCandidateChar) {
      AppendMenuW(menu, MF_SEPARATOR, 0, nullptr);
      AppendMenuW(menu, MF_STRING, kFirstChar, L"首字上屏");
      AppendMenuW(menu, MF_STRING, kLastChar, L"末字上屏");
    }

    SetForegroundWindow(menuHwnd);
    const UINT command = TrackPopupMenu(menu,
                                        TPM_RETURNCMD | TPM_RIGHTBUTTON | TPM_NONOTIFY,
                                        screenPoint.x, screenPoint.y, 0, menuHwnd, nullptr);
    DestroyMenu(menu);
    PostMessageW(menuHwnd, WM_NULL, 0, 0);

    switch (command) {
      case kCommit:
        CommitCandidate(lastContext_, absoluteIndex);
        ResetCompositionState();
        candidateWindow_.Hide();
        break;
      case kPin:
        if (RunCandidateAction("pin", absoluteIndex)) {
          UpdateCandidateWindow();
        }
        break;
      case kForget:
        if (ForgetCandidate(absoluteIndex)) {
          if (selectedIndex_ >= cachedCandidateCount_) {
            selectedIndex_ = max(0, cachedCandidateCount_ - 1);
          }
          UpdateCandidateWindow();
        }
        break;
      case kFirstChar:
        CommitCandidateChar(lastContext_, absoluteIndex, "first");
        ResetCompositionState();
        candidateWindow_.Hide();
        break;
      case kLastChar:
        CommitCandidateChar(lastContext_, absoluteIndex, "last");
        ResetCompositionState();
        candidateWindow_.Hide();
        break;
      default:
        break;
    }
  }

  void RememberContext(ITfContext *context) {
    if (!context || context == lastContext_) {
      return;
    }
    context->AddRef();
    ForgetLastContext();
    lastContext_ = context;
  }

  void ForgetLastContext() {
    if (lastContext_) {
      lastContext_->Release();
      lastContext_ = nullptr;
    }
  }

  bool IsMicrosoftDoublePinyinSemicolonKey(WPARAM key) const {
    return !asciiMode_ && microsoftDoublePinyin_ && (key == VK_OEM_1 || key == L';');
  }

  bool RecognizerLiteralDecision(const std::string &input) const {
    if (!session_ || !g_core.executeCommand || input.empty()) {
      return false;
    }
    const std::string payload = std::string("{\"input\":\"") + JsonEscape(input) + "\"}";
    char *raw = g_core.executeCommand(session_, "recognizer-decision-json", payload.c_str());
    if (!raw) {
      return false;
    }
    const std::string json(raw);
    g_core.freeValue(raw);
    return JsonBoolFieldValue(json, "matched", false) &&
           JsonBoolFieldValue(json, "passThrough", false);
  }

  bool IsRecognizerLiteralCurrent() const {
    return RecognizerLiteralDecision(rawBuffer_);
  }

  bool IsRecognizerLiteralProspective(char ch) const {
    if (rawBuffer_.empty() || ch == 0) {
      return false;
    }
    std::string next = rawBuffer_;
    next.push_back(ch);
    return RecognizerLiteralDecision(next);
  }

  bool ShouldUseRecognizerLiteralChar(char ch) const {
    if (ch == 0 || rawBuffer_.empty() || !IsRecognizerContinuingChar(ch)) {
      return false;
    }
    if (IsRecognizerLiteralCurrent() && IsRecognizerContinuingChar(ch)) {
      return true;
    }
    return IsRecognizerLiteralProspective(ch);
  }

  void InputRecognizerLiteralChar(char ch) {
    selectedIndex_ = 0;
    pageOffset_ = 0;
    compositionLength_++;
    rawBuffer_.push_back(ch);
    const int count = g_core.inputKeyFast(session_, ch);
    UpdateCandidateWindow(count);
  }

  int ConfiguredQuickSelectIndexFromKey(WPARAM key) const {
    if (IsShiftPressed()) {
      return -1;
    }
    if (semicolonQuickSelect_ && !IsMicrosoftDoublePinyinSemicolonKey(key) &&
        (key == VK_OEM_1 || key == L';')) {
      return 1;
    }
    if (quoteQuickSelect_ && (key == VK_OEM_7 || key == L'\'')) {
      return 2;
    }
    return -1;
  }

  bool IsConfiguredBracketPageKey(WPARAM key) const {
    return bracketPageKeys_ && IsBracketPageKey(key);
  }

  bool IsConfiguredPageKey(WPARAM key) const {
    if (IsConfiguredBracketPageKey(key)) {
      return true;
    }
    if (minusEqualPageKeys_ &&
        (key == VK_NEXT || key == VK_PRIOR || key == VK_OEM_MINUS ||
         key == VK_OEM_PLUS || key == L'-' || key == L'=')) {
      return true;
    }
    return commaPeriodPageKeys_ && IsCommaPeriodPageKey(key);
  }

  int ConfiguredPageDeltaForKey(WPARAM key) const {
    if (IsConfiguredBracketPageKey(key)) {
      return CandidatePageDeltaForKey(key);
    }
    if (minusEqualPageKeys_ && (key == VK_PRIOR || key == VK_OEM_MINUS || key == L'-')) {
      return -1;
    }
    if (minusEqualPageKeys_ && (key == VK_NEXT || key == VK_OEM_PLUS || key == L'=')) {
      return 1;
    }
    if (commaPeriodPageKeys_ && !IsShiftPressed() && (key == VK_OEM_COMMA || key == L',')) {
      return -1;
    }
    if (commaPeriodPageKeys_ && !IsShiftPressed() && (key == VK_OEM_PERIOD || key == L'.')) {
      return 1;
    }
    return 0;
  }

  bool ShouldEatKey(WPARAM key) const {
    if (!session_) {
      return false;
    }
    if (HasSystemModifier()) {
      return false;
    }
    if (shiftToggleMode_ && IsShiftKey(key)) {
      return true;
    }
    if (asciiMode_) {
      return cachedCandidateCount_ > 0 && (key == VK_ESCAPE || key == VK_SPACE || key == VK_RETURN);
    }
    if (IsAsciiLetter(key)) {
      return true;
    }
    if (IsMicrosoftDoublePinyinSemicolonKey(key)) {
      return true;
    }
    if (!rawBuffer_.empty()) {
      const char recognizerChar = RecognizerAsciiCharForKey(key);
      if (ShouldUseRecognizerLiteralChar(recognizerChar)) {
        return true;
      }
      if (recognizerChar != 0 && IsChinesePunctuationKey(key) && IsRecognizerLiteralCurrent()) {
        return true;
      }
    }
    if (key == VK_BACK) {
      return compositionLength_ > 0;
    }
    if (key == VK_ESCAPE) {
      return cachedCandidateCount_ > 0 || compositionLength_ > 0;
    }
    if (key == VK_RIGHT || key == VK_DOWN || key == VK_TAB || key == VK_LEFT || key == VK_UP) {
      return cachedCandidateCount_ > 0;
    }
    if (key == VK_HOME || key == VK_END) {
      return cachedCandidateCount_ > 0;
    }
    if (IsConfiguredPageKey(key) && (!IsConfiguredBracketPageKey(key) || cachedCandidateCount_ > CandidatePageSize())) {
      return cachedCandidateCount_ > CandidatePageSize();
    }
    if (key == VK_SPACE || key == VK_RETURN) {
      return cachedCandidateCount_ > 0 || !rawBuffer_.empty();
    }
    const int quickIndex = ConfiguredQuickSelectIndexFromKey(key);
    if (quickIndex >= 0) {
      return cachedCandidateCount_ > pageOffset_ + quickIndex;
    }
    if (!asciiMode_ && IsChinesePunctuationKey(key)) {
      return punctuationFullWidth_ || cachedCandidateCount_ > 0 || !rawBuffer_.empty();
    }
    if (IsCandidateNumberKey(key)) {
      const int relativeIndex = CandidateIndexFromKey(key);
      return relativeIndex < CandidatePageSize() &&
             cachedCandidateCount_ > pageOffset_ + relativeIndex;
    }
    return false;
  }

  void CommitText(ITfContext *context, const std::wstring &text) {
    if (!context || text.empty()) {
      LogLine(L"CommitText skipped: no context/text");
      return;
    }

    EditSession *session = new EditSession(context, text);
    HRESULT editResult = E_FAIL;
    HRESULT hr = context->RequestEditSession(
        clientId_, session, TF_ES_READWRITE | TF_ES_SYNC, &editResult);
    session->Release();
    wchar_t message[192]{};
    StringCchPrintfW(message, ARRAYSIZE(message),
                     L"CommitText text_len=%zu request_hr=0x%08X edit_hr=0x%08X",
                     text.size(), static_cast<unsigned int>(hr), static_cast<unsigned int>(editResult));
    LogDebugLine(message);
    if (hr == TF_E_SYNCHRONOUS || FAILED(hr)) {
      EditSession *asyncSession = new EditSession(context, text);
      HRESULT ignored = E_FAIL;
      context->RequestEditSession(clientId_, asyncSession, TF_ES_READWRITE | TF_ES_ASYNC, &ignored);
      asyncSession->Release();
    }
  }

  void CommitRawBuffer(ITfContext *context, const std::wstring &suffix = L"") {
    if (rawBuffer_.empty()) {
      return;
    }
    std::wstring text = Utf8ToWide(rawBuffer_.c_str()) + suffix;
    ClearSession();
    candidateWindow_.Hide();
    cachedCandidateCount_ = 0;
    CommitText(context, text);
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

    compositionLength_ = 0;
    rawBuffer_.clear();
    CommitText(context, text);
  }

  void CommitCandidateChar(ITfContext *context, int index, const char *side) {
    if (!context || !session_ || !g_core.commitCandidateChar || !side) {
      return;
    }
    char *committed = g_core.commitCandidateChar(session_, index, side);
    std::wstring text = Utf8ToWide(committed);
    if (committed) {
      g_core.freeValue(committed);
    }
    if (text.empty()) {
      return;
    }
    compositionLength_ = 0;
    rawBuffer_.clear();
    CommitText(context, text);
  }

  bool RunCandidateAction(const char *action, int index) {
    if (!session_ || !action) {
      return false;
    }
    if (_stricmp(action, "pin") == 0 && g_core.pinCandidate) {
      char *result = g_core.pinCandidate(session_, index);
      if (!result) {
        return false;
      }
      const std::string json(result);
      g_core.freeValue(result);
      return json.find("\"error\"") == std::string::npos;
    }
    char payload[96]{};
    sprintf_s(payload, "{\"action\":\"%s\",\"index\":%d}", action, index);
    char *result = nullptr;
    if (g_core.candidateAction) {
      result = g_core.candidateAction(session_, payload);
    } else if (g_core.executeCommand) {
      result = g_core.executeCommand(session_, "candidate-action", payload);
    }
    if (!result) {
      return false;
    }
    const std::string json(result);
    g_core.freeValue(result);
    return json.find("\"error\"") == std::string::npos;
  }

  bool ForgetCandidate(int index) {
    if (!session_) {
      return false;
    }
    if (g_core.rejectCandidate) {
      char *result = g_core.rejectCandidate(session_, index);
      if (!result) {
        return false;
      }
      const std::string json(result);
      g_core.freeValue(result);
      return json.find("\"error\"") == std::string::npos;
    }
    return RunCandidateAction("forget", index);
  }

  std::string BuildCandidatePayloadFromCore(int count) {
    const int pageCount = min(CandidatePageSize(), max(0, count - pageOffset_));
    if (g_core.candidatePayloadRange) {
      char *payload = g_core.candidatePayloadRange(session_, pageOffset_, pageCount);
      std::string result = payload ? payload : "";
      if (payload) {
        g_core.freeValue(payload);
      }
      if (!result.empty()) {
        return result;
      }
    }
    if (g_core.candidatePayload && pageOffset_ == 0) {
      char *payload = g_core.candidatePayload(session_, pageCount);
      std::string result = payload ? payload : "";
      if (payload) {
        g_core.freeValue(payload);
      }
      if (!result.empty()) {
        return result;
      }
    }

    std::string payload;
    for (int i = 0; i < pageCount; ++i) {
      const int absoluteIndex = pageOffset_ + i;
      char *text = g_core.candidateText(session_, absoluteIndex);
      char *reading = g_core.candidateReading(session_, absoluteIndex);
      const int score = g_core.candidateScore(session_, absoluteIndex);

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
      pageOffset_ = 0;
      candidateWindow_.Hide();
      return;
    }
    if (count > 0 && selectedIndex_ >= count) {
      selectedIndex_ = count - 1;
    }
    if (selectedIndex_ < 0) {
      selectedIndex_ = 0;
    }
    pageOffset_ = (selectedIndex_ / CandidatePageSize()) * CandidatePageSize();
    std::string candidates;
    if (g_core.inProcess) {
      candidates = BuildCandidatePayloadFromCore(count);
    } else {
      const int pageCount = min(CandidatePageSize(), max(0, count - pageOffset_));
      wchar_t path[120]{};
      StringCchPrintfW(path, ARRAYSIZE(path),
                       L"/ime/candidates?session=%llu&start=%d&limit=%d",
                       session_, pageOffset_, pageCount);
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
    candidateWindow_.Show(candidates, selectedIndex_ - pageOffset_, pageOffset_, count,
                          CandidatePageSize(),
                          Utf8ToWide(rawBuffer_.c_str()));
  }

  void MoveSelection(int delta) {
    const int count = cachedCandidateCount_;
    if (count <= 0) {
      return;
    }
    selectedIndex_ = (selectedIndex_ + delta + count) % count;
    UpdateCandidateWindow();
  }

  void MoveSelectionTo(int index) {
    const int count = cachedCandidateCount_;
    if (count <= 0) {
      return;
    }
    selectedIndex_ = max(0, min(index, count - 1));
    UpdateCandidateWindow();
  }

  void MovePage(int delta) {
    const int count = cachedCandidateCount_;
    if (count <= CandidatePageSize()) {
      return;
    }
    const int lastPageOffset = ((count - 1) / CandidatePageSize()) * CandidatePageSize();
    int nextOffset = pageOffset_ + delta * CandidatePageSize();
    if (nextOffset > lastPageOffset) {
      nextOffset = 0;
    } else if (nextOffset < 0) {
      nextOffset = lastPageOffset;
    }
    pageOffset_ = nextOffset;
    selectedIndex_ = min(pageOffset_, count - 1);
    UpdateCandidateWindow();
  }

  int CandidatePageSize() const {
    return max(kMinCandidatesPerPage, min(kMaxCandidatesPerPage, candidatePageSize_));
  }

  void ClearSession() {
    if (!session_ || !g_core.clearSession) {
      return;
    }
    char *cleared = g_core.clearSession(session_);
    if (cleared) {
      g_core.freeValue(cleared);
    }
    ResetCompositionState();
  }

  void ResetCompositionState() {
    cachedCandidateCount_ = 0;
    selectedIndex_ = 0;
    pageOffset_ = 0;
    compositionLength_ = 0;
    rawBuffer_.clear();
  }

  void ResetPunctuationState() {
    doubleQuoteOpen_ = false;
    singleQuoteOpen_ = false;
  }

  std::wstring ConfiguredPunctuationForKey(
      WPARAM key,
      const std::unordered_map<std::string, std::wstring> &shape) const {
    if (shape.empty()) {
      return L"";
    }
    const std::string label = PunctuationKeyLabel(key);
    if (label.empty()) {
      return L"";
    }
    auto found = shape.find(label);
    if (found != shape.end()) {
      return found->second;
    }
    return L"";
  }

  std::string PunctuationKeyLabel(WPARAM key) const {
    const bool shifted = IsShiftPressed();
    switch (key) {
      case VK_OEM_COMMA:
      case L',':
        return shifted ? "<" : ",";
      case VK_OEM_PERIOD:
      case L'.':
        return shifted ? ">" : ".";
      case VK_OEM_1:
      case L';':
        return shifted ? ":" : ";";
      case VK_OEM_2:
      case L'/':
        return shifted ? "?" : "/";
      case VK_OEM_4:
      case L'[':
        return shifted ? "{" : "[";
      case VK_OEM_6:
      case L']':
        return shifted ? "}" : "]";
      case VK_OEM_7:
      case L'\'':
        return shifted ? "\"" : "'";
      case VK_OEM_MINUS:
      case L'-':
        return shifted ? "_" : "-";
      case L'1':
        return shifted ? "!" : "1";
      case L'6':
        return shifted ? "^" : "6";
      case L'9':
        return shifted ? "(" : "9";
      case L'0':
        return shifted ? ")" : "0";
      case L'<':
        return "<";
      case L'>':
        return ">";
      case L':':
        return ":";
      case L'?':
        return "?";
      case L'{':
        return "{";
      case L'}':
        return "}";
      case L'"':
        return "\"";
      default:
        return "";
    }
  }

  std::wstring ChinesePunctuationForKey(WPARAM key) {
    if (!punctuationFullWidth_) {
      return HalfWidthPunctuationForKey(key);
    }
    const std::wstring configured = ConfiguredPunctuationForKey(key, punctuationFullShape_);
    if (!configured.empty()) {
      return configured;
    }
    const bool shifted = IsShiftPressed();
    switch (key) {
      case VK_OEM_COMMA:
      case L',':
        return shifted ? L"《" : L"，";
      case VK_OEM_PERIOD:
      case L'.':
        return shifted ? L"》" : L"。";
      case VK_OEM_1:
      case L';':
        return shifted ? L"：" : L"；";
      case VK_OEM_2:
      case L'/':
        return shifted ? L"？" : L"、";
      case VK_OEM_4:
      case L'[':
        return shifted ? L"【" : L"「";
      case VK_OEM_6:
      case L']':
        return shifted ? L"】" : L"」";
      case VK_OEM_7:
      case L'\'':
        if (shifted) {
          doubleQuoteOpen_ = !doubleQuoteOpen_;
          return doubleQuoteOpen_ ? L"“" : L"”";
        }
        singleQuoteOpen_ = !singleQuoteOpen_;
        return singleQuoteOpen_ ? L"‘" : L"’";
      case L'<':
        return L"《";
      case L'>':
        return L"》";
      case L':':
        return L"：";
      case L'?':
        return L"？";
      case L'{':
        return L"【";
      case L'}':
        return L"】";
      case L'"':
        doubleQuoteOpen_ = !doubleQuoteOpen_;
        return doubleQuoteOpen_ ? L"“" : L"”";
      case L'1':
        return shifted ? L"！" : L"";
      case L'6':
        return shifted ? L"……" : L"";
      case L'9':
        return shifted ? L"（" : L"";
      case L'0':
        return shifted ? L"）" : L"";
      case VK_OEM_MINUS:
      case L'-':
        return shifted ? L"——" : L"-";
      default:
        return L"";
    }
  }

  std::wstring HalfWidthPunctuationForKey(WPARAM key) const {
    const std::wstring configured = ConfiguredPunctuationForKey(key, punctuationHalfShape_);
    if (!configured.empty()) {
      return configured;
    }
    const bool shifted = IsShiftPressed();
    switch (key) {
      case VK_OEM_COMMA:
      case L',':
        return shifted ? L"<" : L",";
      case VK_OEM_PERIOD:
      case L'.':
        return shifted ? L">" : L".";
      case VK_OEM_1:
      case L';':
        return shifted ? L":" : L";";
      case VK_OEM_2:
      case L'/':
        return shifted ? L"?" : L"/";
      case VK_OEM_4:
      case L'[':
        return shifted ? L"{" : L"[";
      case VK_OEM_6:
      case L']':
        return shifted ? L"}" : L"]";
      case VK_OEM_7:
      case L'\'':
        return shifted ? L"\"" : L"'";
      case VK_OEM_MINUS:
      case L'-':
        return shifted ? L"_" : L"-";
      case L'<':
        return L"<";
      case L'>':
        return L">";
      case L':':
        return L":";
      case L'?':
        return L"?";
      case L'{':
        return L"{";
      case L'}':
        return L"}";
      case L'"':
        return L"\"";
      case L'1':
        return shifted ? L"!" : L"";
      case L'6':
        return shifted ? L"^" : L"";
      case L'9':
        return shifted ? L"(" : L"";
      case L'0':
        return shifted ? L")" : L"";
      default:
        return L"";
    }
  }

  std::wstring ConfigPath() {
    if (configPathResolved_) {
      return configPath_;
    }
    configPathResolved_ = true;
    wchar_t overridePath[MAX_PATH]{};
    DWORD len = GetEnvironmentVariableW(L"SHURUFA233_CONFIG", overridePath, ARRAYSIZE(overridePath));
    if (len > 0 && len < ARRAYSIZE(overridePath)) {
      configPath_ = overridePath;
      return configPath_;
    }
    wchar_t appData[MAX_PATH]{};
    len = GetEnvironmentVariableW(L"APPDATA", appData, ARRAYSIZE(appData));
    if (len == 0 || len >= ARRAYSIZE(appData)) {
      return L"";
    }
    configPath_ = std::wstring(appData) + L"\\shurufa233\\config.json";
    return configPath_;
  }

  static bool SameConfigFileTime(const FILETIME &left, const FILETIME &right) {
    return left.dwLowDateTime == right.dwLowDateTime &&
           left.dwHighDateTime == right.dwHighDateTime;
  }

  static std::string ReadConfigUtf8File(const std::wstring &path) {
    std::ifstream file(path, std::ios::binary);
    if (!file) {
      return "";
    }
    return std::string(std::istreambuf_iterator<char>(file),
                       std::istreambuf_iterator<char>());
  }

  static std::string JsonConfigStringField(const std::string &json, const char *field) {
    const std::string key = std::string("\"") + field + "\"";
    size_t pos = json.find(key);
    if (pos == std::string::npos) {
      return "";
    }
    pos = json.find(':', pos + key.size());
    if (pos == std::string::npos) {
      return "";
    }
    pos = json.find('"', pos + 1);
    if (pos == std::string::npos) {
      return "";
    }
    std::string out;
    bool escaped = false;
    for (size_t i = pos + 1; i < json.size(); ++i) {
      const char ch = json[i];
      if (escaped) {
        out.push_back(ch);
        escaped = false;
        continue;
      }
      if (ch == '\\') {
        escaped = true;
        continue;
      }
      if (ch == '"') {
        return out;
      }
      out.push_back(ch);
    }
    return "";
  }

  static bool JsonConfigBoolField(const std::string &json, const char *field) {
    const std::string key = std::string("\"") + field + "\"";
    const size_t keyPos = json.find(key);
    if (keyPos == std::string::npos) {
      return false;
    }
    const size_t colon = json.find(':', keyPos + key.size());
    if (colon == std::string::npos) {
      return false;
    }
    size_t pos = colon + 1;
    while (pos < json.size() && isspace(static_cast<unsigned char>(json[pos]))) {
      ++pos;
    }
    return json.compare(pos, 4, "true") == 0;
  }

  static int JsonConfigIntField(const std::string &json, const char *field) {
    const std::string key = std::string("\"") + field + "\"";
    const size_t keyPos = json.find(key);
    if (keyPos == std::string::npos) {
      return 0;
    }
    const size_t colon = json.find(':', keyPos + key.size());
    if (colon == std::string::npos) {
      return 0;
    }
    size_t pos = colon + 1;
    while (pos < json.size() && isspace(static_cast<unsigned char>(json[pos]))) {
      ++pos;
    }
    return atoi(json.c_str() + pos);
  }

  static void SkipJsonWhitespace(const std::string &json, size_t &pos) {
    while (pos < json.size() && isspace(static_cast<unsigned char>(json[pos]))) {
      ++pos;
    }
  }

  static int JsonHexValue(char ch) {
    if (ch >= '0' && ch <= '9') {
      return ch - '0';
    }
    if (ch >= 'a' && ch <= 'f') {
      return ch - 'a' + 10;
    }
    if (ch >= 'A' && ch <= 'F') {
      return ch - 'A' + 10;
    }
    return -1;
  }

  static bool ParseJsonHex4(const std::string &json, size_t pos, uint32_t &out) {
    if (pos + 4 > json.size()) {
      return false;
    }
    out = 0;
    for (size_t i = 0; i < 4; ++i) {
      const int value = JsonHexValue(json[pos + i]);
      if (value < 0) {
        return false;
      }
      out = (out << 4) | static_cast<uint32_t>(value);
    }
    return true;
  }

  static void AppendUtf8Codepoint(std::string &out, uint32_t codepoint) {
    if (codepoint <= 0x7F) {
      out.push_back(static_cast<char>(codepoint));
    } else if (codepoint <= 0x7FF) {
      out.push_back(static_cast<char>(0xC0 | (codepoint >> 6)));
      out.push_back(static_cast<char>(0x80 | (codepoint & 0x3F)));
    } else if (codepoint <= 0xFFFF) {
      out.push_back(static_cast<char>(0xE0 | (codepoint >> 12)));
      out.push_back(static_cast<char>(0x80 | ((codepoint >> 6) & 0x3F)));
      out.push_back(static_cast<char>(0x80 | (codepoint & 0x3F)));
    } else if (codepoint <= 0x10FFFF) {
      out.push_back(static_cast<char>(0xF0 | (codepoint >> 18)));
      out.push_back(static_cast<char>(0x80 | ((codepoint >> 12) & 0x3F)));
      out.push_back(static_cast<char>(0x80 | ((codepoint >> 6) & 0x3F)));
      out.push_back(static_cast<char>(0x80 | (codepoint & 0x3F)));
    }
  }

  static bool ParseJsonStringAt(const std::string &json, size_t &pos, std::string &out) {
    SkipJsonWhitespace(json, pos);
    if (pos >= json.size() || json[pos] != '"') {
      return false;
    }
    out.clear();
    bool escaped = false;
    for (++pos; pos < json.size(); ++pos) {
      const char ch = json[pos];
      if (escaped) {
        switch (ch) {
          case '"':
          case '\\':
          case '/':
            out.push_back(ch);
            break;
          case 'n':
            out.push_back('\n');
            break;
          case 'r':
            out.push_back('\r');
            break;
          case 't':
            out.push_back('\t');
            break;
          case 'u': {
            uint32_t codepoint = 0;
            if (ParseJsonHex4(json, pos + 1, codepoint)) {
              pos += 4;
              if (codepoint >= 0xD800 && codepoint <= 0xDBFF &&
                  pos + 6 < json.size() && json[pos + 1] == '\\' && json[pos + 2] == 'u') {
                uint32_t low = 0;
                if (ParseJsonHex4(json, pos + 3, low) && low >= 0xDC00 && low <= 0xDFFF) {
                  codepoint = 0x10000 + ((codepoint - 0xD800) << 10) + (low - 0xDC00);
                  pos += 6;
                }
              }
              AppendUtf8Codepoint(out, codepoint);
            } else {
              out.push_back(ch);
            }
            break;
          }
          default:
            out.push_back(ch);
            break;
        }
        escaped = false;
        continue;
      }
      if (ch == '\\') {
        escaped = true;
        continue;
      }
      if (ch == '"') {
        ++pos;
        return true;
      }
      out.push_back(ch);
    }
    return false;
  }

  static void SkipJsonValue(const std::string &json, size_t &pos) {
    SkipJsonWhitespace(json, pos);
    if (pos >= json.size()) {
      return;
    }
    if (json[pos] == '"') {
      std::string ignored;
      ParseJsonStringAt(json, pos, ignored);
      return;
    }
    int depth = 0;
    bool inString = false;
    bool escaped = false;
    for (; pos < json.size(); ++pos) {
      const char ch = json[pos];
      if (inString) {
        if (escaped) {
          escaped = false;
        } else if (ch == '\\') {
          escaped = true;
        } else if (ch == '"') {
          inString = false;
        }
        continue;
      }
      if (ch == '"') {
        inString = true;
      } else if (ch == '{' || ch == '[') {
        ++depth;
      } else if (ch == '}' || ch == ']') {
        if (depth == 0) {
          return;
        }
        --depth;
      } else if (depth == 0 && ch == ',') {
        return;
      }
    }
  }

  static std::wstring ParseJsonFirstStringValue(const std::string &json, size_t &pos) {
    SkipJsonWhitespace(json, pos);
    std::string value;
    if (pos < json.size() && json[pos] == '"') {
      if (ParseJsonStringAt(json, pos, value)) {
        return Utf8ToWide(value.c_str());
      }
      return L"";
    }
    if (pos < json.size() && json[pos] == '[') {
      ++pos;
      SkipJsonWhitespace(json, pos);
      if (pos < json.size() && json[pos] == '"' && ParseJsonStringAt(json, pos, value)) {
        while (pos < json.size() && json[pos] != ']') {
          ++pos;
        }
        if (pos < json.size()) {
          ++pos;
        }
        return Utf8ToWide(value.c_str());
      }
    }
    SkipJsonValue(json, pos);
    return L"";
  }

  static std::unordered_map<std::string, std::wstring> JsonConfigStringMapField(
      const std::string &json,
      const char *field) {
    std::unordered_map<std::string, std::wstring> out;
    const std::string key = std::string("\"") + field + "\"";
    size_t pos = json.find(key);
    if (pos == std::string::npos) {
      return out;
    }
    pos = json.find(':', pos + key.size());
    if (pos == std::string::npos) {
      return out;
    }
    ++pos;
    SkipJsonWhitespace(json, pos);
    if (pos >= json.size() || json[pos] != '{') {
      return out;
    }
    ++pos;
    for (;;) {
      SkipJsonWhitespace(json, pos);
      if (pos >= json.size() || json[pos] == '}') {
        break;
      }
      std::string itemKey;
      if (!ParseJsonStringAt(json, pos, itemKey)) {
        break;
      }
      SkipJsonWhitespace(json, pos);
      if (pos >= json.size() || json[pos] != ':') {
        break;
      }
      ++pos;
      std::wstring value = ParseJsonFirstStringValue(json, pos);
      if (!itemKey.empty() && !value.empty()) {
        out[itemKey] = value;
      }
      SkipJsonWhitespace(json, pos);
      if (pos < json.size() && json[pos] == ',') {
        ++pos;
        continue;
      }
      break;
    }
    return out;
  }

  void RefreshTypingConfigFromConfig() {
    const std::wstring path = ConfigPath();
    if (path.empty()) {
      ResetTypingConfigDefaults();
      return;
    }
    const DWORD now = GetTickCount();
    if (lastLocalConfigCheckTick_ != 0 && now - lastLocalConfigCheckTick_ < kSkinConfigPollMs) {
      return;
    }
    lastLocalConfigCheckTick_ = now;
    WIN32_FILE_ATTRIBUTE_DATA attrs{};
    if (!GetFileAttributesExW(path.c_str(), GetFileExInfoStandard, &attrs)) {
      ResetTypingConfigDefaults();
      return;
    }
    if (hasConfigWriteTime_ && SameConfigFileTime(lastConfigWriteTime_, attrs.ftLastWriteTime)) {
      return;
    }
    const std::string json = ReadConfigUtf8File(path);
    if (json.empty()) {
      ResetTypingConfigDefaults();
      return;
    }
    const std::string punctuation = JsonConfigStringField(json, "punctuation");
    punctuationFullWidth_ = punctuation != "half";
    punctuationFullShape_ = JsonConfigStringMapField(json, "punctuationFullShape");
    punctuationHalfShape_ = JsonConfigStringMapField(json, "punctuationHalfShape");
    recognizerPatterns_ = JsonConfigStringMapField(json, "recognizerPatterns");
    doublePinyinEnabled_ = JsonConfigBoolField(json, "doublePinyin");
    const std::string scheme = JsonConfigStringField(json, "doublePinyinScheme");
    microsoftDoublePinyin_ = doublePinyinEnabled_ &&
                             (scheme == "microsoft" || scheme == "ms" || scheme == "sogou");
    int pageSize = JsonConfigIntField(json, "candidatePageSize");
    if (pageSize <= 0) {
      pageSize = kDefaultCandidatesPerPage;
    }
    candidatePageSize_ = max(kMinCandidatesPerPage, min(kMaxCandidatesPerPage, pageSize));
    ApplyKeyBehaviorConfig(json);
    lastConfigWriteTime_ = attrs.ftLastWriteTime;
    hasConfigWriteTime_ = true;
  }

  void ResetTypingConfigDefaults() {
    punctuationFullWidth_ = true;
    punctuationFullShape_.clear();
    punctuationHalfShape_.clear();
    recognizerPatterns_.clear();
    doublePinyinEnabled_ = false;
    microsoftDoublePinyin_ = false;
    candidatePageSize_ = kDefaultCandidatesPerPage;
    ApplyWechatKeyBehavior();
  }

  void ApplyWechatKeyBehavior() {
    shiftToggleMode_ = true;
    semicolonQuickSelect_ = true;
    quoteQuickSelect_ = true;
    bracketPageKeys_ = true;
    minusEqualPageKeys_ = true;
    commaPeriodPageKeys_ = false;
  }

  void ApplyRimeKeyBehavior() {
    shiftToggleMode_ = true;
    semicolonQuickSelect_ = false;
    quoteQuickSelect_ = false;
    bracketPageKeys_ = true;
    minusEqualPageKeys_ = true;
    commaPeriodPageKeys_ = true;
  }

  void ApplyKeyBehaviorConfig(const std::string &json) {
    const std::string profile = JsonConfigStringField(json, "keyProfile");
    if (profile == "rime" || profile == "weasel" || profile == "squirrel") {
      ApplyRimeKeyBehavior();
      return;
    }
    if (profile == "custom") {
      shiftToggleMode_ = JsonConfigBoolField(json, "shiftToggleMode");
      semicolonQuickSelect_ = JsonConfigBoolField(json, "semicolonQuickSelect");
      quoteQuickSelect_ = JsonConfigBoolField(json, "quoteQuickSelect");
      bracketPageKeys_ = JsonConfigBoolField(json, "bracketPageKeys");
      minusEqualPageKeys_ = JsonConfigBoolField(json, "minusEqualPageKeys");
      commaPeriodPageKeys_ = JsonConfigBoolField(json, "commaPeriodPageKeys");
      return;
    }
    ApplyWechatKeyBehavior();
  }

  void RefreshAsciiModeFromCore() {
    if (!session_ || !g_core.mode) {
      asciiMode_ = false;
      return;
    }
    char *mode = g_core.mode(session_);
    asciiMode_ = ModePayloadIsEnglish(mode);
    if (mode) {
      g_core.freeValue(mode);
    }
  }

  void ToggleAsciiMode() {
    if (session_ && g_core.toggleMode) {
      char *state = g_core.toggleMode(session_);
      asciiMode_ = ModePayloadIsEnglish(state);
      if (state) {
        g_core.freeValue(state);
      }
    } else {
      asciiMode_ = !asciiMode_;
      if (session_ && g_core.setMode) {
        char *state = g_core.setMode(session_, asciiMode_ ? "en" : "zh");
        if (state) {
          g_core.freeValue(state);
        }
      }
    }
    ResetCompositionState();
    ResetPunctuationState();
    candidateWindow_.Hide();
    wchar_t message[96]{};
    StringCchPrintfW(message, ARRAYSIZE(message), L"ToggleAsciiMode ascii=%d", asciiMode_ ? 1 : 0);
    LogDebugLine(message);
    candidateWindow_.ShowStatus(asciiMode_ ? L"EN" : L"中");
  }

  long refCount_ = 1;
  ITfThreadMgr *threadMgr_ = nullptr;
  ITfKeystrokeMgr *keyMgr_ = nullptr;
  TfClientId clientId_ = TF_CLIENTID_NULL;
  uint64_t session_ = 0;
  int selectedIndex_ = 0;
  int pageOffset_ = 0;
  int cachedCandidateCount_ = 0;
  int compositionLength_ = 0;
  std::string rawBuffer_;
  bool asciiMode_ = false;
  bool shiftDown_ = false;
  bool shiftToggleCandidate_ = false;
  bool doubleQuoteOpen_ = false;
  bool singleQuoteOpen_ = false;
  bool punctuationFullWidth_ = true;
  std::unordered_map<std::string, std::wstring> punctuationFullShape_;
  std::unordered_map<std::string, std::wstring> punctuationHalfShape_;
  std::unordered_map<std::string, std::wstring> recognizerPatterns_;
  bool doublePinyinEnabled_ = false;
  bool microsoftDoublePinyin_ = false;
  bool shiftToggleMode_ = true;
  bool semicolonQuickSelect_ = true;
  bool quoteQuickSelect_ = true;
  bool bracketPageKeys_ = true;
  bool minusEqualPageKeys_ = true;
  bool commaPeriodPageKeys_ = false;
  int candidatePageSize_ = kDefaultCandidatesPerPage;
  std::wstring configPath_;
  bool configPathResolved_ = false;
  DWORD lastLocalConfigCheckTick_ = 0;
  FILETIME lastConfigWriteTime_{};
  bool hasConfigWriteTime_ = false;
  ITfContext *lastContext_ = nullptr;
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
    LogDebugLine(L"ClassFactory CreateInstance called");
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
    LogDebugLine(L"DllMain PROCESS_ATTACH");
  } else if (reason == DLL_PROCESS_DETACH) {
    LogDebugLine(L"DllMain PROCESS_DETACH");
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
  LogDebugLine(L"DllGetClassObject called");
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
