package symbols

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/ChristianBoehm/go-opsiscript-lsp/internal/ast"
)

type Builtin struct {
	Label         string
	Detail        string
	Documentation string
}

var commands = map[string]Builtin{
	"add":                       builtin("Add", "registry command", "Add values in a `Registry_*` section."),
	"autoactivitydisplay":       builtin("AutoActivityDisplay", "configuration command", "Toggle the activity window display."),
	"changedirectory":           builtin("ChangeDirectory", "filesystem command", "Change the current working directory for later commands."),
	"checktargetpath":           builtin("CheckTargetPath", "filesystem command", "Validate that a destination path is usable before file operations."),
	"chmod":                     builtin("chmod", "filesystem command", "Change file mode bits."),
	"comment":                   builtin("comment", "logging command", "Write an informational log entry."),
	"copy":                      builtin("copy", "filesystem command", "Copy files and directories."),
	"defstringlist":             builtin("DefStringList", "declaration command", "Declare a string list variable."),
	"defvar":                    builtin("DefVar", "declaration command", "Declare a scalar variable."),
	"delete":                    builtin("delete", "filesystem command", "Delete files or directories."),
	"deletekey":                 builtin("DeleteKey", "registry command", "Delete a registry key."),
	"del":                       builtin("del", "filesystem command", "Delete files or directories."),
	"delsec":                    builtin("delsec", "registry command", "Delete a section entry."),
	"encoding":                  builtin("encoding", "configuration command", "Set the file encoding used by the script."),
	"escapestring":              builtin("EscapeString", "string helper", "Escape a string for later processing."),
	"executesection":            builtin("executeSection", "control command", "Execute another Opsiscript section."),
	"exitonerror":               builtin("ExitOnError", "configuration command", "Control whether the script aborts on command errors."),
	"fatalonruntimeerror":       builtin("FatalOnRuntimeError", "configuration command", "Treat runtime errors as fatal."),
	"fatalonsyntaxerror":        builtin("FatalOnSyntaxError", "configuration command", "Treat syntax errors as fatal."),
	"forceloginappendmode":      builtin("forceLogInAppendMode", "configuration command", "Append to the opsi log instead of truncating it."),
	"hardlink":                  builtin("hardlink", "filesystem command", "Create a hard link."),
	"iconizewinst":              builtin("IconizeWinst", "window command", "Minimize the interpreter window."),
	"importlib":                 builtin("importlib", "include command", "Load another Opsiscript library file."),
	"include_append":            builtin("include_append", "include command", "Append another script after the current content."),
	"include_insert":            builtin("include_insert", "include command", "Insert another script at the current location."),
	"includelog":                builtin("includelog", "logging command", "Attach another log file to the current log output."),
	"isfatalerror":              builtin("isFatalError", "error command", "Mark the current run as fatal with a message."),
	"loglevel":                  builtin("LogLevel", "legacy configuration command", "Deprecated logging configuration alias kept for older scripts."),
	"logerror":                  builtin("LogError", "logging command", "Write an error log entry."),
	"logwarning":                builtin("LogWarning", "logging command", "Write a warning log entry."),
	"maximizewinst":             builtin("MaximizeWinst", "window command", "Maximize the interpreter window."),
	"message":                   builtin("Message", "ui command", "Show a user-facing status message."),
	"move":                      builtin("move", "filesystem command", "Move or rename files."),
	"normalizewinst":            builtin("NormalizeWinst", "window command", "Restore the interpreter window to normal state."),
	"pause":                     builtin("Pause", "control command", "Pause script execution."),
	"reloadproductlist":         builtin("reloadProductList", "opsi command", "Reload the opsi product list."),
	"rename":                    builtin("rename", "filesystem command", "Rename a file or directory."),
	"requiredopsiscriptversion": builtin("requiredOpsiscriptVersion", "configuration command", "Declare the minimum supported interpreter version."),
	"requiredwinstversion":      builtin("requiredWinstVersion", "configuration command", "Declare the minimum supported winst version."),
	"restorewinst":              builtin("RestoreWinst", "window command", "Restore the interpreter window."),
	"scripterrormessages":       builtin("ScriptErrorMessages", "configuration command", "Control script error message display."),
	"set":                       builtin("Set", "assignment command", "Assign a value to a variable."),
	"setconfidential":           builtin("SetConfidential", "logging command", "Mark later log output as confidential."),
	"setloglevel":               builtin("SetLogLevel", "logging command", "Set the interpreter log verbosity."),
	"setskindirectory":          builtin("SetSkinDirectory", "ui command", "Set the directory used for skin assets."),
	"showbitmap":                builtin("ShowBitmap", "ui command", "Show a bitmap alongside status output."),
	"sleepseconds":              builtin("SleepSeconds", "control command", "Sleep for the given number of seconds."),
	"sourcepath":                builtin("SourcePath", "filesystem command", "Adjust the source path for file operations."),
	"stayontop":                 builtin("StayOnTop", "ui command", "Keep the opsi-script window on top."),
	"stop":                      builtin("Stop", "control command", "Stop the current script."),
	"sub":                       builtin("sub", "section command", "Execute a `Sub_*` section."),
	"symlink":                   builtin("symlink", "filesystem command", "Create a symbolic link."),
	"tracemode":                 builtin("TraceMode", "debug command", "Show every log entry in a confirmation dialog."),
	"unzipfile":                 builtin("unzipfile", "archive command", "Extract a zip archive."),
	"zipfile":                   builtin("zipfile", "archive command", "Create a zip archive."),
}

