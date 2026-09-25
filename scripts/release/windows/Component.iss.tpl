#define Component "__COMPONENT__"
#define Title "__TITLE__"
#define Version "__VERSION__"
[Setup]
AppId={{__APP_ID__}
AppName=Curated __TITLE__
AppVersion={#Version}
DefaultDirName={localappdata}\Programs\Curated\__TITLE__
PrivilegesRequired=lowest
DisableProgramGroupPage=yes
OutputDir=__OUTPUT__
OutputBaseFilename=__SETUP_NAME__
Compression=lzma
SolidCompression=yes
WizardStyle=modern
CloseApplications=yes
CloseApplicationsFilter=__EXE__
RestartApplications=no
UninstallDisplayIcon={app}\curated.ico
[Files]
Source: "__PAYLOAD__\*"; DestDir: "{app}"; Flags: recursesubdirs createallsubdirs ignoreversion
[InstallDelete]
#if Component == "server"
Type: filesandordirs; Name: "{app}\frontend-dist"
#else
Type: filesandordirs; Name: "{app}\resources\app\electron-dist\launcher"
#endif
[Icons]
Name: "{autoprograms}\Curated __TITLE__"; Filename: "{app}\__EXE__"; Parameters: "__ARGS__"; WorkingDir: "{app}"
[Run]
Filename: "{app}\__EXE__"; Parameters: "__ARGS__"; WorkingDir: "{app}"; Description: "Launch Curated __TITLE__"; Flags: nowait postinstall; Check: ShouldStartApp
[Code]
function ShouldStartApp(): Boolean;
begin
  Result := Pos('/NOAPPSTART', Uppercase(GetCmdTail)) = 0;
end;
function VersionPart(var Value: String): Integer;
var Position: Integer; Part: String;
begin
  Position := Pos('.', Value);
  if Position = 0 then begin Part := Value; Value := ''; end
  else begin Part := Copy(Value, 1, Position - 1); Delete(Value, 1, Position); end;
  Result := StrToIntDef(Part, 0);
end;
function IsNewer(Installed: String; Incoming: String): Boolean;
var Index, Left, Right: Integer;
begin
  Result := False;
  for Index := 1 to 3 do begin
    Left := VersionPart(Installed); Right := VersionPart(Incoming);
    if Left > Right then begin Result := True; Exit; end;
    if Left < Right then Exit;
  end;
end;
function InitializeSetup(): Boolean;
var Installed: String;
begin
  Result := False;
  if RegKeyExists(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{8C9E9E66-7058-4D09-9F9A-8AFD060A7E1B}_is1') or
     RegKeyExists(HKLM, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{8C9E9E66-7058-4D09-9F9A-8AFD060A7E1B}_is1') or
     RegKeyExists(HKLM64, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{8C9E9E66-7058-4D09-9F9A-8AFD060A7E1B}_is1') then begin
    MsgBox('An older combined Curated installation was detected. Complete the documented backup and migration before installing independent components.', mbError, MB_OK);
    Exit;
  end;
  if RegQueryStringValue(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{__APP_ID__}_is1', 'DisplayVersion', Installed) and IsNewer(Installed, '{#Version}') then begin
    MsgBox('A newer Curated __TITLE__ is already installed. Downgrading is not supported.', mbError, MB_OK);
    Exit;
  end;
  Result := True;
end;
