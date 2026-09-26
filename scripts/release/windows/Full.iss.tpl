[Setup]
AppName=Curated Full
AppVersion=__VERSION__
AppPublisher=Curated
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
CloseApplications=yes
RestartApplications=no
CreateAppDir=no
Uninstallable=no
DisableWelcomePage=no
OutputDir=__OUTPUT__
OutputBaseFilename=__BASENAME__
Compression=lzma2
SolidCompression=yes
WizardStyle=modern

[Files]
Source: "__MIGRATION_HELPER__"; DestName: "curated-migrate.exe"; Flags: dontcopy
Source: "__SERVER_INSTALLER__"; DestName: "server-setup.exe"; Flags: dontcopy
Source: "__DESKTOP_INSTALLER__"; DestName: "desktop-setup.exe"; Flags: dontcopy

[Code]
var
  Failure, LegacyDirectory: String;
  DataPage: TInputDirWizardPage;
  ConfigPage: TInputFileWizardPage;

procedure InitializeWizard;
var
  DataRoot: String;
begin
  if not RegQueryStringValue(HKLM64, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{8C9E9E66-7058-4D09-9F9A-8AFD060A7E1B}_is1', 'InstallLocation', LegacyDirectory) then
    if not RegQueryStringValue(HKLM32, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{8C9E9E66-7058-4D09-9F9A-8AFD060A7E1B}_is1', 'InstallLocation', LegacyDirectory) then
      RegQueryStringValue(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{8C9E9E66-7058-4D09-9F9A-8AFD060A7E1B}_is1', 'InstallLocation', LegacyDirectory);
  DataPage := CreateInputDirPage(wpWelcome, 'Upgrade your existing Curated',
    'Keep your original library and settings',
    'Fully quit Curated from its tray. Confirm the original data directory (containing data and config). Full will verify a backup, remove the old program and install Server and Desktop. Windows may request permission to remove the old program. Your media files stay in place.', False, '');
  DataPage.Add('Original data directory:');
  DataRoot := GetEnv('CURATED_DATA_DIR');
  if DataRoot = '' then DataRoot := ExpandConstant('{localappdata}\Curated');
  DataPage.Values[0] := ExpandConstant('{param:LEGACYDATADIR|' + DataRoot + '}');
  ConfigPage := CreateInputFilePage(DataPage.ID, 'Custom Server configuration',
    'Only needed if the old Server used -config',
    'Select the same runtime configuration file if you previously started Server with -config. Otherwise leave this blank. Custom filesystem paths must be absolute and outside the old program directory.');
  ConfigPage.Add('Original runtime configuration (optional):', 'Configuration files|*.json;*.cfg;*.yaml|All files|*.*', '');
  ConfigPage.Values[0] := ExpandConstant('{param:LEGACYCONFIG|}');
end;

function ShouldSkipPage(PageID: Integer): Boolean;
begin
  Result := (LegacyDirectory = '') and ((PageID = DataPage.ID) or (PageID = ConfigPage.ID));
end;

procedure RegisterExtraCloseApplicationsResources;
begin
  if LegacyDirectory <> '' then begin
#if VER >= EncodeVer(7, 0, 0)
    RegisterExtraCloseApplicationsResource(AddBackslash(LegacyDirectory) + 'Curated.exe');
    RegisterExtraCloseApplicationsResource(AddBackslash(LegacyDirectory) + 'resources\app\curated.exe');
#else
    RegisterExtraCloseApplicationsResource(False, AddBackslash(LegacyDirectory) + 'Curated.exe');
    RegisterExtraCloseApplicationsResource(False, AddBackslash(LegacyDirectory) + 'resources\app\curated.exe');
#endif
  end;
end;

function RunMigration(Action: String): Boolean;
var
  Code: Integer;
  Parameters, ErrorPath: String;
  ErrorText: AnsiString;
begin
  ExtractTemporaryFile('curated-migrate.exe');
  ErrorPath := ExpandConstant('{tmp}\curated-migration-error.txt');
  DeleteFile(ErrorPath);
  Parameters := '-action ' + Action + ' -error-file "' + ErrorPath + '"';
  if LegacyDirectory <> '' then begin
    Parameters := Parameters + ' -data-root "' + AddBackslash(DataPage.Values[0]) + '."';
    if ConfigPage.Values[0] <> '' then Parameters := Parameters + ' -config "' + ConfigPage.Values[0] + '"';
  end;
  Result := Exec(ExpandConstant('{tmp}\curated-migrate.exe'), Parameters, '', SW_HIDE, ewWaitUntilTerminated, Code);
  if Result then Result := Code = 0;
  if not Result then begin
    Failure := 'The Curated upgrade could not finish. Run Full again after resolving the error.';
    if LoadStringFromFile(ErrorPath, ErrorText) then Failure := UTF8Decode(ErrorText);
    SuppressibleMsgBox(Failure, mbError, MB_OK, IDOK);
  end;
end;

function InstallComponent(Name, Version, Installer: String): Boolean;
var
  Installed: String;
  OldVersion, NewVersion: Int64;
  Code: Integer;
begin
  Result := False;
  if RegQueryStringValue(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\Curated.' + Name + '_is1', 'DisplayVersion', Installed) then begin
    if not StrToVersion(Installed, OldVersion) or not StrToVersion(Version, NewVersion) then begin
      Failure := Name + ': installed version could not be read.';
      exit;
    end;
    if ComparePackedVersion(OldVersion, NewVersion) >= 0 then begin
      Log(Name + ' already installed at ' + Installed + '; reusing component.');
      Result := True;
      exit;
    end;
  end;
  ExtractTemporaryFile(Installer);
  if not Exec(ExpandConstant('{tmp}\') + Installer, '/VERYSILENT /SUPPRESSMSGBOXES /SP- /NORESTART /NOLAUNCH=1 /LOG="' + ExpandConstant('{tmp}\Curated-') + Name + '-install.log"', '', SW_HIDE, ewWaitUntilTerminated, Code) then
    Failure := Name + ': could not start installer.'
  else if (Code <> 0) and (Code <> 3010) then
    Failure := Name + ': installer failed with code ' + IntToStr(Code) + '.'
  else Result := True;
end;

procedure CurStepChanged(CurStep: TSetupStep);
var
  ServerOK, DesktopOK: Boolean;
  Directory: String;
  Code: Integer;
begin
  if CurStep <> ssPostInstall then exit;
  if not RunMigration('prepare') then exit;
  ServerOK := InstallComponent('Server', '__SERVER_VERSION__', 'server-setup.exe');
  if not ServerOK then begin
    SuppressibleMsgBox(Failure + ' Desktop was not changed. Existing components and library data were kept. Resolve the error and run Full again.', mbError, MB_OK, IDOK);
    exit;
  end;
  DesktopOK := InstallComponent('Desktop', '__DESKTOP_VERSION__', 'desktop-setup.exe');
  if not DesktopOK then begin
    SuppressibleMsgBox(Failure + ' Server is installed. Existing components and library data were kept. Run Full again to retry Desktop.', mbError, MB_OK, IDOK);
    exit;
  end;
  if not RunMigration('complete') then exit;
  if ExpandConstant('{param:NOLAUNCH|0}') = '1' then exit;
  if RegQueryStringValue(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\Curated.Server_is1', 'InstallLocation', Directory) then
    Exec(AddBackslash(Directory) + 'curated.exe', '-mode tray -autostart', Directory, SW_HIDE, ewNoWait, Code);
  if RegQueryStringValue(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\Curated.Desktop_is1', 'InstallLocation', Directory) then
    Exec(AddBackslash(Directory) + 'Curated Desktop.exe', '--curated-connect-local', Directory, SW_SHOWNORMAL, ewNoWait, Code);
end;

function GetCustomSetupExitCode: Integer;
begin
  if Failure <> '' then Result := 1 else Result := 0;
end;
