// Copyright (c) 2013-2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package version

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkt-cash/pktd/pktlog"
)

// appBuild is defined as a variable so it can be overridden during the build
// process with '-ldflags "-X main.appBuild foo' if needed.  It MUST only
// contain characters from semanticAlphabet per the semantic versioning spec.
var appBuild string

var userAgentName = "unknown" // pktd, pktwallet, pktctl...
var appMajor uint = 0
var appMinor uint = 0
var appPatch uint = 0
var version = "0.0.0-custom"
var custom = true
var prerelease = false
var dirty = false

func init() {
	if appBuild == "" {
		// custom build
		return
	}
	tag := "-custom"
	// pktd-v1.1.0-beta-19-gfa3ba767
	if _, err := fmt.Sscanf(appBuild, "pktd-v%d.%d.%d", &appMajor, &appMinor, &appPatch); err == nil {
		tag = ""
		custom = false
		if x := regexp.MustCompile(`-[0-9]+-g[0-9a-f]{8}`).FindString(appBuild); len(x) > 0 {
			tag += "-" + x[strings.LastIndex(x, "-")+2:]
			prerelease = true
		}
		if strings.Contains(appBuild, "-dirty") {
			tag += "-dirty"
			dirty = true
		}
	}
	version = fmt.Sprintf("%d.%d.%d%s", appMajor, appMinor, appPatch, tag)
}

func IsCustom() bool {
	return custom
}

func IsDirty() bool {
	return dirty
}

func IsPrerelease() bool {
	return prerelease
}

func AppMajorVersion() uint {
	return appMajor
}
func AppMinorVersion() uint {
	return appMinor
}
func AppPatchVersion() uint {
	return appPatch
}

func SetUserAgentName(ua string) {
	if userAgentName != "unknown" {
		panic("setting useragent to [" + ua +
			"] failed, useragent was already set to [" + userAgentName + "]")
	}
	userAgentName = ua
}

func Version() string {
	return version
}

func UserAgentName() string {
	return userAgentName
}

func UserAgentVersion() string {
	return version
}

func WarnIfPrerelease(log pktlog.Logger) {
	if IsCustom() || IsDirty() {
		log.Warnf("THIS IS A DEVELOPMENT VERSION, THINGS MAY BREAK")
	} else if IsPrerelease() {
		log.Infof("This is a pre-release version")
	}
}