var functions = map[string]Builtin{
	"addlisttolist":                  builtin("addListToList", "list function", "Append all items from one string list to another."),
	"addtolist":                      builtin("addToList", "list function", "Append a value to a string list."),
	"backupposipoc":                  builtin("backupOpsiPoc", "uib_backend library function", "Write localboot productOnClient objects to a JSON backup file."),
	"bootnexttopepartition":          builtin("bootnextToPePartition", "uib_bootutils library function", "Configure the next boot to start the PE partition."),
	"bootnexttouefilabel":            builtin("bootNextToUefiLabel", "uib_bootutils library function", "Set the UEFI boot manager to boot next to the given label."),
	"bootnexttowinlabel":             builtin("bootNextToWinLabel", "uib_bootutils library function", "Set the Windows boot manager to boot next to the given label."),
	"check_module_activation":        builtin("check_module_activation", "uib_backend library function", "Verify that the given opsi module is activated."),
	"contains":                       builtin("contains", "string function", "Return true if the first string contains the second string."),
	"count":                          builtin("count", "list function", "Return the number of items in a string list."),
	"delfromwindowsbootmanager":      builtin("delFromWindowsBootmanager", "uib_bootutils library function", "Delete the given entry from the Windows boot manager."),
	"delopsipoc":                     builtin("delOpsiPoc", "uib_backend library function", "Delete localboot productOnClient objects except the protected list."),
	"deluefibootnext":                builtin("delUefiBootNext", "uib_bootutils library function", "Remove the current UEFI boot-next entry."),
	"deluefibootnextbylabel":         builtin("delUefiBootNextByLabel", "uib_bootutils library function", "Remove the UEFI boot-next entry if it has the given label."),
	"directoryexists":                builtin("directoryExists", "filesystem function", "Check whether a directory exists."),
	"enablepepartition":              builtin("enablePEPartition", "uib_bootutils library function", "Make the given PE partition visible and bootable."),
	"envvar":                         builtin("EnvVar", "environment function", "Read a system environment variable."),
	"escaperegexmetachars":           builtin("escapeRegexMetaChars", "uib_bootutils library function", "Escape regex meta characters in a string."),
	"extractfileextension":           builtin("ExtractFileExtension", "path function", "Return the extension part of a path."),
	"extractfilename":                builtin("ExtractFileName", "path function", "Return the file name part of a path."),
	"extractfilepath":                builtin("ExtractFilePath", "path function", "Return the directory part of a path."),
	"fileexists":                     builtin("FileExists", "filesystem function", "Check whether a file exists."),
	"fileexists32":                   builtin("FileExists32", "filesystem function", "Check whether a file exists using the 32-bit view."),
	"fileexists64":                   builtin("FileExists64", "filesystem function", "Check whether a file exists using the 64-bit view."),
	"fileexistssysnative":            builtin("FileExistsSysNative", "filesystem function", "Check whether a file exists using the native system view."),
	"fileorfolderexists":             builtin("FileOrFolderExists", "filesystem function", "Check whether a file or folder exists."),
	"fileissymlink":                  builtin("fileIsSymlink", "filesystem function", "Check whether a path is a symbolic link."),
	"forcepathdelims":                builtin("forcePathDelims", "path function", "Normalize path delimiters to the current operating system."),
	"getconfidentialproductproperty": builtin("GetConfidentialProductProperty", "product property function", "Read a product property as confidential data."),
	"getdiskuuid":                    builtin("getDiskUuid", "uib_bootutils library function", "Determine the UUID of the given disk."),
	"gethostsaddr":                   builtin("GetHostsAddr", "network function", "Return the IP address for a host from the local hosts file."),
	"gethostsname":                   builtin("GetHostsName", "network function", "Return the host name for an IP address from the local hosts file."),
	"getindexfromlistbycontaining":   builtin("getIndexFromListByContaining", "list function", "Return the index of the first list item containing the search string."),
	"getini":                         builtin("GetIni", "ini function", "Deprecated ini-file lookup helper."),
	"getinstallablelocalbootproductswithversion": builtin("getInstallableLocalbootProductsWithVersion", "uib_backend library function", "Return installable localboot products with versions for the depot."),
	"getinstalledlocalbootproducts":              builtin("getInstalledLocalbootProducts", "uib_backend library function", "Return installed localboot product IDs for the client."),
	"getinstalledlocalbootproductswithversion":   builtin("getInstalledLocalbootProductsWithVersion", "uib_backend library function", "Return installed localboot products with versions."),
	"getlastexitcode":                            builtin("getLastExitCode", "process function", "Return the exit code of the last external process."),
	"getlinuxdistrotype":                         builtin("getLinuxDistroType", "system function", "Return a coarse Linux distribution family."),
	"getmacosversioninfo":                        builtin("getMacosVersionInfo", "system function", "Return the macOS version information."),
	"getmsversioninfo":                           builtin("GetMsVersionInfo", "system function", "Return the Windows version reported by the API."),
	"getmsversionname":                           builtin("GetMsVersionName", "system function", "Return the Windows marketing version."),
	"getntversion":                               builtin("GetNtVersion", "system function", "Deprecated Windows version helper."),
	"getos":                                      builtin("GetOS", "system function", "Return the current operating system identifier."),
	"getosarchitecture":                          builtin("getOSArchitecture", "system function", "Return the processor architecture of the operating system."),
	"getoutstreamfromsection":                    builtin("getOutStreamFromSection", "section function", "Capture stdout from a section call."),
	"getproductproperty":                         builtin("GetProductProperty", "product property function", "Read a product property value."),
	"getregistrykeylist":                         builtin("getRegistryKeyList", "registry function", "List subkeys from a registry key using an explicit access mode."),
	"getregistrykeylist32":                       builtin("getRegistryKeyList32", "registry function", "List subkeys from the 32-bit registry view."),
	"getregistrykeylist64":                       builtin("getRegistryKeyList64", "registry function", "List subkeys from the 64-bit registry view."),
	"getregistrykeylistsysnative":                builtin("getRegistryKeyListSysNative", "registry function", "List subkeys from the native registry view."),
	"getregistrystringvalue":                     builtin("GetRegistryStringValue", "registry function", "Deprecated registry string lookup helper."),
	"getregistrystringvalue32":                   builtin("GetRegistryStringValue32", "registry function", "Read a registry string using the 32-bit registry view."),
	"getregistrystringvalue64":                   builtin("GetRegistryStringValue64", "registry function", "Read a registry string using the 64-bit registry view."),
	"getregistrystringvaluesysnative":            builtin("GetRegistryStringValueSysNative", "registry function", "Read a registry string using the native registry view."),
	"getregistryvalue":                           builtin("getRegistryValue", "registry function", "Read a registry value, optionally with an explicit access mode."),
	"getregexmatchlist":                          builtin("getRegexMatchList", "regex function", "Filter a list by a regular expression."),
	"getreturnlistfromsection":                   builtin("getReturnListFromSection", "section function", "Capture return values from XMLPatch or opsiServiceCall sections."),
	"getsystemtype":                              builtin("GetSystemType", "system function", "Return whether the system is x86 or 64 bit."),
	"getuefibcdbootguid":                         builtin("getUefiBcdbootGuid", "uib_bootutils library function", "Get the UEFI boot entry GUID for the given label."),
	"getuefibootorder":                           builtin("getUefiBootOrder", "uib_bootutils library function", "Return the UEFI boot order."),
	"getusersid":                                 builtin("GetUserSID", "user function", "Return the SID for the given Windows user."),
	"getusercontext":                             builtin("GetUsercontext", "user function", "Return the /usercontext value passed to opsi-script."),
	"getvaluefrominifile":                        builtin("GetValueFromInifile", "ini function", "Read a value from an ini file, with optional encoding."),
	"getwinbcdbootguid":                          builtin("getWinBcdbootGuid", "uib_bootutils library function", "Get the Windows boot entry GUID for the given label."),
	"handle_setup_after_property":                builtin("handle_setup_after_property", "uib_backend library function", "Resolve a product-property list and set setup actions accordingly."),
	"hasminimumspace":                            builtin("HasMinimumSpace", "filesystem function", "Check whether a drive has enough free space."),
	"hexstrtodecstr":                             builtin("HexStrToDecStr", "conversion function", "Convert a hexadecimal string to a decimal string."),
	"inivar":                                     builtin("IniVar", "product property function", "Deprecated alias for product property lookup."),
	"ismsiexitcodefatal":                         builtin("isMsiExitcodeFatal", "installer function", "Classify an MSI exit code as fatal or not."),
	"isnumber":                                   builtin("isNumber", "type function", "Check whether a string contains a numeric value."),
	"isregexmatch":                               builtin("isRegexMatch", "regex function", "Check whether a string matches a regex."),
	"lower":                                      builtin("lower", "string function", "Return the lowercase version of a string."),
	"paramstr":                                   builtin("ParamStr", "environment function", "Return the /parameter string from the opsi-script command line."),
	"powershellcall":                             builtin("powershellCall", "process function", "Run PowerShell and optionally control the registry/filesystem access view."),
	"regkeyexists":                               builtin("RegKeyExists", "registry function", "Check whether a registry key exists."),
	"regstring":                                  builtin("RegString", "registry function", "Escape backslashes for registry-string format."),
	"regvarexists":                               builtin("RegVarExists", "registry function", "Check whether a registry value exists."),
	"removefromlistbymatch":                      builtin("removeFromListByMatch", "list function", "Remove matching values from a string list."),
	"resolvesymlink":                             builtin("resolveSymlink", "path function", "Resolve a symbolic link recursively."),
	"restoreopsipoc":                             builtin("restoreOpsiPoc", "uib_backend library function", "Restore productOnClient objects from a JSON backup file."),
	"setproductstosetup":                         builtin("setProductsToSetup", "uib_backend library function", "Set setup actions for the given opsi products."),
	"setproductstouninstall":                     builtin("setProductsToUninstall", "uib_backend library function", "Set uninstall actions for the given opsi products."),
	"setuefilabeltofirstbootorder":               builtin("setUefiLabelToFirstBootOrder", "uib_bootutils library function", "Move the given UEFI boot label to the first position."),
	"splitstring":                                builtin("splitString", "string function", "Split a string into a string list."),
	"strlength":                                  builtin("strLength", "string function", "Return the number of characters in a string."),
	"strpos":                                     builtin("strPos", "string function", "Return the first position of a substring in a string."),
	"strpart":                                    builtin("strPart", "string function", "Return a substring by position and length."),
	"stringreplace":                              builtin("stringReplace", "string function", "Replace text within a string."),
	"stringsplit":                                builtin("StringSplit", "string function", "Deprecated split helper, prefer splitString/takeString."),
	"stringtobool":                               builtin("stringToBool", "type function", "Convert an opsi boolean string to a boolean value."),
	"takefirststringcontaining":                  builtin("takeFirstStringContaining", "list function", "Return the first list item containing the search string."),
	"takestring":                                 builtin("TakeString", "list function", "Read an item by index from a string list."),
	"trim":                                       builtin("trim", "string function", "Trim leading and trailing whitespace."),
	"unquote":                                    builtin("unquote", "string function", "Remove matching quote characters from a string."),
	"unquote2":                                   builtin("unquote2", "string function", "Remove matching start/end quote characters from a string."),
	"upper":                                      builtin("upper", "string function", "Return the uppercase version of a string."),
	"which":                                      builtin("which", "path function", "Locate a command in the current search path."),
}

