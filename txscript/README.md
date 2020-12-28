# txscript

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://Copyfree.org)

Package txscript implements the bitcoin transaction script language. There is
a comprehensive test suite.

This package has intentionally been designed so it can be used as a standalone
package for any projects needing to use or validate bitcoin transaction scripts.

## Bitcoin Scripts

Bitcoin provides a stack-based, FORTH-like language for the scripts in
the bitcoin transactions. This language is not turing complete
although it is still fairly powerful. A description of the language
can be found at https://en.bitcoin.it/wiki/Script

## License

Package txscript is licensed under the [Copyfree](http://Copyfree.org) ISC
License.
