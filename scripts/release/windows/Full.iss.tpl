[Setup]
AppName=Curated Full
AppVersion=__VERSION__
AppPublisher=Curated
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
CreateAppDir=no
Uninstallable=no
DisableWelcomePage=no
OutputDir=__OUTPUT__
OutputBaseFilename=__BASENAME__
Compression=lzma2
SolidCompression=yes
WizardStyle=modern

[Files]
Source: "__SERVER_INSTALLER__"; DestName: "server-setup.exe"; Flags: dontcopy
Source: "__DESKTOP_INSTALLER__"; DestName: "desktop-setup.exe"; Flags: dontcopy

[Code]
var
  Failure: String;

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
