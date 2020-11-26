---
name: Bug Report
about: Bug reporting for pktd/pktwallet/pktctl
title: ''
labels: ''
assignees: ''

---

* When submitting a bug report, please provide a brief description of your build environment, and attach the output of:
  * `go version`
  * `go env`
  * `git describe --tags --always --abbrev=40`
  * `git status -s -b`
  
* macOS X users should attach the output of:
  * `uname -a`
  * `sw_vers`
  * `system_profiler -detailLevel mini`
  * `serverinfo --plist`
  
* Linux users should attach the output of:
  * `uname -a`
  * `cat /etc/*elease`
  * `lsb_release -a`.
  
* Windows users should attach the *PowerShell* output of:
  * `Get-CimInstance Win32_OperatingSystem | FL * | ?{$_ -notmatch 'SerialNumber'}`
  
* Users of all POSIX environments (*including* OS X, WSL/WSL2, Linux, other UNIX systems, etc.) should attach the output of:
  * `command -p getconf -a`.
