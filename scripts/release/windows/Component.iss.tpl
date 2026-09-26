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
OutputDir=__OUTPUT__
OutputBaseFilename=__BASENAME__
Compression=lzma2
SolidCompression=yes
WizardStyle=modern
SetupIconFile=__SOURCE__\curated.ico
UninstallDisplayIcon={app}\curated.ico
CloseApplications=yes
RestartApplications=no
CloseApplicationsFilter=__EXE__

[Files]
Source: "__SOURCE__\*"; DestDir: "{app}"; Flags: recursesubdirs createallsubdirs ignoreversion

[InstallDelete]
Type: filesandordirs; Name: "{app}\__MANAGED_DELETE__"

[Icons]
Name: "{autoprograms}\Curated __COMPONENT__"; Filename: "{app}\__EXE__"; Parameters: "__PARAMS__"; IconFilename: "{app}\curated.ico"

[Run]
Filename: "{app}\__EXE__"; Parameters: "__PARAMS__"; Description: "Launch Curated __COMPONENT__"; Flags: __RUN_FLAGS__; Check: ShouldLaunch

[Code]
function ShouldLaunch: Boolean;
begin
  Result := ExpandConstant('{param:NOLAUNCH|0}') <> '1';
end;

function InitializeSetup: Boolean;
var
  Installed: String;
  OldVersion, NewVersion: Int64;
begin
  Result := False;
  if __LEGACY_CHECK__ and (RegKeyExists(HKLM64, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{8C9E9E66-7058-4D09-9F9A-8AFD060A7E1B}_is1') or
     RegKeyExists(HKLM32, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{8C9E9E66-7058-4D09-9F9A-8AFD060A7E1B}_is1') or
     RegKeyExists(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{8C9E9E66-7058-4D09-9F9A-8AFD060A7E1B}_is1')) then begin
    SuppressibleMsgBox('An older all-in-one Curated installation was found. Use Curated Full 1.7.2 or later to back up and upgrade this installation automatically. Fully exit Curated before running Full. See the migration guide for custom configurations.', mbError, MB_OK, IDOK);
    exit;
  end;
  if RegQueryStringValue(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\__APP_ID___is1', 'DisplayVersion', Installed) then begin
    if not StrToVersion(Installed, OldVersion) or not StrToVersion('__VERSION__', NewVersion) then exit;
    if ComparePackedVersion(OldVersion, NewVersion) > 0 then begin
      SuppressibleMsgBox('A newer Curated __COMPONENT__ is already installed. Downgrade was cancelled.', mbError, MB_OK, IDOK);
      exit;
    end;
  end;
  Result := True;
end;

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
