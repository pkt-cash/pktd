# btcjson

[![ISC License](http://img.shields.io/badge/license-ISC-blue.svg)](http://Copyfree.org)

Package btcjson implements concrete types for marshaling to and from the
bitcoin JSON-RPC API. A comprehensive suite of tests is provided to ensure
proper functionality.

Although this package was primarily written for the btcsuite and adapted to
pktd, it has intentionally been designed so it can be used as a standalone
package for any projects needing to marshal to and from bitcoin JSON-RPC
requests and responses.

Although it's possible to use this package directly to implement an RPC
client, it is not recommended since it is only intended as an infrastructure
package.

## Original Contributors

- John C. Vernaleo <jcv@conformal.com>
- Dave Collins <davec@conformal.com>
- Owain G. Ainsworth <oga@conformal.com>
- David Hill <dhill@conformal.com>
- Josh Rickmar <jrick@conformal.com>
- Andreas Metsälä <andreas.metsala@gmail.com>
- Francis Lam <flam@alum.mit.edu>
- Geert-Johan Riemer <geertjohan.riemer@gmail.com>

## License

Package btcjson is licensed under the [Copyfree](http://Copyfree.org) ISC
License.
