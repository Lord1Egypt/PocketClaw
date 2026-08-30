#!/usr/bin/env python3
"""Package the Python Lite standard library as importable zips.

EXPERIMENTAL — PYTHON LITE PHASE A. Run with the CPython 3.14 build
interpreter, so the emitted .pyc magic matches the target interpreter.

Writes stdlib-py.zip (source) and stdlib-pyc.zip (bytecode) and reports any
module left in the set whose module-level imports cannot be satisfied by the
Lite interpreter. A non-empty "broken_modules" means the exclusion list is
wrong: fix the list rather than shipping a stdlib that raises on import.
"""
import ast
import json
import os
import py_compile
import shutil
import sys
import tempfile
import zipfile

# Extension modules the Lite interpreter does not contain, either disabled in
# Modules/Setup.local or unavailable on Android.
ABSENT_C = {
    "_socket", "_ssl", "_hashlib", "_ctypes", "_zstd", "_asyncio",
    "_multibytecodec", "_codecs_cn", "_codecs_hk", "_codecs_iso2022",
    "_codecs_jp", "_codecs_kr", "_codecs_tw", "_interpreters",
    "_interpchannels", "_interpqueues", "_remote_debugging", "_lsprof",
    "_zoneinfo", "syslog", "termios", "_curses", "_curses_panel", "_dbm",
    "_gdbm", "_multiprocessing", "_posixshmem", "_tkinter", "grp", "readline",
}

DROP_DIRS = {
    "test", "idlelib", "tkinter", "ensurepip", "pydoc_data", "turtledemo",
    "__pycache__", "lib-dynload", "site-packages", "asyncio", "concurrent",
    "multiprocessing", "unittest", "venv", "wsgiref", "xmlrpc", "curses",
    "dbm", "zoneinfo", "ctypes", "email", "http", "_pyrepl", "__phello__",
}

DROP_FILES = {
    "pydoc.py", "doctest.py", "pdb.py", "bdb.py", "profile.py", "cProfile.py",
    "pstats.py", "turtle.py", "this.py", "antigravity.py", "socket.py",
    "ssl.py", "ftplib.py", "smtplib.py", "poplib.py", "imaplib.py",
    "socketserver.py", "getpass.py", "webbrowser.py", "mailbox.py", "tty.py",
    "pty.py", "cgi.py", "cgitb.py", "timeit.py", "trace.py", "cmd.py",
    "code.py", "compileall.py", "__hello__.py", "_aix_support.py",
    "_apple_support.py", "_ios_support.py", "_osx_support.py",
    # Records the machine the interpreter was built on; nothing here imports it.
    "_sysconfigdata__android_aarch64-linux-android.py",
}

# codeop.py is deliberately NOT dropped: traceback imports it at module level.
CJK_ENCODINGS = (
    "big5", "big5hkscs", "cp932", "cp949", "cp950", "euc_jis_2004",
    "euc_jisx0213", "euc_jp", "euc_kr", "gb18030", "gb2312", "gbk", "hz",
    "iso2022_jp", "iso2022_jp_1", "iso2022_jp_2", "iso2022_jp_2004",
    "iso2022_jp_3", "iso2022_jp_ext", "iso2022_kr", "johab", "shift_jis",
    "shift_jis_2004", "shift_jisx0213",
)

DROP_REL = {
    "urllib/request.py", "urllib/response.py", "urllib/robotparser.py",
    "logging/handlers.py", "logging/config.py",
    "xml/sax/__init__.py", "xml/sax/saxutils.py", "xml/sax/xmlreader.py",
    "xml/sax/handler.py", "xml/sax/_exceptions.py", "xml/sax/expatreader.py",
    "xml/dom/xmlbuilder.py",
    "sqlite3/__main__.py",
    "compression/zstd/__init__.py", "compression/zstd/_zstdfile.py",
}
DROP_REL |= {f"encodings/{name}.py" for name in CJK_ENCODINGS}
# importlib.metadata needs email, which the Lite profile excludes.
DROP_REL |= {
    "importlib/metadata/__init__.py", "importlib/metadata/_adapters.py",
    "importlib/metadata/_collections.py", "importlib/metadata/_functools.py",
    "importlib/metadata/_itertools.py", "importlib/metadata/_meta.py",
    "importlib/metadata/_text.py", "importlib/metadata/diagnose.py",
}


