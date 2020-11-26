---
name: Bug Report
about: Bug reports accepted only for johnsonjh/pktd's develop branch.
title: ''
labels: ''
assignees: ''

---

* Before submitting your report, please be sure you have included a brief description of your build environment, and include the output of `go version`, `go env`, `git describe --tags --always --abbrev=40`, and `git status -s -b`.
* macOS X users should include the output of `uname -a` and `sw_vers` and attach the output of `system_profiler -detailLevel mini` and `serverinfo --plist`.
* Linux users should include the output of `uname -a`, `cat /etc/*elease`, and `lsb_release -a`.
* Windows users should attach the output of the `Get-CimInstance Win32_OperatingSystem | FL * | ?{$_ -notmatch 'SerialNumber'}` *PowerShell* command.
* Users of all POSIX environments (*including* macOS X, Windows WSL/WSL2, Linux, AIX, other UNIX systems, etc.) should also attach the output of `command -p getconf -a`.
