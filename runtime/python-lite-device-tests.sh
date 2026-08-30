#!/usr/bin/env bash
# PocketClaw Python Lite — Phase A physical device harness.
#
# Installs a throwaway diagnostic APK and runs the interpreter from
# nativeLibraryDir, i.e. under the same execution model as the Managed Runtime.
# It touches nothing belonging to PocketClaw and installs no product code.
#
# Remote command transport
# ------------------------
# `adb shell a b c` does NOT preserve the local shell's argument boundaries: adb
# joins its arguments with spaces and hands the result to the device shell as a
# single string. So `adb shell sh -c "mkdir -p /x"` arrives as
# `sh -c mkdir -p /x`, which runs `mkdir` with NO arguments and assigns "-p" to
# $0. Every command in this harness is therefore base64-encoded locally and
# decoded on the device, so the string adb transports contains nothing that a
# shell could re-split.
#
# Set PCPROBE_VERBOSE=1 for stage-by-stage diagnostics. Verbose mode prints the
# remote command *structure*, never Python source and never file contents.
set -u

PKG=com.pocketclaw.pythonprobe
APK="$(cd "$(dirname "$0")" && pwd)/pocketclaw-python-probe.apk"
ADB="${ADB:-adb}"
VERBOSE="${PCPROBE_VERBOSE:-0}"

pass=0; fail=0
ok(){ printf '  PASS  %s\n' "$1"; pass=$((pass+1)); }
no(){ printf '  FAIL  %s%s\n' "$1" "${2:+  -- $2}"; fail=$((fail+1)); }
note(){ printf '        %s\n' "$1"; }

RC=0; OUT=""; ERR=""

# remote <stage> <script>  — runs <script> under the app's uid via run-as.
# Sets RC (the script's exit status), OUT (its stdout) and ERR (its stderr).
remote() {
  local stage="$1" script="$2" b64 raw
  # Markers, because adb merges the remote streams into one.
  local wrapped="{
$script
} >__pcp_out 2>__pcp_err
__pcp_rc=\$?
echo \"__RC=\$__pcp_rc\"
echo __OUT__
cat __pcp_out
echo __ERR__
cat __pcp_err
rm -f __pcp_out __pcp_err"
  b64=$(printf '%s' "$wrapped" | base64 | tr -d '\n')
  raw=$(timeout 180 "$ADB" shell "echo $b64 | base64 -d | run-as $PKG sh" 2>&1 | tr -d '\r')
  RC=$(printf '%s\n' "$raw" | sed -n 's/^__RC=\([0-9]*\)$/\1/p' | head -1)
  [ -n "$RC" ] || RC=255
  OUT=$(printf '%s\n' "$raw" | sed -n '/^__OUT__$/,/^__ERR__$/p' | sed '1d;$d')
  ERR=$(printf '%s\n' "$raw" | sed -n '/^__ERR__$/,$p' | sed '1d')
  if [ "$VERBOSE" = "1" ]; then
    printf '  [dbg] stage=%s transport=base64->run-as-sh rc=%s stdout_bytes=%s stderr_bytes=%s\n' \
      "$stage" "$RC" "${#OUT}" "${#ERR}"
  fi
}

# shell_uid <script> — same transport, but as the adb shell user, not the app.
# Used only as a control, to tell "the binary is broken" apart from
# "the app sandbox refused it".
shell_uid() {
  local b64; b64=$(printf '%s' "$1" | base64 | tr -d '\n')
  timeout 90 "$ADB" shell "echo $b64 | base64 -d | sh" 2>&1 | tr -d '\r'
}

