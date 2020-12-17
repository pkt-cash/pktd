// Copyright 2020 Anode LLC
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// This file is ignored during the regular build due to the following build tag.
// It is called by go generate and used to automatically generate pre-computed
// tables used to accelerate operations.
// +build ignore

package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/pkt-cash/pktd/pktconfig/version"
)

func dieIfError(err error) {
	if err == nil {
		return
	}
	fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
	os.Exit(1)
}

func main() {
	fmt.Printf("Generating seed words\n")
	version.SetUserAgentName("genwords")
	fi, err := os.Create("words.go")
	dieIfError(err)
	defer fi.Close()

	var langs []string

	wd, err := os.Getwd()
	dieIfError(err)
	files, err := ioutil.ReadDir(wd)
	dieIfError(err)
	for _, f := range files {
		if strings.HasSuffix(f.Name(), ".words.txt") {
			langs = append(langs, strings.ReplaceAll(f.Name(), ".words.txt", ""))
		}
	}

	fmt.Fprintln(fi, "// Copyright (c) 2020 Anode LLC")
	fmt.Fprintln(fi, "// Use of this source code is governed by an ISC")
	fmt.Fprintln(fi, "// license that can be found in the LICENSE file.")
	fmt.Fprintln(fi)
	fmt.Fprintln(fi, "package seedwords")
	fmt.Fprintln(fi)
	fmt.Fprintln(fi, "// Auto-generated file (see genwords.go)")
	fmt.Fprintln(fi, "// DO NOT EDIT")
	fmt.Fprintln(fi)

	for _, l := range langs {
		b, err := ioutil.ReadFile(l + ".words.txt")
		if err != nil {
			dieIfError(err)
		}
		strs := strings.Split(string(b), "\n")
		fmt.Fprintf(fi, "var words_%s = [2048]string{\n", l)
		for _, str := range strs {
			if str == "" {
				continue
			}
			fmt.Fprintf(fi, "    \"%s\",\n", str)
		}
		fmt.Fprintln(fi, "}")
		fmt.Fprintf(fi, "var rwords_%s = map[string]int16{\n", l)
		for i, str := range strs {
			if str == "" {
				continue
			}
			fmt.Fprintf(fi, "    \"%s\": %d,\n", str, i)
		}
		fmt.Fprintln(fi, "}")
	}
	fmt.Fprintln(fi, "func init() {")
	for _, l := range langs {
		fmt.Fprintf(fi, `    allWords["%s"] = &wordsDesc{ words: words_%s, rwords: rwords_%s, lang: "%s" }%s`,
			l, l, l, l, "\n")
	}
	fmt.Fprintln(fi, "}")
}
