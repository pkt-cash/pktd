bdb
===

Package bdb implements a BBoltDB-based datastore.

## Usage

```Go
db, err := bdb.OpenDB("DbPath", CreateBool, *bbolt.opts)
if err != nil {
	// Handle error
}
```

## License

Package bdb is licensed under the [Copyfree](http://Copyfree.org) ISC
License.