"$ADB" get-state >/dev/null 2>&1 || { echo "no device (adb get-state failed)"; exit 1; }
MODEL=$("$ADB" shell getprop ro.product.model | tr -d '\r')
REL=$("$ADB" shell getprop ro.build.version.release | tr -d '\r')
SDK=$("$ADB" shell getprop ro.build.version.sdk | tr -d '\r')
ABI=$("$ADB" shell getprop ro.product.cpu.abi | tr -d '\r')
PAGE=$("$ADB" shell getconf PAGE_SIZE | tr -d '\r')
echo "device: $MODEL / Android $REL / API $SDK / $ABI / page size $PAGE"

# Preflight: the transport itself must work before anything is concluded.
if [ "$(shell_uid 'echo TRANSPORT_OK')" != "TRANSPORT_OK" ]; then
  echo "FATAL: base64 remote transport is not working on this device."; exit 1
fi
[ "$VERBOSE" = "1" ] && echo "  [dbg] transport preflight OK"

echo; echo "== install =="
"$ADB" install -r -t "$APK" >/dev/null 2>&1 || { echo "install failed"; exit 1; }
APKPATH=$("$ADB" shell pm path $PKG | tr -d '\r' | sed 's/^package://' | head -1)
[ -n "$APKPATH" ] || { echo "could not resolve APK path"; exit 1; }
NLD="$(dirname "$APKPATH")/lib/arm64"
PY="$NLD/libpocketclaw-python.so"
echo "  nativeLibraryDir: $NLD"

# run-as starts in the app's data directory, so the interpreter's HOME and
# TMPDIR are taken from there rather than reconstructed from a shell variable
# that might be empty.
remote appdir 'echo "$PWD"'
APPDIR="$OUT"
if [ -z "$APPDIR" ]; then no "resolve app data dir" "run-as returned no \$PWD"; else note "app data dir:     $APPDIR"; fi

# NOTE: do not route this through `env`. Android installs the APK under a
# randomised directory whose name ends in "==", and toybox `env` parses any
# argument containing "=" as a VAR=value assignment -- so it swallows the
# interpreter path and tries to exec the next argument ("-P") instead. The
# shell's own assignment prefix does not have that problem, because a word
# starting with "/" is not a valid assignment and is treated as the command.
PYENV="PYTHONHOME=/pocketclaw/python PYTHONPATH='$PY' PYTHONDONTWRITEBYTECODE=1 PYTHONUTF8=1 PYTHONNOUSERSITE=1 HOME='$APPDIR' TMPDIR='$APPDIR/pytmp'"
PYRUN="$PYENV '$PY' -P -s -S -B -u"
remote mktmp "mkdir -p '$APPDIR/pytmp' && echo made"
[ "$RC" = "0" ] && ok "create TMPDIR ($APPDIR/pytmp)" || no "create TMPDIR" "rc=$RC ${ERR:-no stderr}"

echo; echo "== ACCEPTANCE 1-4 (gate: the full matrix only runs if these pass) =="

remote payload_exists "ls -l '$PY' >/dev/null && echo present"
if [ "$RC" = "0" ] && [ "$OUT" = "present" ]; then ok "1. nativeLibraryDir payload exists"
else no "1. nativeLibraryDir payload exists" "rc=$RC ${ERR:-}"; fi

remote direct_exec "$PYRUN -c 'print(42)'"
if [ "$RC" = "0" ] && [ "$OUT" = "42" ]; then ok "2. direct executable invocation (app uid)"
else
  no "2. direct executable invocation (app uid)" "rc=$RC stderr=${ERR:-<empty>}"
  note "control, as adb shell uid: $(shell_uid "$PYRUN -c 'print(42)' 2>&1 | head -3")"
fi

remote version "$PYRUN --version"
PYVER="$OUT"
if [ "$RC" = "0" ] && [ "${OUT#Python 3.14}" != "$OUT" ]; then ok "3. python --version -> $OUT"
else no "3. python --version" "rc=$RC out='${OUT}' stderr=${ERR:-<empty>}"; fi

