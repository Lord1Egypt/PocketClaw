#!/usr/bin/env python3
"""Behavioural tests for the key-ceremony helper's fingerprint extraction.

The static half of this contract lives in
test/unit/production_signing_contract_test.dart, where the release gate runs it.
This half needs a real keytool and a real keystore, so it is separate: it builds
a disposable PKCS12 in a temporary directory, reads its fingerprint back through
the helper, and throws the keystore away.

Nothing here touches the production keystore, and no password in this file is a
secret: each one is generated per-run, used only by the temporary keystore it
creates, and dies with it.
"""

from __future__ import annotations

import base64
import os
import re
import secrets
import shutil
import subprocess
import tempfile
import unittest
from pathlib import Path

REPO = Path(__file__).resolve().parent.parent
HELPER = REPO / "tool/create_release_keystore.sh"
ALIAS = "pocketclaw-release"

HEX64 = re.compile(r"^[0-9a-f]{64}$")


def keytool() -> str | None:
    found = shutil.which("keytool")
    if found:
        return found
    java_home = os.environ.get("JAVA_HOME")
    if java_home and (Path(java_home) / "bin/keytool").is_file():
        return str(Path(java_home) / "bin/keytool")
    return None


@unittest.skipIf(keytool() is None, "keytool is not on PATH; set JAVA_HOME")
class FingerprintExtraction(unittest.TestCase):
    """A disposable keystore, created once for the whole class."""

    @classmethod
    def setUpClass(cls):
        cls._dir = tempfile.mkdtemp(prefix="pocketclaw-keystore-test-")
        cls.keystore = Path(cls._dir) / "disposable.p12"
        # Per-run and throwaway. The keystore is deleted in tearDownClass.
        cls.password = base64.b64encode(secrets.token_bytes(18)).decode()
        env = dict(os.environ, DISPOSABLE_PW=cls.password)
        subprocess.run(
            [keytool(), "-genkeypair",
             "-keystore", str(cls.keystore), "-storetype", "PKCS12",
             "-alias", ALIAS, "-keyalg", "RSA", "-keysize", "2048",
             "-sigalg", "SHA256withRSA", "-validity", "1",
             "-dname", "CN=disposable-test-key",
             "-storepass:env", "DISPOSABLE_PW", "-keypass:env", "DISPOSABLE_PW"],
            env=env, check=True, capture_output=True, text=True,
        )

    @classmethod
    def tearDownClass(cls):
        shutil.rmtree(cls._dir, ignore_errors=True)

    def run_helper(self, stdin: str):
        return subprocess.run(
            ["bash", str(HELPER), "--print-fingerprint", str(self.keystore), ALIAS],
            input=stdin, capture_output=True, text=True,
        )

    def expected_fingerprint(self) -> str:
        env = dict(os.environ, DISPOSABLE_PW=self.password)
        listing = subprocess.run(
            [keytool(), "-list", "-v", "-keystore", str(self.keystore),
             "-alias", ALIAS, "-storepass:env", "DISPOSABLE_PW"],
            env=env, check=True, capture_output=True, text=True,
        ).stdout
        for line in listing.splitlines():
            if "SHA256:" in line:
                return line.split()[1].replace(":", "").lower()
        self.fail("keytool itself printed no SHA256 fingerprint")

    def test_it_prints_the_certificate_fingerprint(self):
        result = self.run_helper(self.password + "\n")
        self.assertEqual(result.returncode, 0, result.stderr)
        printed = result.stdout.strip()
        self.assertRegex(printed, HEX64,
                         "the enrollment file wants 64 lowercase hex, no colons")
        self.assertEqual(printed, self.expected_fingerprint())

    def test_no_password_fails_loudly_instead_of_printing_nothing(self):
        # The regression. keytool exits 0 when it gets no password: it lists the
        # entry without its certificate, so there is no SHA256 line to find. The
        # helper must not report that as a successful empty fingerprint.
        result = self.run_helper("")
        self.assertNotEqual(result.returncode, 0,
                            "a missing fingerprint must be a failure")
        self.assertEqual(result.stdout.strip(), "",
                         "nothing may be printed as if it were a fingerprint")
        self.assertIn("could not read a SHA-256 certificate fingerprint",
                      result.stderr)

    def test_a_wrong_alias_fails_rather_than_printing_nothing(self):
        result = subprocess.run(
            ["bash", str(HELPER), "--print-fingerprint",
             str(self.keystore), "no-such-alias"],
            input=self.password + "\n", capture_output=True, text=True,
        )
        self.assertNotEqual(result.returncode, 0)
        self.assertEqual(result.stdout.strip(), "")

    def test_the_helper_does_not_echo_the_password(self):
        result = self.run_helper(self.password + "\n")
        self.assertNotIn(self.password, result.stdout)
        self.assertNotIn(self.password, result.stderr)


class ReadOnlyMode(unittest.TestCase):
    def test_print_fingerprint_creates_nothing(self):
        with tempfile.TemporaryDirectory() as tmp:
            missing = Path(tmp) / "absent.p12"
            subprocess.run(
                ["bash", str(HELPER), "--print-fingerprint", str(missing), ALIAS],
                input="", capture_output=True, text=True,
            )
            self.assertFalse(missing.exists(),
                             "the re-read mode must never create a keystore")

    def test_usage_error_without_a_keystore(self):
        result = subprocess.run(
            ["bash", str(HELPER), "--print-fingerprint"],
            input="", capture_output=True, text=True,
        )
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("usage", result.stderr)


if __name__ == "__main__":
    unittest.main(verbosity=2)