var constants = map[string]Builtin{
	"allusersprofiledir":       builtin("%AllUsersProfileDir%", "builtin constant", "The common public profile directory."),
	"appdatadir":               builtin("%AppdataDir%", "builtin constant", "The current user's roaming appdata directory."),
	"commonappdatadir":         builtin("%CommonAppDataDir%", "builtin constant", "The common application data directory."),
	"commondesktopdir":         builtin("%CommonDesktopDir%", "builtin constant", "The common desktop directory."),
	"commonprofiledir":         builtin("%CommonProfileDir%", "builtin constant", "Alias for the public profile directory."),
	"commonprogramsdir":        builtin("%CommonProgramsDir%", "builtin constant", "The common Start Menu Programs directory."),
	"commonstartmenudir":       builtin("%CommonStartMenuDir%", "builtin constant", "The common Start Menu directory."),
	"commonstartmenupath":      builtin("%CommonStartMenuPath%", "builtin constant", "The common Start Menu directory."),
	"commonstartupdir":         builtin("%CommonStartupDir%", "builtin constant", "The common Startup directory."),
	"currentappdatadir":        builtin("%CurrentAppdataDir%", "builtin constant", "The current user's roaming appdata directory."),
	"currentdesktopdir":        builtin("%CurrentDesktopDir%", "builtin constant", "The current user's desktop directory."),
	"currentprofiledir":        builtin("%CurrentProfileDir%", "builtin constant", "The current user's profile directory."),
	"currentprogramsdir":       builtin("%CurrentProgramsDir%", "builtin constant", "The current user's Start Menu Programs directory."),
	"currentsendtodir":         builtin("%CurrentSendToDir%", "builtin constant", "The current user's SendTo directory."),
	"currentstartmenudir":      builtin("%CurrentStartMenuDir%", "builtin constant", "The current user's Start Menu directory."),
	"currentstartupdir":        builtin("%CurrentStartupDir%", "builtin constant", "The current user's Startup directory."),
	"defaultuserprofiledir":    builtin("%DefaultUserProfileDir%", "builtin constant", "The default user profile directory."),
	"fqdn":                     builtin("%FQDN%", "builtin constant", "The fully qualified domain name of the machine in network context."),
	"host":                     builtin("%Host%", "builtin constant", "Legacy host environment value."),
	"hostid":                   builtin("%HostID%", "builtin constant", "The client FQDN in opsi service context."),
	"installingprodname":       builtin("%installingProdName%", "builtin constant", "The product ID currently being installed by the opsi service."),
	"installingprodversion":    builtin("%installingProdVersion%", "builtin constant", "The product and package version currently being installed."),
	"installingproduct":        builtin("%installingProduct%", "builtin constant", "Legacy alias for the current installing product."),
	"ipaddress":                builtin("%IPAddress%", "builtin constant", "Legacy IP address constant."),
	"ipname":                   builtin("%IPName%", "builtin constant", "The DNS name of the current machine."),
	"logfile":                  builtin("%Logfile%", "builtin constant", "The currently active opsi-script log file."),
	"opsiapplog":               builtin("%opsiapplog%", "builtin constant", "The directory for user-context application logs."),
	"opsidata":                 builtin("%opsidata%", "builtin constant", "The directory for opsi data files."),
	"opsidepotid":              builtin("%opsiDepotId%", "builtin constant", "The current depot server identifier."),
	"opsilogdir":               builtin("%opsiLogDir%", "builtin constant", "The current opsi log directory."),
	"opsiscriptdir":            builtin("%OpsiScriptDir%", "builtin constant", "The installation directory of the running opsi-script executable."),
	"opsiscripthelperpath":     builtin("%opsiScriptHelperPath%", "builtin constant", "The helper path for opsi-script support tools and libraries."),
	"opsiscriptprocname":       builtin("%opsiscriptProcname%", "builtin constant", "The current opsi-script process name."),
	"opsiscriptversion":        builtin("%OpsiscriptVersion%", "builtin constant", "The version string of the running opsi-script."),
	"opsiserver":               builtin("%opsiServer%", "builtin constant", "The opsi server derived from the service URL."),
	"opsiservicepassword":      builtin("%opsiservicePassword%", "builtin constant", "The password used for the current opsi-service connection."),
	"opsiserviceuser":          builtin("%opsiserviceUser%", "builtin constant", "The user ID used for the current opsi-service connection."),
	"opsitmpdir":               builtin("%opsiTmpDir%", "builtin constant", "The directory intended for temporary files."),
	"opsiusertmpdir":           builtin("%opsiUserTmpDir%", "builtin constant", "The per-user temporary directory that does not require admin rights."),
	"pcname":                   builtin("%PCName%", "builtin constant", "The NetBIOS or computer name of the current machine."),
	"profiledir":               builtin("%ProfileDir%", "builtin constant", "The base profile directory."),
	"programfiles32dir":        builtin("%ProgramFiles32Dir%", "builtin constant", "The 32-bit Program Files directory."),
	"programfiles64dir":        builtin("%ProgramFiles64Dir%", "builtin constant", "The 64-bit Program Files directory."),
	"programfilesdir":          builtin("%ProgramFilesDir%", "builtin constant", "The default Program Files directory, historically the 32-bit view on 64-bit systems."),
	"programfilessysnativedir": builtin("%ProgramFilesSysnativeDir%", "builtin constant", "The architecture-native Program Files directory."),
	"realscriptpath":           builtin("%RealScriptPath%", "builtin constant", "The resolved script directory when the script is reached through a symlink."),
	"scriptdir":                builtin("%ScriptDir%", "builtin constant", "Alias for the current script directory."),
	"scriptdrive":              builtin("%ScriptDrive%", "builtin constant", "The drive containing the current script."),
	"scriptpath":               builtin("%ScriptPath%", "builtin constant", "The directory containing the running script."),
	"system":                   builtin("%System%", "builtin constant", "The Windows system32 directory."),
	"systemdrive":              builtin("%SystemDrive%", "builtin constant", "The active system drive."),
	"systemroot":               builtin("%Systemroot%", "builtin constant", "The Windows root directory."),
	"userprofiledir":           builtin("%UserProfileDir%", "builtin constant", "The active user profile directory when iterating user profiles."),
	"username":                 builtin("%Username%", "builtin constant", "The currently logged-in user name."),
	"winstdir":                 builtin("%WinstDir%", "builtin constant", "Legacy alias for %OpsiScriptDir%."),
	"winstversion":             builtin("%WinstVersion%", "builtin constant", "Legacy alias for %OpsiscriptVersion%."),
}

