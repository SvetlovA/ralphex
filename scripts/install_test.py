#!/usr/bin/env python3
"""Exercise release selection and checksum enforcement without network access."""
import hashlib
import io
import json
import os
from pathlib import Path
import subprocess
import tarfile
import tempfile
import unittest


INSTALLER = Path(__file__).with_name("install.sh")
TAG = "v1.7.0-windows.2"


class InstallerTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.bin = self.root / "tools"
        self.bin.mkdir()
        self.destination = self.root / "install with spaces"
        self.env = dict(os.environ, PATH=f"{self.bin}:{os.environ['PATH']}",
                        RALPHEX_INSTALL_DIR=str(self.destination),
                        TEST_FIXTURES=str(self.root), TEST_OS="Linux", TEST_ARCH="x86_64")
        self.env.pop("RALPHEX_VERSION", None)
        self.write_tool("uname", '#!/bin/sh\ncase "$1" in -s) echo "$TEST_OS";; -m) echo "$TEST_ARCH";; esac\n')
        self.write_tool("curl", '''#!/usr/bin/env python3
import os, pathlib, sys
args = sys.argv[1:]
url = next(arg for arg in args if arg.startswith("https://"))
source = pathlib.Path(os.environ["TEST_FIXTURES"]) / url.rsplit("/", 1)[1]
if "-o" in args:
    pathlib.Path(args[args.index("-o") + 1]).write_bytes(source.read_bytes())
else:
    sys.stdout.buffer.write(source.read_bytes())
''')
        (self.root / "latest").write_text(json.dumps({"tag_name": TAG}))

    def write_tool(self, name, content):
        path = self.bin / name
        path.write_text(content)
        path.chmod(0o755)

    def fixture(self, os_name="linux", arch="amd64", binary=True):
        asset = f"ralphex_{TAG[1:]}_{os_name}_{arch}.tar.gz"
        path = self.root / asset
        with tarfile.open(path, "w:gz") as archive:
            data = b"#!/bin/sh\necho fixture\n"
            member = tarfile.TarInfo("ralphex" if binary else "other")
            member.size = len(data)
            archive.addfile(member, io.BytesIO(data))
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        (self.root / f"ralphex_{TAG[1:]}_checksums.txt").write_text(f"{digest}  {asset}\n")
        return path

    def run_installer(self):
        return subprocess.run(["sh", str(INSTALLER)], env=self.env,
                              capture_output=True, text=True)

    def test_latest_release_all_platforms(self):
        for system, machine, release_os, arch in [
            ("Linux", "x86_64", "linux", "amd64"),
            ("Linux", "aarch64", "linux", "arm64"),
            ("Darwin", "x86_64", "darwin", "amd64"),
            ("Darwin", "arm64", "darwin", "arm64"),
        ]:
            with self.subTest(system=system, machine=machine):
                self.env.update(TEST_OS=system, TEST_ARCH=machine)
                self.fixture(release_os, arch)
                result = self.run_installer()
                self.assertEqual(result.returncode, 0, result.stderr)
                installed = self.destination / "ralphex"
                self.assertEqual(installed.read_bytes(), b"#!/bin/sh\necho fixture\n")
                self.assertEqual(installed.stat().st_mode & 0o777, 0o755)

    def test_pinned_release_skips_latest_lookup(self):
        self.fixture()
        self.env["RALPHEX_VERSION"] = TAG
        (self.root / "latest").unlink()
        result = self.run_installer()
        self.assertEqual(result.returncode, 0, result.stderr)

    def test_checksum_mismatch_preserves_existing_installation(self):
        archive = self.fixture()
        archive.write_bytes(archive.read_bytes() + b"tampered")
        self.destination.mkdir()
        installed = self.destination / "ralphex"
        installed.write_text("previous version")
        result = self.run_installer()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("checksum mismatch", result.stderr)
        self.assertEqual(installed.read_text(), "previous version")

    def test_missing_checksum_rejected(self):
        self.fixture()
        (self.root / f"ralphex_{TAG[1:]}_checksums.txt").write_text("")
        result = self.run_installer()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("missing or invalid checksum", result.stderr)
        self.assertFalse(self.destination.exists())

    def test_missing_binary_rejected(self):
        self.fixture(binary=False)
        self.assertNotEqual(self.run_installer().returncode, 0)
        self.assertFalse(self.destination.exists())

    def test_invalid_release_rejected(self):
        self.env["RALPHEX_VERSION"] = "v1.7.0"
        result = self.run_installer()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("invalid fork release tag", result.stderr)

    def test_unsupported_architecture_rejected(self):
        self.env["TEST_ARCH"] = "riscv64"
        result = self.run_installer()
        self.assertNotEqual(result.returncode, 0)
        self.assertIn("unsupported architecture", result.stderr)


if __name__ == "__main__":
    unittest.main()
