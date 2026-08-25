# Android Termux Guide

> Back to [Guides](README.md)

This guide covers running the PicoClaw terminal binary on an ARM64 Android phone with Termux. Use the APK from [picoclaw.io](https://picoclaw.io/download/) if you want the Android app experience; use Termux when you want a lightweight command-line install on an older or resource-constrained device.

## Requirements

- ARM64 Android device. Run `uname -m` in Termux and use this guide when it prints `aarch64`.
- Termux installed from [Termux GitHub Releases](https://github.com/termux/termux-app/releases) or F-Droid.
- Network access for downloading the release and calling your LLM provider.
- An API key for at least one configured model provider.

## Install PicoClaw

Open Termux and install the packages used by the release archive and chroot wrapper:

```bash
pkg update
pkg install -y wget tar proot
```

Download and unpack the ARM64 Linux release:

```bash
mkdir -p ~/picoclaw
cd ~/picoclaw
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz
tar xzf picoclaw_Linux_arm64.tar.gz
chmod +x ./picoclaw
```

Start first-run setup through `termux-chroot`, which gives the Linux binary a more standard filesystem layout than a raw Android userspace:

```bash
termux-chroot ./picoclaw onboard
```

## Configure

Edit the generated config and add at least one model provider API key:

```bash
vim ~/.picoclaw/config.json
```

The default workspace is `~/.picoclaw/workspace`. If you want PicoClaw to read or write Android shared storage, run `termux-setup-storage` first and then point the workspace or any file paths at the mounted storage directory.

See [Configuration Guide](configuration.md) and [Providers & Model Configuration](providers.md) for the available config fields and provider examples.

## Run

Use one-shot agent mode to confirm the installation:

```bash
termux-chroot ./picoclaw agent -m "Hello from Termux"
```

For long-running use, start the gateway:

```bash
termux-chroot ./picoclaw gateway
```

Keep the Termux session open while PicoClaw is running. Android battery optimization can stop background work, so disable battery optimization for Termux if you expect PicoClaw to keep running after the screen locks.

## Update

Your config and workspace live under `~/.picoclaw`, so updating the binary does not remove them:

```bash
cd ~/picoclaw
rm -f picoclaw_Linux_arm64.tar.gz
wget https://github.com/sipeed/picoclaw/releases/latest/download/picoclaw_Linux_arm64.tar.gz
tar xzf picoclaw_Linux_arm64.tar.gz
chmod +x ./picoclaw
termux-chroot ./picoclaw version
```

## Troubleshooting

| Symptom | Check |
|---------|-------|
| `permission denied` | Run `chmod +x ./picoclaw` after unpacking the archive. |
| `not found` after running `./picoclaw` | Confirm `uname -m` prints `aarch64` and that you downloaded `picoclaw_Linux_arm64.tar.gz`. |
| Files or paths behave differently than Linux | Run PicoClaw through `termux-chroot` instead of calling the binary directly. |
| Provider requests fail | Check the API key and network access in `~/.picoclaw/config.json`. |
| PicoClaw stops when the phone sleeps | Disable Android battery optimization for Termux and keep a foreground Termux session active. |
