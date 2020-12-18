// Copyright (c) 2015 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package wallet

import (
	"os"

	"github.com/pkt-cash/pktd/btcutil/er"
)

// checkCreateDir checks that the path exists and is a directory.
// If path does not exist, it is created.
func checkCreateDir(path string) er.R {
	if fi, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			// Attempt data directory creation
			if err = os.MkdirAll(path, 0o700); err != nil {
				return er.Errorf("cannot create directory: %s", err)
			}
		} else {
			return er.Errorf("error checking directory: %s", err)
		}
	} else {
		if !fi.IsDir() {
			return er.Errorf("path '%s' is not a directory", path)
		}
	}

	return nil
}
