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
procedure CurStepChanged(CurStep: TSetupStep);
var Code: Integer;
begin
  if CurStep = ssPostInstall then begin
    if not Exec(ExpandConstant('{tmp}\__SERVER_SETUP__'), '/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /NOAPPSTART', '', SW_HIDE, ewWaitUntilTerminated, Code) or (Code <> 0) then
      RaiseException('Server installation did not complete. Existing components and library data were preserved. Retry the component installer.');
    if not Exec(ExpandConstant('{tmp}\__DESKTOP_SETUP__'), '/VERYSILENT /SUPPRESSMSGBOXES /NORESTART /NOAPPSTART', '', SW_HIDE, ewWaitUntilTerminated, Code) or (Code <> 0) then
      RaiseException('Server is installed, but Desktop installation did not complete. Retry the Desktop installer.');
    if not WizardSilent then begin
      Exec(ExpandConstant('{localappdata}\Programs\Curated\Server\curated.exe'), '-mode tray -autostart', '', SW_SHOW, ewNoWait, Code);
      Exec(ExpandConstant('{localappdata}\Programs\Curated\Desktop\Curated.exe'), '', '', SW_SHOW, ewNoWait, Code);
    end;
  end;
end;
