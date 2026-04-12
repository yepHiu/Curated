#define MyAppName "Curated"
#define MyAppVersion "__APP_VERSION__"
#define MyAppPublisher "Curated"
#define MyAppExeName "curated.exe"
#define MyAppSourceDir "__APP_DIR__"
#define MyOutputDir "__OUTPUT_DIR__"
#define MySetupBaseName "__SETUP_BASENAME__"

[Setup]
AppId={{8C9E9E66-7058-4D09-9F9A-8AFD060A7E1B}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={autopf}\Curated
DefaultGroupName=Curated
DisableProgramGroupPage=yes
OutputDir={#MyOutputDir}
OutputBaseFilename={#MySetupBaseName}
Compression=lzma
SolidCompression=yes
WizardStyle=modern
SetupIconFile={#MyAppSourceDir}\curated.ico
UninstallDisplayIcon={app}\curated.ico
CloseApplications=yes
RestartApplications=no
AppMutex=Local\Curated.Tray.Singleton
CloseApplicationsFilter=curated.exe

[Languages]
Name: "english"; MessagesFile: "compiler:Default.isl"

[Tasks]
Name: "desktopicon"; Description: "{cm:CreateDesktopIcon}"; GroupDescription: "{cm:AdditionalIcons}"; Flags: unchecked

[Files]
Source: "{#MyAppSourceDir}\*"; DestDir: "{app}"; Flags: recursesubdirs createallsubdirs ignoreversion

[Icons]
Name: "{autoprograms}\Curated"; Filename: "{app}\{#MyAppExeName}"; IconFilename: "{app}\curated.ico"
Name: "{autodesktop}\Curated"; Filename: "{app}\{#MyAppExeName}"; Tasks: desktopicon; IconFilename: "{app}\curated.ico"

[Run]
Filename: "{app}\{#MyAppExeName}"; Description: "Launch Curated"; Flags: nowait postinstall skipifsilent