var sectionPrefixes = map[string]string{
	"sub":             "Sub section",
	"files":           "Files section",
	"winbatch":        "WinBatch section",
	"shellinanicon":   "ShellInAnIcon section",
	"shellscript":     "ShellScript section",
	"execwith":        "ExecWith section",
	"execpython":      "ExecPython section",
	"registry":        "Registry section",
	"patchhosts":      "PatchHosts section",
	"xml2":            "XML2 section",
	"xmlpatch":        "XMLPatch section",
	"patchtextfile":   "PatchTextFile section",
	"patchinifile":    "PatchIniFile section",
	"ldapsearch":      "LDAPsearch section",
	"linkfolder":      "LinkFolder section",
	"opsiservicecall": "OpsiServiceCall section",
}

type Index struct {
	Sections  map[string][]ast.Section
	Variables map[string]map[string][]ast.VariableDecl
	Functions map[string][]ast.FunctionDecl
}

func BuildIndex(doc *ast.Document) *Index {
	return BuildIndexDocuments(doc)
}

func BuildIndexDocuments(documents ...*ast.Document) *Index {
	index := &Index{
		Sections:  map[string][]ast.Section{},
		Variables: map[string]map[string][]ast.VariableDecl{},
		Functions: map[string][]ast.FunctionDecl{},
	}

	for _, doc := range documents {
		if doc == nil {
			continue
		}

		for _, section := range doc.Sections {
			index.Sections[section.NormalizedName] = append(index.Sections[section.NormalizedName], section)
		}

		for _, variable := range doc.Variables {
			scope := NormalizeName(variable.Scope)
			if index.Variables[scope] == nil {
				index.Variables[scope] = map[string][]ast.VariableDecl{}
			}
			index.Variables[scope][variable.NormalizedName] = append(index.Variables[scope][variable.NormalizedName], variable)
		}

		for _, function := range doc.Functions {
			index.Functions[function.NormalizedName] = append(index.Functions[function.NormalizedName], function)
		}
	}

	return index
}

