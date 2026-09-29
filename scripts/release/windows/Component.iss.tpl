#define CuratedWindowsSupport "__WINDOWS_SUPPORT__"
#include "__WINDOWS_SUPPORT__\Languages.iss"

[Setup]
AppId=__APP_ID__
AppName=Curated __COMPONENT__
AppVersion=__VERSION__
AppPublisher=Curated
DefaultDirName={localappdata}\Programs\Curated\__COMPONENT__
DefaultGroupName=Curated
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
DisableProgramGroupPage=yes
DisableWelcomePage=no
DisableDirPage=no
UsePreviousAppDir=yes
SetupLogging=yes
SetupMutex=Curated.Setup.__APP_ID__
OutputDir=__OUTPUT__
OutputBaseFilename=__BASENAME__
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
ShowLanguageDialog=yes
LanguageDetectionMethod=uilanguage
SetupIconFile=__SOURCE__\curated.ico
UninstallDisplayIcon={app}\curated.ico
; The scoped helper closes and waits before InstallDelete can remove old files.
CloseApplications=no
RestartApplications=no
CloseApplicationsFilter=__EXE__

[Files]
Source: "__MIGRATION_HELPER__"; DestName: "curated-migrate.exe"; Flags: dontcopy
Source: "__SOURCE__\*"; DestDir: "{app}"; Flags: recursesubdirs createallsubdirs ignoreversion

[InstallDelete]
Type: filesandordirs; Name: "{app}\__MANAGED_DELETE__"

[Icons]
Name: "{autoprograms}\Curated __COMPONENT__"; Filename: "{app}\__EXE__"; Parameters: "__PARAMS__"; IconFilename: "{app}\curated.ico"

[Run]
Filename: "{app}\__EXE__"; Parameters: "{code:LaunchParameters}"; Description: "{cm:LaunchApp,Curated __COMPONENT__}"; Flags: __RUN_FLAGS__; Check: ShouldLaunch

[Code]
var
  InstalledVersion, InstalledDirectory, InstallMode, HelperError: String;
  RunningPage: TWizardPage;
  RunningLabel: TNewStaticText;
  RecheckButton: TNewButton;
  StopProgress: TOutputProgressWizardPage;
  RunningState: Integer;
  WasRunning, StopRequested: Boolean;

{ Normalize directory spelling for registry and wizard comparisons. }
function NormalPath(Path: String): String;
begin
  Result := RemoveBackslashUnlessRoot(ExpandFileName(Path));
end;

{ Compare Windows directory paths without case or trailing-slash differences. }
function SamePath(A, B: String): Boolean;
begin
  Result := CompareText(NormalPath(A), NormalPath(B)) = 0;
end;

{ Reject a concurrent change to the captured installation registration. }
function SnapshotUnchanged: Boolean;
var
  Version, Directory: String;
  Exists: Boolean;