def included(lib):
    found = []
    for root, dirs, files in os.walk(lib):
        dirs[:] = [d for d in dirs if d not in DROP_DIRS]
        rel_root = os.path.relpath(root, lib)
        for name in files:
            if not name.endswith(".py"):
                continue
            rel = os.path.normpath(os.path.join(rel_root, name)).replace(os.sep, "/")
            if rel_root == "." and name in DROP_FILES:
                continue
            if rel in DROP_REL:
                continue
            found.append(rel)
    return sorted(found)


def module_level_imports(path):
    """Names imported unconditionally at module level.

    Imports inside try/except ImportError are what the stdlib uses for optional
    accelerators, and those degrade gracefully, so they are not reported.
    """
    try:
        tree = ast.parse(open(path, encoding="utf-8", errors="ignore").read())
    except SyntaxError:
        return set()
    names = set()
    for node in tree.body:
        if isinstance(node, ast.Import):
            names |= {a.name.split(".")[0] for a in node.names}
        elif isinstance(node, ast.ImportFrom) and node.level == 0 and node.module:
            names.add(node.module.split(".")[0])
    return names


def main():
    lib, out = sys.argv[1], sys.argv[2]
    files = included(lib)
    present = {rel.split("/")[0].removesuffix(".py") for rel in files}
    dropped_py = {name.removesuffix(".py") for name in DROP_FILES}

    broken = {}
    for rel in files:
        bad = sorted(
            name for name in module_level_imports(os.path.join(lib, rel))
            if name in ABSENT_C
            or (name in (DROP_DIRS | dropped_py) and name not in present)
        )
        if bad:
            broken[rel] = bad

    os.makedirs(out, exist_ok=True)
    py_zip = os.path.join(out, "stdlib-py.zip")
    with zipfile.ZipFile(py_zip, "w", zipfile.ZIP_DEFLATED, compresslevel=9) as z:
        for rel in files:
            z.write(os.path.join(lib, rel), rel)

    staging = tempfile.mkdtemp()
    failures = []
    try:
        for rel in files:
            dst = os.path.join(staging, rel + "c")
            os.makedirs(os.path.dirname(dst), exist_ok=True)
            try:
                py_compile.compile(os.path.join(lib, rel), cfile=dst, dfile=rel,
                                   doraise=True)
            except py_compile.PyCompileError as exc:
                failures.append(f"{rel}: {exc}")
        pyc_zip = os.path.join(out, "stdlib-pyc.zip")
        with zipfile.ZipFile(pyc_zip, "w", zipfile.ZIP_DEFLATED, compresslevel=9) as z:
            for root, _, names in os.walk(staging):
                for name in names:
                    full = os.path.join(root, name)
                    z.write(full, os.path.relpath(full, staging))
        raw_pyc = sum(os.path.getsize(os.path.join(r, n))
                      for r, _, ns in os.walk(staging) for n in ns)
    finally:
        shutil.rmtree(staging)

    print(json.dumps({
        "files": len(files),
        "raw_py": sum(os.path.getsize(os.path.join(lib, r)) for r in files),
        "zip_py": os.path.getsize(py_zip),
        "raw_pyc": raw_pyc,
        "zip_pyc": os.path.getsize(pyc_zip),
        "compile_failures": failures,
        "broken_modules": broken,
    }, indent=2))

    if failures or broken:
        sys.exit(1)


if __name__ == "__main__":
    main()
