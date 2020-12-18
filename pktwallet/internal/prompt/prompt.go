// Copyright (c) 2015-2016 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package prompt

import (
	"bufio"
	"bytes"
	"encoding/hex"
	"fmt"
	"os"
	"strings"

	"github.com/pkt-cash/pktd/btcutil/er"

	"github.com/pkt-cash/pktd/btcutil/hdkeychain"
	"github.com/pkt-cash/pktd/pktwallet/internal/legacy/keystore"
	"github.com/pkt-cash/pktd/pktwallet/wallet/seedwords"
	terminal "golang.org/x/term"
)

// ProvideSeed is used to prompt for the wallet seed which maybe required during
// upgrades.
func ProvideSeed() ([]byte, er.R) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter existing wallet seed: ")
		seedStr, err := reader.ReadString('\n')
		if err != nil {
			return nil, er.E(err)
		}
		seedStr = strings.TrimSpace(strings.ToLower(seedStr))

		seed, err := hex.DecodeString(seedStr)
		if err != nil || len(seed) < hdkeychain.MinSeedBytes ||
			len(seed) > hdkeychain.MaxSeedBytes {

			fmt.Printf("Invalid seed specified.  Must be a "+
				"hexadecimal value that is at least %d bits and "+
				"at most %d bits\n", hdkeychain.MinSeedBytes*8,
				hdkeychain.MaxSeedBytes*8)
			continue
		}

		return seed, nil
	}
}

// ProvidePrivPassphrase is used to prompt for the private passphrase which
// maybe required during upgrades.
func ProvidePrivPassphrase() ([]byte, er.R) {
	prompt := "Enter the private passphrase of your wallet: "
	for {
		fmt.Print(prompt)
		pass, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return nil, er.E(err)
		}
		fmt.Print("\n")
		pass = bytes.TrimSpace(pass)
		if len(pass) == 0 {
			continue
		}

		return pass, nil
	}
}

// promptList prompts the user with the given prefix, list of valid responses,
// and default list entry to use.  The function will repeat the prompt to the
// user until they enter a valid response.
func promptList(reader *bufio.Reader, prefix string, validResponses []string, defaultEntry string) (string, er.R) {
	// Setup the prompt according to the parameters.
	validStrings := strings.Join(validResponses, "/")
	var prompt string
	if defaultEntry != "" {
		prompt = fmt.Sprintf("%s (%s) [%s]: ", prefix, validStrings,
			defaultEntry)
	} else {
		prompt = fmt.Sprintf("%s (%s): ", prefix, validStrings)
	}

	// Prompt the user until one of the valid responses is given.
	for {
		fmt.Print(prompt)
		reply, err := reader.ReadString('\n')
		if err != nil {
			return "", er.E(err)
		}
		reply = strings.TrimSpace(strings.ToLower(reply))
		if reply == "" {
			reply = defaultEntry
		}

		for _, validResponse := range validResponses {
			if reply == validResponse {
				return reply, nil
			}
		}
	}
}

// promptListBool prompts the user for a boolean (yes/no) with the given prefix.
// The function will repeat the prompt to the user until they enter a valid
// response.
func promptListBool(reader *bufio.Reader, prefix string, defaultEntry string) (bool, er.R) {
	// Setup the valid responses.
	valid := []string{"n", "no", "y", "yes"}
	response, err := promptList(reader, prefix, valid, defaultEntry)
	if err != nil {
		return false, err
	}
	return response == "yes" || response == "y", nil
}

// promptPass prompts the user for a passphrase with the given prefix.  The
// function will ask the user to confirm the passphrase and will repeat the
// prompts until they enter a matching response.
func promptPass(reader *bufio.Reader, prefix string, confirm bool) ([]byte, er.R) {
	// Prompt the user until they enter a passphrase.
	prompt := fmt.Sprintf("%s: ", prefix)
	for {
		fmt.Print(prompt)
		pass, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return nil, er.E(err)
		}
		fmt.Print("\n")
		pass = bytes.TrimSpace(pass)
		if len(pass) == 0 {
			continue
		}

		if !confirm {
			return pass, nil
		}

		fmt.Print("Confirm passphrase: ")
		confirm, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return nil, er.E(err)
		}
		fmt.Print("\n")
		confirm = bytes.TrimSpace(confirm)
		if !bytes.Equal(pass, confirm) {
			fmt.Println("The entered passphrases do not match")
			continue
		}

		return pass, nil
	}
}

// PrivatePass prompts the user for a private passphrase with varying behavior
// depending on whether the passed legacy keystore exists.  When it does, the
// user is prompted for the existing passphrase which is then used to unlock it.
// On the other hand, when the legacy keystore is nil, the user is prompted for
// a new private passphrase.  All prompts are repeated until the user enters a
// valid response.
func PrivatePass(reader *bufio.Reader, legacyKeyStore *keystore.Store) ([]byte, er.R) {
	// When there is not an existing legacy wallet, simply prompt the user
	// for a new private passphase and return it.
	if legacyKeyStore == nil {
		return promptPass(reader, "Enter the private "+
			"passphrase for your new wallet", true)
	}

	// At this point, there is an existing legacy wallet, so prompt the user
	// for the existing private passphrase and ensure it properly unlocks
	// the legacy wallet so all of the addresses can later be imported.
	fmt.Println("You have an existing legacy wallet.  All addresses from " +
		"your existing legacy wallet will be imported into the new " +
		"wallet format.")
	for {
		privPass, err := promptPass(reader, "Enter the private "+
			"passphrase for your existing wallet", false)
		if err != nil {
			return nil, err
		}

		// Keep prompting the user until the passphrase is correct.
		if err := legacyKeyStore.Unlock([]byte(privPass)); err != nil {
			if keystore.ErrWrongPassphrase.Is(err) {
				fmt.Println(err)
				continue
			}

			return nil, err
		}

		return privPass, nil
	}
}