# Source goes in on stdin, never on argv. Only its byte count is ever logged.
remote stdin_pass "printf 'print(\"PYTHON-PASS\")\n' | $PYRUN -"
if [ "$RC" = "0" ] && [ "$OUT" = "PYTHON-PASS" ]; then ok "4. stdin script -> PYTHON-PASS"
else no "4. stdin script" "rc=$RC out='${OUT}' stderr=${ERR:-<empty>}"; fi

if [ $fail -ne 0 ]; then
  echo; echo "Acceptance 1-4 did not pass; stopping before the matrix so the failure stays readable."
  echo "RESULT: $pass passed, $fail failed"
  "$ADB" uninstall $PKG >/dev/null 2>&1
  exit 1
fi

echo; echo "== STEP 5: getpath =="
remote getpath "$PYRUN -c 'import sys
print(\"sys.executable :\", sys.executable)
print(\"sys.prefix     :\", sys.prefix)
print(\"sys.base_prefix:\", sys.base_prefix)
print(\"sys.path       :\", sys.path)
print(\"flags          : isolated=%d no_site=%d no_user_site=%d dont_write_bytecode=%d safe_path=%d\" % (sys.flags.isolated, sys.flags.no_site, sys.flags.no_user_site, sys.flags.dont_write_bytecode, sys.flags.safe_path))'"
printf '%s\n' "$OUT" | sed 's/^/        /'

echo; echo "== STEP 7: module matrix =="
for m in json csv re math statistics decimal fractions datetime pathlib hashlib hmac \
         secrets uuid urllib.parse html xml.etree.ElementTree zipfile tarfile gzip \
         bz2 lzma sqlite3 tempfile shutil glob fnmatch argparse subprocess signal \
         select fcntl unicodedata traceback logging; do
  remote "import_$m" "$PYRUN -c 'import $m; print(\"ok\")'"
  if [ "$RC" = "0" ] && [ "$OUT" = "ok" ]; then ok "import $m"
  else no "import $m" "rc=$RC $(printf '%s' "$ERR" | tail -1)"; fi
done
echo "  -- expected exclusions --"
for m in socket ssl ctypes multiprocessing email http; do
  remote "excl_$m" "$PYRUN -c 'import $m'"
  case "$ERR" in *ModuleNotFoundError*|*ImportError*) ok "$m correctly absent";;
    *) no "$m NOT excluded" "rc=$RC ${ERR:-<empty>}";; esac
done

echo; echo "== STEP 7b: real operations, not bare imports =="
remote compute "$PYRUN -c '
import json, csv, io, re, statistics, decimal, base64, urllib.parse
import xml.etree.ElementTree as ET
assert json.loads(json.dumps({\"a\": [1, 2]}))[\"a\"] == [1, 2]
assert list(csv.reader(io.StringIO(\"a,b\n1,2\"))) == [[\"a\",\"b\"],[\"1\",\"2\"]]
assert re.sub(r\"[0-9]+\", \"N\", \"ab123\") == \"abN\"
assert statistics.median([1, 3, 2]) == 2
assert str(decimal.Decimal(\"0.1\") + decimal.Decimal(\"0.2\")) == \"0.3\"
assert ET.fromstring(\"<r><c>x</c></r>\").find(\"c\").text == \"x\"
assert urllib.parse.urlparse(\"https://h/p?q=1\").query == \"q=1\"
assert base64.b64decode(base64.b64encode(b\"pc\")) == b\"pc\"
print(\"compute OK\")'"
[ "$RC" = "0" ] && ok "compute matrix ($OUT)" || no "compute matrix" "rc=$RC $(printf '%s' "$ERR" | tail -2)"

echo; echo "== STEP 8: sqlite3 =="
remote sqlite "$PYRUN -c '
import sqlite3
print(\"sqlite_version:\", sqlite3.sqlite_version)
c = sqlite3.connect(\":memory:\")
c.execute(\"create table t(a, b)\")
c.execute(\"insert into t values(?, ?)\", (1, \"x\"))
# sqlite3 opens a transaction implicitly under the default
# LEGACY_TRANSACTION_CONTROL, so an explicit BEGIN here would be an error.
c.execute(\"insert into t values(?, ?)\", (2, \"y\")); c.commit()
c.execute(\"insert into t values(?, ?)\", (3, \"rolled-back\")); c.rollback()
print(\"rows:\", c.execute(\"select * from t order by a\").fetchall())
c.execute(\"create virtual table f using fts5(body)\")
c.execute(\"insert into f values(?)\", (\"pocketclaw runtime\",))
print(\"fts5:\", c.execute(\"select body from f where f match ?\", (\"runtime\",)).fetchall())'"
if [ "$RC" = "0" ]; then ok "sqlite3 functional"; printf '%s\n' "$OUT" | sed 's/^/        /'
else no "sqlite3" "rc=$RC $(printf '%s' "$ERR" | tail -2)"; fi

echo; echo "== STEP 9: hashing without OpenSSL =="
remote hashing "$PYRUN -c '
import hashlib, hmac, secrets
print(\"sha256  :\", hashlib.sha256(b\"pocketclaw\").hexdigest())
print(\"sha3_256:\", hashlib.sha3_256(b\"pocketclaw\").hexdigest()[:32])
print(\"blake2b :\", hashlib.blake2b(b\"pocketclaw\").hexdigest()[:32])
print(\"hmac    :\", hmac.new(b\"k\", b\"m\", \"sha256\").hexdigest()[:32])
print(\"secrets :\", len(secrets.token_bytes(16)), \"bytes\")'"
if [ "$RC" = "0" ]; then ok "hashlib/hmac/secrets without OpenSSL"; printf '%s\n' "$OUT" | sed 's/^/        /'
else no "hashing" "rc=$RC $(printf '%s' "$ERR" | tail -2)"; fi

echo; echo "== STEP 10: unicode =="
remote unicode "$PYRUN -c '
import os
s = \"مرحبا من PocketClaw \U0001F40D\U0001F99E\U0001F525\"
print(\"stdout:\", s)
p = os.path.join(os.environ[\"TMPDIR\"], \"u.txt\")
open(p, \"w\", encoding=\"utf-8\").write(s)
back = open(p, encoding=\"utf-8\").read()
print(\"roundtrip:\", \"PASS\" if back == s else \"FAIL\")
print(\"len:\", len(s), \"utf8 bytes:\", len(s.encode()))
os.remove(p)'"
if [ "$RC" = "0" ] && printf '%s' "$OUT" | grep -q "roundtrip: PASS"; then ok "unicode stdout + file roundtrip"; printf '%s\n' "$OUT" | sed 's/^/        /'
else no "unicode" "rc=$RC $(printf '%s' "$ERR" | tail -2)"; fi

echo; echo "== STEP 11: shell facts (measured, not assumed) =="
remote shellfacts 'echo "command -v sh   : $(command -v sh 2>/dev/null || echo NONE)"
echo "/system/bin/sh  : $(ls -l /system/bin/sh 2>/dev/null || echo ABSENT)"
echo "/bin            : $(ls -ld /bin 2>/dev/null || echo ABSENT)"
echo "/bin/sh         : $(ls -l /bin/sh 2>/dev/null || echo ABSENT)"'
printf '%s\n' "$OUT" | sed 's/^/        /'
remote shellpy "$PYRUN -c '
import subprocess, os
r = subprocess.run([\"/system/bin/echo\", \"argv-exec-ok\"], capture_output=True, text=True)
print(\"shell=False        :\", (r.stdout or r.stderr).strip(), \"rc=%d\" % r.returncode)
try:
    r = subprocess.run(\"echo shell-true-ok\", shell=True, capture_output=True, text=True)
    print(\"shell=True         :\", (r.stdout or r.stderr).strip(), \"rc=%d\" % r.returncode)
except Exception as e:
    print(\"shell=True         : raised\", type(e).__name__)
print(\"os.system rc       :\", os.system(\"echo os-system-ok > /dev/null\"))'"
printf '%s\n' "$OUT" | sed 's/^/        /'
[ -n "$ERR" ] && printf '%s\n' "$ERR" | sed 's/^/        stderr: /'

echo; echo "== STEP 12: runaway termination =="
# Start, kill and reap inside ONE remote shell, so the PID comes from $! rather
# than from a ps scrape that Android may not show across sessions.
remote runaway "$PYRUN -c 'while True: pass' &
pid=\$!
sleep 2
kill -0 \$pid 2>/dev/null && echo running=yes || echo running=no
t0=\$(date +%s%N)
kill -9 \$pid 2>/dev/null
wait \$pid 2>/dev/null
t1=\$(date +%s%N)
echo kill_ms=\$(( (t1 - t0) / 1000000 ))
kill -0 \$pid 2>/dev/null && echo orphan=yes || echo orphan=no"
if [ "$RC" = "0" ]; then
  printf '%s\n' "$OUT" | sed 's/^/        /'
  printf '%s' "$OUT" | grep -q "running=yes" && ok "runaway process started" || no "runaway did not start"
  printf '%s' "$OUT" | grep -q "orphan=no"   && ok "killed, no orphan"        || no "orphan remained"
else no "runaway test" "rc=$RC ${ERR:-}"; fi

echo; echo "== STEP 15: startup latency (device, 10 runs) =="
bench() {
  local label="$1" code="$2" n="${3:-10}"
  remote "bench_$label" "i=0
while [ \$i -lt $n ]; do
  s=\$(date +%s%N)
  $PYRUN -c '$code' >/dev/null 2>&1
  e=\$(date +%s%N)
  echo \$(( (e - s) / 1000000 ))
  i=\$(( i + 1 ))
done"
  printf '%s\n' "$OUT" | grep -E '^[0-9]+$' | sort -n | awk -v L="$label" '
    {a[NR]=$1} END {if(NR)printf "        %-12s min=%sms  median=%sms  max=%sms  (n=%d)\n", L, a[1], a[int((NR+1)/2)], a[NR], NR;
                    else printf "        %-12s no timing samples\n", L}'
}
bench bare      'pass'
bench imports   'import json, re, datetime, hashlib'
bench sqlite    'import sqlite3; c = sqlite3.connect(":memory:"); c.execute("create table t(a)"); c.execute("insert into t values(1)"); c.execute("select * from t").fetchall()' 5

echo; echo "== STEP 16: memory (RSS, from /proc/self/status) =="
remote rss "$PYRUN -c '
def rss():
    for line in open(\"/proc/self/status\"):
        if line.startswith(\"VmRSS\"):
            return line.split()[1] + \" kB\"
    return \"?\"
print(\"bare       :\", rss())
import json, re, datetime, hashlib
print(\"typical    :\", rss())
import sqlite3
c = sqlite3.connect(\":memory:\"); c.execute(\"create table t(a)\")
for i in range(1000): c.execute(\"insert into t values(?)\", (i,))
print(\"sqlite 1k  :\", rss())
d = [{\"i\": i, \"s\": \"x\" * 50} for i in range(20000)]
json.loads(json.dumps(d))
print(\"json 20k   :\", rss())'"
if [ "$RC" = "0" ]; then printf '%s\n' "$OUT" | sed 's/^/        /'; ok "RSS measured"
else no "RSS" "rc=$RC $(printf '%s' "$ERR" | tail -2)"; fi

echo; echo "== cleanup =="
remote cleanrm "rm -rf '$APPDIR/pytmp'"
"$ADB" uninstall $PKG >/dev/null 2>&1 && echo "  probe app uninstalled"
echo; echo "RESULT: $pass passed, $fail failed"
[ $fail -eq 0 ]