func NormalizeName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func LookupCommand(name string) (Builtin, bool) {
	builtin, ok := commands[NormalizeName(name)]
	return builtin, ok
}

func LookupFunction(name string) (Builtin, bool) {
	builtin, ok := functions[NormalizeName(name)]
	return builtin, ok
}

func LookupConstant(name string) (Builtin, bool) {
	trimmed := strings.Trim(name, "%")
	builtin, ok := constants[NormalizeName(trimmed)]
	return builtin, ok
}

func TypedSectionKind(name string) (string, string, bool) {
	normalized := NormalizeName(name)
	for _, prefix := range SectionPrefixes() {
		detail := sectionPrefixes[prefix]
		prefixWithUnderscore := prefix + "_"
		if strings.HasPrefix(normalized, prefixWithUnderscore) {
			return prefix, detail, true
		}
	}

	return "", "", false
}

func ResolveVariable(index *Index, name, scope string) *ast.VariableDecl {
	if index == nil {
		return nil
	}

	normalizedName := NormalizeName(name)
	normalizedScope := NormalizeName(scope)

	if scopedDecls, ok := index.Variables[normalizedScope]; ok {
		if declarations := scopedDecls[normalizedName]; len(declarations) > 0 {
			decl := declarations[0]
			return &decl
		}
	}

	if globalDecls, ok := index.Variables[""]; ok {
		if declarations := globalDecls[normalizedName]; len(declarations) > 0 {
			decl := declarations[0]
			return &decl
		}
	}

	return nil
}

