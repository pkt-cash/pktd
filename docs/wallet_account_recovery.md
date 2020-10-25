# Wallet Account Recovery

The use of accounts with the wallet is *unsupported*, in case you have accidentally created a wallet with accounts and are now unable to access funds in an account, this procedure will recover the funds.

First, you must determine what account address(es) have received funds which must be recovered. Get the list of addresses which need to be recovered.

Secondly, you need to use the dumpprivkey command to extract the key for the account, for example:

```
btcctl -u x -P x --wallet dumpprivkey myaccount
```

You may need to use walletpassphrase to unlock your wallet first.

Now that you have the private key(s), you will need to stop the pktwallet daemon and create a new wallet. Once pktwallet daemon is stopped, you should *move* your wallet to safety, for example:

```
# Linux
mv ~/.pktwallet/pkt/wallet.db ~/.pktwallet/pkt/wallet_personal.db

# Apple
mv ~/Library/Application\ Support/Pktwallet/pkt/wallet.db ~/Library/Application\ Support/Pktwallet/pkt/wallet_personal.db
```

Now that these are out of the way, you can create a new wallet:

```
pktwallet --create
```

After you have followed all of the steps, you then launch the pktwallet daemon:

```
pktwallet -u x -P x
```

Now that the daemon is launched, you need to *import* the private keys which you exported earlier:

```
# Repeat this for each of your private keys
btcctl -u x -P x --wallet importprivkey <secret private key>
```

Now that you have imported all of your private keys, you must use the resync command to cause pktwallet to search the blockchain for any funds associated with those keys.

```
btcctl -u x -P x --wallet resync
```

After that is complete, use getbalance to check that the coins are present.

```
btcctl -u x -P x --wallet getbalance
```

Now you will be able to spend your coins using this wallet, however because the address in this wallet is *imported*, it is not derived from the seed and is therefore at risk in case of a loss of the wallet, so it is recommended that you migrate the funds to a more reliable wallet as soon as possible.