#!/usr/bin/env python3
import os
import shutil
import subprocess
import sys
from pathlib import Path

VERSION = "0.1.1"
BIN_DIR = "bin"
DIST_DIR = "dist"
STORY = "examples/intro"

def run_cmd(cmd):
    print(f"[RUN] {' '.join(cmd)}")
    result = subprocess.run(cmd)
    if result.returncode != 0:
        print(f"[ERROR] Perintah gagal dengan kode {result.returncode}")
        sys.exit(1)

def build_target(goos, goarch, exe_ext=""):
    pkg_name = f"ango-{VERSION}-{goos}-{goarch}"
    pkg_dir = Path(DIST_DIR) / pkg_name
    
    print(f"\n--- Membangun untuk {goos}/{goarch} ---")
    
    # 1. Buat direktori paket
    if pkg_dir.exists():
        shutil.rmtree(pkg_dir)
    projects_dir = pkg_dir / "projects" / Path(STORY).name
    projects_dir.mkdir(parents=True, exist_ok=True)
    
    # 2. Compile Go binary dengan env GOOS & GOARCH
    env = os.environ.copy()
    env["GOOS"] = goos
    env["GOARCH"] = goarch
    env["CGO_ENABLED"] = "0"
    
    ango_bin = f"bin/ango{exe_ext}"
    launcher_bin = f"bin/ango-launcher{exe_ext}"
    
    ldflags = f"-s -w -X main.version={VERSION}"

    run_cmd(["go", "build", "-trimpath", f"-ldflags={ldflags}", "-o", ango_bin, "./cmd/ango"])
    run_cmd(["go", "build", "-trimpath", f"-ldflags={ldflags}", "-o", launcher_bin, "./cmd/ango-launcher"])
    
    # 3. Salin binary ke folder dist
    shutil.copy(ango_bin, pkg_dir / f"ango{exe_ext}")
    shutil.copy(launcher_bin, pkg_dir / f"ango-launcher{exe_ext}")
    
    # 4. Salin skrip launcher & aset cerita
    if goos == "windows":
        shutil.copy("scripts/ango.bat", pkg_dir / "ango.bat")
        shutil.copy("scripts/play.bat", pkg_dir / "play.bat")
    else:
        shutil.copy("scripts/ango.sh", pkg_dir / "ango.sh")
        shutil.copy("scripts/play.sh", pkg_dir / "play.sh")
        # Berikan hak akses executable di Linux
        (pkg_dir / "ango.sh").chmod(0o755)
        (pkg_dir / "play.sh").chmod(0o755)
        
    shutil.copytree(STORY, projects_dir, dirs_exist_ok=True)
    shutil.copy("README.md", pkg_dir / "README.md")
    shutil.copy("LICENSE", pkg_dir / "LICENSE")
    if Path("docs").exists():
        shutil.copytree("docs", pkg_dir / "docs", dirs_exist_ok=True)
        
    # 5. Kompresi arsip (Zip untuk Windows, Tar.gz untuk Linux)
    os.makedirs(DIST_DIR, exist_ok=True)
    if goos == "windows":
        archive_path = Path(DIST_DIR) / f"{pkg_name}.zip"
        shutil.make_archive(str(archive_path.with_suffix("")), 'zip', DIST_DIR, pkg_name)
    else:
        archive_path = Path(DIST_DIR) / f"{pkg_name}.tar.gz"
        run_cmd(["tar", "-C", DIST_DIR, "-czf", str(archive_path), pkg_name])
        
    print(f"[SUKSES] Paket rilis tersimpan di: {archive_path}")

if __name__ == "__main__":
    # Jalankan build lintas platform otomatis
    build_target("linux", "amd64", "")
    build_target("windows", "amd64", ".exe")
    print("\Semua proses build selesai!")
