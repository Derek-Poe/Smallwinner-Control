"""Supplement APK compilation/tests with explicit public-demo manifest contracts."""
from pathlib import Path
import re
import xml.etree.ElementTree as ET

ROOT = Path(__file__).resolve().parent
ANDROID = "{http://schemas.android.com/apk/res/android}"
manifest = ET.parse(ROOT / "app/src/main/AndroidManifest.xml").getroot()
assert [node.get(ANDROID + "name") for node in manifest.findall("uses-permission")] == ["android.permission.INTERNET"]
app = manifest.find("application")
assert app.get(ANDROID + "allowBackup") == "false"
assert app.get(ANDROID + "dataExtractionRules") == "@xml/data_extraction_rules"
extraction = ET.parse(ROOT / "app/src/main/res/xml/data_extraction_rules.xml").getroot()
for transport in ("cloud-backup", "device-transfer"):
    exclusions = extraction.find(transport).findall("exclude")
    assert {item.get("domain") for item in exclusions} == {"root", "file", "database", "sharedpref", "external", "device_root", "device_file", "device_database", "device_sharedpref"}
    assert all(item.get("path") == "." for item in exclusions)
assert app.get(ANDROID + "usesCleartextTraffic") == "false"
assert app.get(ANDROID + "debuggable") not in {"true"}
assert len(app.findall("activity")) == 1
assert not any(app.findall(kind) for kind in ("service", "receiver", "provider", "activity-alias"))
activity = app.find("activity")
assert activity.get(ANDROID + "name") == ".MainActivity"
assert activity.get(ANDROID + "exported") == "true"
assert [item.get(ANDROID + "name") for item in activity.findall("intent-filter/category")] == ["android.intent.category.LAUNCHER"]
assert not manifest.findall("queries/package")
config = ET.parse(ROOT / "app/src/main/res/xml/network_security_config.xml").getroot()
assert config.find("base-config").get("cleartextTrafficPermitted") == "false"
domains = config.findall("domain-config/domain")
assert len(domains) == 1 and domains[0].text == "127.0.0.1" and domains[0].get("includeSubdomains") == "false"
sources = "\n".join(path.read_text(encoding="utf-8") for path in (ROOT / "app/src/main/java").rglob("*.java"))
for forbidden in (r"@JavascriptInterface", r"addJavascriptInterface\s*\(", r"addWebMessageListener\s*\(",
                  r"ProcessBuilder", r"Runtime\.getRuntime", r"startService\s*\(",
                  r"getSharedPreferences\s*\(", r"setAllowFileAccess\(true\)",
                  r"setAllowContentAccess\(true\)", r"setWebContentsDebuggingEnabled\(true\)"):
    assert not re.search(forbidden, sources), f"Forbidden capability pattern: {forbidden}"
assert "setDomStorageEnabled(false)" in sources
assert "setAcceptCookie(false)" in sources
assert "setJavaScriptEnabled(false)" in sources
assert "setComponent(new ComponentName(choice.packageName, choice.activityName))" in sources
print("Android source contracts passed: narrow permissions, loopback cleartext, native-only catalog, no bridge/process/service/token storage.")
