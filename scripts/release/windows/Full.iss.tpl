[Setup]
AppId=Curated.Full.Bootstrap
AppName=Curated Full
AppVersion=__VERSION__
CreateAppDir=no
Uninstallable=no
PrivilegesRequired=lowest
DisableProgramGroupPage=yes
OutputDir=__OUTPUT__
OutputBaseFilename=__SETUP_NAME__
Compression=lzma
SolidCompression=yes
WizardStyle=modern
[Files]
Source: "__OUTPUT__\__SERVER_SETUP__"; DestDir: "{tmp}"; Flags: deleteafterinstall
Source: "__OUTPUT__\__DESKTOP_SETUP__"; DestDir: "{tmp}"; Flags: deleteafterinstall
[Code]
// Read the component's registered path so an existing custom installation is reused.
procedure LaunchComponent(AppId: String; Executable: String; Arguments: String);
var Directory: String; Code: Integer;
begin
  if not RegQueryStringValue(HKCU, 'Software\Microsoft\Windows\CurrentVersion\Uninstall\{' + AppId + '}_is1', 'Inno Setup: App Path', Directory) then begin
    MsgBox('The component is installed, but its launch path could not be read. Open it from the Start menu.', mbInformation, MB_OK);
    Exit;
  end;
  if not Exec(AddBackslash(Directory) + Executable, Arguments, Directory, SW_SHOW, ewNoWait, Code) then
    MsgBox('The component is installed, but could not be started. Open it from the Start menu.', mbInformation, MB_OK);
end;
// Install both standalone payloads without rolling back an existing component on failure.
procedure CurStepChanged(CurStep: TSetupStep);
var Code: Integer;
begin
  if CurStep = ssPostInstall then begin
    if not Exec(ExpandConstant('{tmp}\__SERVER_SETUP__'), '/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /NOAPPSTART', '', SW_HIDE, ewWaitUntilTerminated, Code) or (Code <> 0) then
      RaiseException('Server installation did not complete. Existing components and library data were preserved. Retry the component installer.');
    if not Exec(ExpandConstant('{tmp}\__DESKTOP_SETUP__'), '/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /NOAPPSTART', '', SW_HIDE, ewWaitUntilTerminated, Code) or (Code <> 0) then
      RaiseException('Server is installed, but Desktop installation did not complete. Retry the Desktop installer.');
    if not WizardSilent then begin
      LaunchComponent('__SERVER_APP_ID__', 'curated.exe', '-mode tray -autostart');
      LaunchComponent('__DESKTOP_APP_ID__', 'Curated.exe', '');
    end;
  end;
end;
