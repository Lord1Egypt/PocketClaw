#!/usr/bin/env python3
"""Install pocketclaw_bootstrap into a Python Lite payload.

The payload is a CPython ELF with the standard library appended as a zip that
zipimport reads out of the same file. The bootstrap belongs in that zip: it is
trusted PocketClaw code, and shipping it inside the payload means it is covered
by the checksum the runtime verifies before executing anything, rather than
living in writable storage where the code it supervises could rewrite it.

The module is added as source rather than bytecode on purpose. A .pyc entry
would have to match the target interpreter's magic number, which would tie this
step to the exact CPython that built the payload; zipimport compiles a .py entry
itself, and the cost is a millisecond on a 90 ms interpreter start.

Usage: install-python-bootstrap.py <payload> [module source]
"""
import os
import sys
import zipfile

ENTRY = "pocketclaw_bootstrap.py"


def main(argv):
    if not 2 <= len(argv) <= 3:
        sys.exit(__doc__.strip().splitlines()[-1])
    payload = argv[1]
    source_path = argv[2] if len(argv) == 3 else os.path.join(
        os.path.dirname(os.path.abspath(__file__)), ENTRY)

    with open(source_path, "rb") as handle:
        source = handle.read()

    with open(payload, "rb") as handle:
        if handle.read(4) != b"\x7fELF":
            sys.exit("error: %s does not start with an ELF header" % payload)

    try:
        with zipfile.ZipFile(payload) as archive:
            present = ENTRY in archive.namelist()
            if present and archive.read(ENTRY) == source:
                print("  %s already installed and current" % ENTRY)
                return
            if present:
                sys.exit(
                    "error: %s carries a different %s.\n"
                    "Appending would leave two entries with one name. Rebuild the "
                    "payload with runtime/build-python-android-arm64.sh." % (payload, ENTRY))
    except zipfile.BadZipFile as error:
        sys.exit("error: %s has no readable appended stdlib: %s" % (payload, error))

    with zipfile.ZipFile(payload, "a", zipfile.ZIP_DEFLATED, compresslevel=9) as archive:
        archive.writestr(ENTRY, source)

    verify(payload, source)


def verify(payload, source):
    """Prove the payload is still what the interpreter and the runtime need.

    Appending rewrites the central directory at the end of the file. The ELF in
    front of it must be untouched, every module that was already there must
    still read back, and the new entry must be importable.
    """
    with open(payload, "rb") as handle:
        if handle.read(4) != b"\x7fELF":
            sys.exit("error: the ELF header did not survive the append")

    with zipfile.ZipFile(payload) as archive:
        names = archive.namelist()
        if "json/__init__.pyc" not in names:
            sys.exit("error: the appended stdlib no longer reads back")
        if archive.read(ENTRY) != source:
            sys.exit("error: %s did not read back as written" % ENTRY)
        broken = archive.testzip()
        if broken is not None:
            sys.exit("error: %s is corrupt in the rewritten archive" % broken)
    print("  %s installed, %d entries in the appended stdlib" % (ENTRY, len(names)))


if __name__ == "__main__":
    main(sys.argv)
