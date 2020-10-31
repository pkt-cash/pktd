package pktconfig

const PktwalletSampleConfig = `
[Application Options]

; ------------------------------------------------------------------------------
; PKT wallet settings
; ------------------------------------------------------------------------------

; The directory to open and save wallet, transaction, and unspent transaction
; output files.
; appdata=~/.pktwallet


; ------------------------------------------------------------------------------
; RPC client settings
; ------------------------------------------------------------------------------

; The server and port used for pktd websocket connections.
; rpcconnect=localhost:18334

; File containing root certificates to authenticate a TLS connections with pktd
; cafile=~/.pktwallet/pktd.cert



; ------------------------------------------------------------------------------
; RPC server settings
; ------------------------------------------------------------------------------

; TLS certificate and key file locations
; rpccert=~/.pktwallet/rpc.cert
; rpckey=~/.pktwallet/rpc.key

; Enable one time TLS keys.  This option results in the process generating
; a new certificate pair each startup, writing only the certificate file
; to disk.  This is a more secure option for clients that only interact with
; a local wallet process where persistent certs are not needed.
;
; This option will error at startup if the key specified by the rpckey option
; already exists.
; onetimetlskey=0

; Specify the interfaces for the RPC server listen on.  One rpclisten address
; per line.  Multiple rpclisten options may be set in the same configuration,
; and each will be used to listen for connections.  NOTE: The default port is
; modified by some options such as 'testnet', so it is recommended to not
; specify a port and allow a proper default to be chosen unless you have a
; specific reason to do otherwise.
; rpclisten=                ; all interfaces on default port
; rpclisten=0.0.0.0         ; all ipv4 interfaces on default port
; rpclisten=::              ; all ipv6 interfaces on default port
; rpclisten=:8332           ; all interfaces on port 8332
; rpclisten=0.0.0.0:8332    ; all ipv4 interfaces on port 8332
; rpclisten=[::]:8332       ; all ipv6 interfaces on port 8332
; rpclisten=127.0.0.1:8332  ; only ipv4 localhost on port 8332 (this is a default)
; rpclisten=[::1]:8332      ; only ipv6 localhost on port 8332 (this is a default)
; rpclisten=127.0.0.1:8337  ; only ipv4 localhost on non-standard port 8337
; rpclisten=:8337           ; all interfaces on non-standard port 8337
; rpclisten=0.0.0.0:8337    ; all ipv4 interfaces on non-standard port 8337
; rpclisten=[::]:8337       ; all ipv6 interfaces on non-standard port 8337


; ------------------------------------------------------------------------------
; RPC settings (both client and server)
; ------------------------------------------------------------------------------

; Username and password to authenticate to pktd a RPC server and authenticate
; new client connections
; rpcuser=
; rpcpass=

; Alternative username and password for pktd.  If set, these will be used
; instead of the username and password set above for authentication to a
; pktd RPC server.
; pktdusername=
; pktdpassword=


; ------------------------------------------------------------------------------
; Debug
; ------------------------------------------------------------------------------

; Debug logging level.
; Valid options are {trace, debug, info, warn, error, critical}
; debuglevel=info

; The port used to listen for HTTP profile requests.  The profile server will
; be disabled if this option is not specified.  The profile information can be
; accessed at http://localhost:<profileport>/debug/pprof once running.
; profile=6062
`