begin
  Exists := RegQueryStringValue(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\__APP_ID___is1', 'DisplayVersion', Version);
  if InstalledVersion = '' then
    Result := not Exists
  else
    Result := Exists and (Version = InstalledVersion) and
      RegQueryStringValue(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\__APP_ID___is1', 'InstallLocation', Directory) and
      SamePath(Directory, InstalledDirectory);
end;

function RunHelper(Action: String): Integer; forward;

{ Validate the selected destination before shutdown or resource replacement. }
function DirectoryError: String;
var
  Find: TFindRec;
  Directory: String;
begin
  Result := '';
  Directory := WizardDirValue;
  if (InstalledVersion <> '') and not SamePath(Directory, InstalledDirectory) then begin
    Result := FmtMessage(CustomMessage('KeepDirectory'), [InstalledDirectory]);
    exit;
  end;
  { A first install must never adopt someone else's files or another component. }
  if (InstalledVersion = '') and FindFirst(AddBackslash(Directory) + '*', Find) then begin
    try
      repeat
        if (Find.Name <> '.') and (Find.Name <> '..') then begin
          Result := CustomMessage('DirectoryNotEmpty');
          break;
        end;
      until not FindNext(Find);
    finally
      FindClose(Find);
    end;
  end;
  if (Result = '') and (RunHelper('validate-directory') <> 0) then
    Result := CustomMessage('InvalidDirectory') + #13#10 + HelperError;
end;

{ Run the embedded helper for the exact selected directory and collect diagnostics. }
function RunHelper(Action: String): Integer;
var
  ErrorPath, Parameters: String;
  ErrorText: AnsiString;
begin
  HelperError := '';
  ExtractTemporaryFile('curated-migrate.exe');
  ErrorPath := ExpandConstant('{tmp}\curated-stop-error.txt');
  DeleteFile(ErrorPath);
  Parameters := '-action ' + Action + ' -program-dir "' + AddBackslash(WizardDirValue) + '." -error-file "' + ErrorPath + '"';
  if Action = 'validate-directory' then
    Parameters := Parameters + ' -installed-dir "' + AddBackslash(InstalledDirectory) + '."';
  if (Action = 'validate-directory') and (InstalledDirectory = '') then
    Parameters := '-action validate-directory -program-dir "' + AddBackslash(WizardDirValue) + '." -error-file "' + ErrorPath + '"';
  Result := 1;
  if not Exec(ExpandConstant('{tmp}\curated-migrate.exe'), Parameters, '', SW_HIDE, ewWaitUntilTerminated, Result) then
    Result := 1;
  if LoadStringFromFile(ErrorPath, ErrorText) then HelperError := UTF8Decode(ErrorText);
  Log('Curated ' + Action + ': ' + IntToStr(Result));
end;

{ Refresh the read-only running state and enable only the appropriate action. }
procedure RefreshRunning;
begin
  RunningState := RunHelper('probe');
  if RunningState = 2 then begin
    WasRunning := True;
    RunningLabel.Caption := FmtMessage(CustomMessage('RunningDetails'), ['Curated __COMPONENT__']) + #13#10#13#10 + CustomMessage('__COMPONENT__Impact');
    WizardForm.NextButton.Caption := CustomMessage('CloseContinue');
    WizardForm.NextButton.Enabled := True;
  end else if RunningState = 0 then begin
    RunningLabel.Caption := CustomMessage('StoppedDetails');
    WizardForm.NextButton.Caption := CustomMessage('ContinueInstall');
    WizardForm.NextButton.Enabled := True;
  end else begin
    RunningLabel.Caption := CustomMessage('ProbeFailed') + #13#10#13#10 + HelperError;
    WizardForm.NextButton.Enabled := False;
  end;
end;

{ Let users retry detection after exiting from the tray themselves. }
procedure RecheckClick(Sender: TObject);
begin
  RefreshRunning;
end;

{ Build the dedicated pages using the wizard's standard button dimensions. }
procedure InitializeWizard;
begin
  { Leave room for the themed checkbox glyph at scaled display sizes. }
  WizardForm.RunList.Offset := ScaleX(8);
  WizardForm.WelcomeLabel2.Caption := FmtMessage(CustomMessage('InstallIntroduction'), ['Curated __COMPONENT__', '__VERSION__']);
  if InstalledVersion <> '' then
    WizardForm.WelcomeLabel2.Caption := WizardForm.WelcomeLabel2.Caption + #13#10#13#10 +
      FmtMessage(CustomMessage('ExistingVersion'), [InstalledVersion, '__VERSION__']);
  WizardForm.WelcomeLabel2.Caption := WizardForm.WelcomeLabel2.Caption + #13#10#13#10 + CustomMessage('__COMPONENT__Purpose');
  WizardForm.SelectDirLabel.Caption := CustomMessage('ChooseDirectory');
  RunningPage := CreateCustomPage(wpReady, CustomMessage('RunningTitle'), CustomMessage('RunningSubtitle'));
  RunningLabel := TNewStaticText.Create(RunningPage);
  RunningLabel.Parent := RunningPage.Surface;
  RunningLabel.SetBounds(0, 0, RunningPage.SurfaceWidth, ScaleY(175));
  RunningLabel.AutoSize := False;
  RunningLabel.WordWrap := True;
  RecheckButton := TNewButton.Create(RunningPage);
  RecheckButton.Parent := RunningPage.Surface;
  RecheckButton.SetBounds(0, ScaleY(185), WizardForm.NextButton.Width, WizardForm.NextButton.Height);
  RecheckButton.Caption := CustomMessage('Recheck');
  RecheckButton.OnClick := @RecheckClick;
  StopProgress := CreateOutputProgressPage(CustomMessage('ClosingTitle'), CustomMessage('ClosingDetails'));
end;

{ Omit shutdown UI when stopped or when running without a wizard. }
function ShouldSkipPage(PageID: Integer): Boolean;
begin
  Result := False;
  if PageID = RunningPage.ID then begin
    if WizardSilent then begin
      Result := True;
      exit;
    end;
    { Only a read-only probe occurs before showing the conditional page. }
    RunningState := RunHelper('probe');
    if RunningState = 2 then WasRunning := True;
    Result := RunningState = 0;
  end;
end;

{ Update confirmation buttons and default launch choice on page entry. }
procedure CurPageChanged(CurPageID: Integer);
begin
  if CurPageID = wpReady then begin
    WizardForm.NextButton.Enabled := True;
    WizardForm.NextButton.Caption := CustomMessage(InstallMode);
  end;
  if CurPageID = RunningPage.ID then RefreshRunning;
  if CurPageID = wpFinished then begin
    if (InstalledVersion <> '') and not WasRunning and (WizardForm.RunList.Items.Count > 0) then
      WizardForm.RunList.Checked[0] := False;
  end;
end;

{ Summarize version, location and data preservation before installation. }
function UpdateReadyMemo(Space, NewLine, MemoUserInfoInfo, MemoDirInfo, MemoTypeInfo, MemoComponentsInfo, MemoGroupInfo, MemoTasksInfo: String): String;
begin
  Result := 'Curated __COMPONENT__' + NewLine;
  if InstalledVersion <> '' then
    Result := Result + FmtMessage(CustomMessage('ExistingVersion'), [InstalledVersion, '__VERSION__']) + NewLine
  else
    Result := Result + FmtMessage(CustomMessage('TargetVersion'), ['__VERSION__']) + NewLine;
  Result := Result + NewLine + MemoDirInfo + NewLine + NewLine + CustomMessage('__COMPONENT__KeepData');
  if InstallMode = 'UpgradeInstall' then Result := Result + NewLine + NewLine + CustomMessage('OverwriteQuestion');
  if InstallMode = 'Reinstall' then Result := Result + NewLine + NewLine + CustomMessage('ReinstallQuestion');
end;

{ Validate navigation and request shutdown only after explicit confirmation. }
function NextButtonClick(CurPageID: Integer): Boolean;
var
  Error: String;
  Code: Integer;
begin
  Result := True;
  if (CurPageID = wpSelectDir) or (CurPageID = wpReady) or (CurPageID = RunningPage.ID) then begin
    Error := DirectoryError;
    if not SnapshotUnchanged then Error := CustomMessage('InstallationChanged');
    if Error <> '' then begin
      SuppressibleMsgBox(Error, mbError, MB_OK, IDOK);
      Result := False;
      exit;
    end;
  end;
  if CurPageID = RunningPage.ID then begin
    if WizardSilent then exit;
    RefreshRunning;
    if (RunningState <> 0) and (RunningState <> 2) then begin
      Result := False;
      exit;
    end;
    if RunningState <> 0 then begin
      StopRequested := True;
      StopProgress.SetText(CustomMessage('ClosingDetails'), '');
      StopProgress.Show;
      try
        Code := RunHelper('stop');
      finally
        StopProgress.Hide;
      end;
      if Code <> 0 then begin
        SuppressibleMsgBox(CustomMessage('CloseFailed') + #13#10 + HelperError, mbError, MB_OK, IDOK);
        Result := False;
        RefreshRunning;
        exit;
      end;
      RefreshRunning;
      Result := RunningState = 0;
    end;
  end;
end;

{ Recheck destination and running state immediately before replacing files. }
function PrepareToInstall(var NeedsRestart: Boolean): String;
var
  Code: Integer;
begin
  Result := DirectoryError;
  if Result <> '' then exit;
  if not SnapshotUnchanged then begin
    Result := CustomMessage('InstallationChanged');
    exit;
  end;
  Code := RunHelper('probe');
  if Code = 2 then begin
    WasRunning := True;
    { Silent automation needs explicit shutdown consent. Interactive runs must
      return to the running page if a new instance started after confirmation. }
    if WizardSilent and (ExpandConstant('{param:CLOSECURATED|0}') = '1') then begin
      StopRequested := True;
      Code := RunHelper('stop');
      if Code = 0 then Code := RunHelper('probe');
    end;
    if Code = 2 then Result := CustomMessage('RunningAgain');
  end;
  if (Code <> 0) and (Result = '') then Result := CustomMessage('ProbeFailed') + #13#10 + HelperError;
end;

{ Use management launch for a fresh Server and quiet startup for upgrades. }
function LaunchParameters(Param: String): String;
begin
  Result := '__PARAMS__';
  { Manual first launch opens the local management page after Server is ready.
    Upgrades restore background service without opening an extra browser tab. }
  if __LEGACY_CHECK__ and (InstalledVersion <> '') then Result := '-mode tray -autostart';
end;

{ Respect callers that explicitly suppress the completion launch option. }
function ShouldLaunch: Boolean;
begin
  Result := ExpandConstant('{param:NOLAUNCH|0}') <> '1';
end;

{ Explain the application state when cancelling after a shutdown request. }
procedure CancelButtonClick(CurPageID: Integer; var Cancel, Confirm: Boolean);
begin
  if StopRequested and ((CurPageID <= wpReady) or (CurPageID = RunningPage.ID)) then
    MsgBox(CustomMessage('CancelledAfterStop'), mbInformation, MB_OK);
end;

{ Capture existing registration and block legacy conflicts or downgrades early. }
function InitializeSetup: Boolean;
var
  OldVersion, NewVersion: Int64;
begin
  Result := False;
  InstallMode := 'FreshInstall';
  if __LEGACY_CHECK__ and (RegKeyExists(HKLM64, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{8C9E9E66-7058-4D09-9F9A-8AFD060A7E1B}_is1') or
     RegKeyExists(HKLM32, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{8C9E9E66-7058-4D09-9F9A-8AFD060A7E1B}_is1') or
     RegKeyExists(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{8C9E9E66-7058-4D09-9F9A-8AFD060A7E1B}_is1')) then begin
    SuppressibleMsgBox(CustomMessage('LegacyDetected'), mbError, MB_OK, IDOK);
    exit;
  end;
  if RegKeyExists(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\__APP_ID___is1') then begin
    if not RegQueryStringValue(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\__APP_ID___is1', 'DisplayVersion', InstalledVersion) or
       not StrToVersion(InstalledVersion, OldVersion) or not StrToVersion('__VERSION__', NewVersion) or
       not RegQueryStringValue(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\__APP_ID___is1', 'InstallLocation', InstalledDirectory) or
       (InstalledDirectory = '') then begin
      SuppressibleMsgBox(CustomMessage('InstallationUnreadable'), mbError, MB_OK, IDOK);
      exit;
    end;
    if ComparePackedVersion(OldVersion, NewVersion) > 0 then begin
      SuppressibleMsgBox(FmtMessage(CustomMessage('DowngradeDetails'), [InstalledVersion, '__VERSION__']), mbError, MB_OK, IDOK);
      exit;
    end;
    if ComparePackedVersion(OldVersion, NewVersion) = 0 then InstallMode := 'Reinstall'
    else InstallMode := 'UpgradeInstall';
  end;
  Result := True;
end;

{ Remove only the Server startup command owned by this installation. }
procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var
  Command: String;
begin
  if (CurUninstallStep = usPostUninstall) and __LEGACY_CHECK__ then begin
    if RegQueryStringValue(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Run', 'Curated', Command) then
      if Command = '"' + ExpandConstant('{app}\curated.exe') + '" -mode tray -autostart' then
        RegDeleteValue(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Run', 'Curated');
  end;
end;
