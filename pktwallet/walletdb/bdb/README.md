bdb
===

Package bdb implements an driver for walletdb that uses bboltdb for the
backing datastore.  Package bdb is licensed under the Copyfree ISC license.

## Usage

This package is only a driver to the walletdb package and provides the database
type of "bdb".  The only parameter the Open and Create functions take is the
database path as a string:

```Go
db, err := walletdb.Open("bdb", "path/to/database.db")
if err != nil {
	// Handle error
}
```

```Go
db, err := walletdb.Create("bdb", "path/to/database.db", *bbolt.opts)
if err != nil {
	// Handle error
}
```

## License

Package bdb is licensed under the [Copyfree](http://Copyfree.org) ISC
License.
