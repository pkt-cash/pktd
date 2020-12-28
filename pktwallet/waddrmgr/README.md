# waddrmgr

Package waddrmgr provides a secure hierarchical deterministic wallet address
manager.

## Feature Overview

- BIP0032 hierarchical deterministic keys
- BIP0043/BIP0044 multi-account hierarchy
- Strong focus on security:
  - Fully encrypted database including public information such as addresses as
    well as private information such as private keys and scripts needed to
    redeem pay-to-script-hash transactions
  - Hardened against memory scraping through the use of actively clearing
    private material from memory when locked
  - Different crypto keys used for public, private, and script data
  - Ability for different passphrases for public and private data
  - Scrypt-based key derivation
  - NaCl-based secretbox cryptography (XSalsa20 and Poly1305)
- Scalable design:
  - Multi-tier key design to allow instant password changes regardless of the
    number of addresses stored
  - Import WIF keys
  - Import pay-to-script-hash scripts for things such as multi-signature
    transactions
  - Ability to export a watching-only version which does not contain any private
    key material
  - Programmatically detectable errors, including encapsulation of errors from
    packages it relies on
  - Address synchronization capabilities
- Comprehensive test coverage

## License

Package waddrmgr is licensed under the [Copyfree](http://Copyfree.org) ISC
License.