// PublicPass prompts the user whether they want to add an additional layer of
// encryption to the wallet.  When the user answers yes and there is already a
// public passphrase provided via the passed config, it prompts them whether or
// not to use that configured passphrase.  It will also detect when the same
// passphrase is used for the private and public passphrase and prompt the user
// if they are sure they want to use the same passphrase for both.  Finally, all
// prompts are repeated until the user enters a valid response.
func PublicPass(reader *bufio.Reader, privPass []byte,
	defaultPubPassphrase, configPubPassphrase []byte) ([]byte, er.R) {

	pubPass := defaultPubPassphrase
	usePubPass, err := promptListBool(reader, "Do you want "+
		"to add an additional layer of encryption for public "+
		"data?", "no")
	if err != nil {
		return nil, err
	}

	if !usePubPass {
		return pubPass, nil
	}

	if !bytes.Equal(configPubPassphrase, pubPass) {
		useExisting, err := promptListBool(reader, "Use the "+
			"existing configured public passphrase for encryption "+
			"of public data?", "no")
		if err != nil {
			return nil, err
		}

		if useExisting {
			return configPubPassphrase, nil
		}
	}

	for {
		pubPass, err = promptPass(reader, "Enter the public "+
			"passphrase for your new wallet", true)
		if err != nil {
			return nil, err
		}

		if bytes.Equal(pubPass, privPass) {
			useSamePass, err := promptListBool(reader,
				"Are you sure want to use the same passphrase "+
					"for public and private data?", "no")
			if err != nil {
				return nil, err
			}

			if useSamePass {
				break
			}

			continue
		}

		break
	}

	fmt.Println("NOTE: Use the --walletpass option to configure your " +
		"public passphrase.")
	return pubPass, nil
}

// Seed prompts the user whether they want to use an existing wallet generation
// seed.  When the user answers no, a seed will be generated and displayed to
// the user along with prompting them for confirmation.  When the user answers
// yes, a the user is prompted for it.  All prompts are repeated until the user
// enters a valid response.
func Seed(reader *bufio.Reader, passphrase []byte) ([]byte, *seedwords.Seed, er.R) {
	// Ascertain the wallet generation seed.
	useUserSeed, err := promptListBool(reader, "Do you have an "+
		"existing wallet seed you want to use?", "no")
	if err != nil {
		return nil, nil, err
	}
	if !useUserSeed {
		seed, err := seedwords.RandomSeed()
		if err != nil {
			return nil, nil, err
		}
		fmt.Println("Encrypting your seed...")
		seedEnc := seed.Encrypt(passphrase)
		words, err := seedEnc.Words("english")
		if err != nil {
			return nil, nil, err
		}
		seedEnc.Zero()
		fmt.Println("Your wallet generation seed is:")
		fmt.Printf("\n%s\n\n", words)
		fmt.Println("IMPORTANT: Keep the seed in a safe place.\n" +
			"If your wallet is destroyed, you can recover it as long as\n" +
			"you have this seed and your wallet passphrase.")
		fmt.Println("Please keep in mind that anyone who has access\n" +
			"to the seed only needs to guess your wallet passphrase to\n" +
			"access your funds.")

		for {
			fmt.Print(`Once you have stored the seed in a safe ` +
				`and secure location, type "OK" to continue: `)
			confirmSeed, err := reader.ReadString('\n')
			if err != nil {
				return nil, nil, er.E(err)
			}
			confirmSeed = strings.TrimSpace(confirmSeed)
			confirmSeed = strings.Trim(confirmSeed, `"`)
			if confirmSeed == "OK" {
				break
			}
		}

		return nil, seed, nil
	}

	for {
		fmt.Print("Enter existing wallet seed: ")
		seedStr, err := reader.ReadString('\n')
		if err != nil {
			return nil, nil, er.E(err)
		}
		seedStr = strings.TrimSpace(strings.ToLower(seedStr))

		if seed, err := hex.DecodeString(seedStr); err != nil {
		} else if len(seed) < hdkeychain.MinSeedBytes {
		} else if len(seed) > hdkeychain.MaxSeedBytes {
		} else {
			return []byte(seedStr), nil, nil
		}

		if sw, err := seedwords.SeedFromWords(seedStr); err != nil {
			fmt.Printf("Invalid seed specified [%s]", err.Message())
		} else if sw.NeedsPassphrase() {
			fmt.Println("This seed was taken from a wallet protected by a password.")
			for {
				pass, err := promptPass(reader, "Enter the wallet password now", false)
				if err != nil {
					return nil, nil, err
				}
				fmt.Println("Decrypting your seed...")
				if seed, err := sw.Decrypt(pass, false); err != nil {
					fmt.Println("The seed did not decrypt properly, please try again.")
				} else {
					return nil, seed, nil
				}
			}
		} else {
			if seed, err := sw.Decrypt(nil, false); err != nil {
				return nil, nil, err
			} else {
				return nil, seed, nil
			}
		}
	}
}
