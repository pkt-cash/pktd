// Copyright (c) 2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

// The vast majority of the rules tested in this package were ported from the
// the original Java-based 'official' block acceptance tests at
// https://github.com/TheBlueMatt/test-scripts as well as some additional tests
// available in the Core python port of the same.

package testdata

import (
	"bytes"
	"compress/bzip2"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/wire/protocol"

	"github.com/pkt-cash/pktd/btcutil"
	"github.com/pkt-cash/pktd/wire"
)

// LoadBlocks reads files containing bitcoin block data (gzipped but otherwise
// in the format bitcoind writes) from disk and returns them as an array of
// btcutil.Block.  This is largely borrowed from the test code in pktdb.
func LoadBlocks(filename string) ([]*btcutil.Block, er.R) {
	network := protocol.MainNet
	var dr io.Reader
	var fi io.ReadCloser

	fi, err := os.Open(filename)
	if err != nil {
		return nil, er.E(err)
	}

	if strings.HasSuffix(filename, ".bz2") {
		dr = bzip2.NewReader(fi)
	} else {
		dr = fi
	}
	defer fi.Close()

	var blocks []*btcutil.Block

	for height := int64(1); err == nil; height++ {
		var rintbuf uint32
		err := er.E(binary.Read(dr, binary.LittleEndian, &rintbuf))
		if er.Wrapped(err) == io.EOF {
			// hit end of file at expected offset: no warning
			height--
			err = nil
			break
		}
		if err != nil {
			break
		}
		if rintbuf != uint32(network) {
			break
		}
		err = er.E(binary.Read(dr, binary.LittleEndian, &rintbuf))
		blocklen := rintbuf

		rbytes := make([]byte, blocklen)

		// read block
		dr.Read(rbytes)

		mb := wire.MsgBlock{}
		err = mb.BtcDecode(bytes.NewBuffer(rbytes), 0, wire.NoPacketCryptEncoding)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, btcutil.NewBlock(&mb))
	}

	return blocks, nil
}

func GetBlock(name string, t *testing.T) *btcutil.Block {
	runtime.GOMAXPROCS(runtime.NumCPU() * 6)

	blocks, err := LoadBlocks(name)
	if err != nil {
		t.Errorf("Error loading file: %v\n", err)
		return nil
	}
	if len(blocks) > 1 {
		t.Errorf("The test block file must only have one block in it")
		return nil
	}
	if len(blocks) == 0 {
		t.Errorf("The test block file seems empty")
		return nil
	}
	return blocks[0]
}

func OutputBlock(mb *wire.MsgBlock, name string) er.R {
	buf := bytes.NewBuffer(make([]byte, 0, mb.SerializeSize()))
	err := mb.Serialize(buf)
	b := buf.Bytes()
	if err != nil {
		return err
	}
	fmt.Printf("Creating file %v\n", name)
	fi, errr := os.Create(name)
	if errr != nil {
		return er.E(errr)
	}
	defer fi.Close()
	var header [8]byte
	binary.LittleEndian.PutUint32(header[:4], uint32(protocol.MainNet))
	binary.LittleEndian.PutUint32(header[4:], uint32(len(b)))
	fi.Write(header[:])
	fi.Write(b)
	return nil
}

// LoadAnnouncements reads a file containing a sequence of announcements
// 1024 byte irreducible header followed by content
func LoadAnnouncements(filename string) ([]*wire.PacketCryptAnn, er.R) {
	var fi io.ReadCloser

	fi, errr := os.Open(filename)
	if errr != nil {
		return nil, er.E(errr)
	}
	defer fi.Close()

	anns := make([]*wire.PacketCryptAnn, 0)
	for {
		ann := wire.PacketCryptAnn{}
		if err := ann.BtcDecode(fi, 0, 0); err != nil {
			if er.Wrapped(err) == io.EOF {
				return anns, nil
			}
			return nil, err
		}
		anns = append(anns, &ann)
	}
}

func OutputAnnouncements(anns [][]byte) er.R {
	fi, err := os.Create("./outputanns.dat")
	if err != nil {
		return er.E(err)
	}
	defer fi.Close()
	for _, annb := range anns {
		fi.Write(annb)
	}
	return nil
}