func ResolveSection(index *Index, name string) *ast.Section {
	if index == nil {
		return nil
	}

	declarations := index.Sections[NormalizeName(name)]
	if len(declarations) == 0 {
		return nil
	}

	decl := declarations[0]
	return &decl
}

func ResolveFunction(index *Index, name string) *ast.FunctionDecl {
	if index == nil {
		return nil
	}

	declarations := index.Functions[NormalizeName(name)]
	if len(declarations) == 0 {
		return nil
	}

	decl := declarations[0]
	return &decl
}

func CommandNames() []string {
	return sortedKeys(commands)
}

func FunctionNames() []string {
	return sortedKeys(functions)
}

func ConstantNames() []string {
	names := make([]string, 0, len(constants))
	for key, builtin := range constants {
		if builtin.Label != "" {
			names = append(names, builtin.Label)
			continue
		}
		names = append(names, fmt.Sprintf("%%%s%%", key))
	}
	sort.Strings(names)
	return names
}

func SectionPrefixes() []string {
	prefixes := make([]string, 0, len(sectionPrefixes))
	for prefix := range sectionPrefixes {
		prefixes = append(prefixes, prefix)
	}
	sort.Slice(prefixes, func(i, j int) bool {
		if len(prefixes[i]) == len(prefixes[j]) {
			return prefixes[i] < prefixes[j]
		}
		return len(prefixes[i]) > len(prefixes[j])
	})
	return prefixes
}

func SectionCallPrefixesPattern() string {
	prefixes := SectionPrefixes()
	escaped := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		escaped = append(escaped, regexp.QuoteMeta(prefix))
	}
	return strings.Join(escaped, "|")
}

func builtin(label, detail, documentation string) Builtin {
	return Builtin{
		Label:         label,
		Detail:        detail,
		Documentation: documentation,
	}
}

func sortedKeys[K ~string](items map[K]Builtin) []string {
	keys := make([]string, 0, len(items))
	for _, builtin := range items {
		keys = append(keys, builtin.Label)
	}
	sort.Strings(keys)
	return keys
}
