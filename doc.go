// Copyright (c) 2013-2017 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

/*
pktd is the reference full-node PKT Cash implementation, written in Go.

pktd will work 'out of the box' for most users. However, there are a wide
variety of flags that can be used to control it.

The following section provides a usage overview which enumerates the flags. An
interesting point to note is that the long form of all of these options
(except -C) can be specified in a configuration file that is automatically
parsed when pktd starts up.  By default, the configuration file is located at
~/.pktd/pktd.conf on POSIX operating systems and %LOCALAPPDATA%\pktd\pktd.conf
on Windows. The -C (--configfile) flag, as shown below, is used to override
this location.

Usage:
  pktd [OPTIONS]

Application Options:
  -V, --version               Display version information and exit
  -C, --configfile=           Path to configuration file (default: /home/jhj/.pktd/pktd.conf)
  -b, --datadir=              Directory to store data (default: /home/jhj/.pktd/data)
      --logdir=               Directory to log output. (default: /home/jhj/.pktd/logs)
  -a, --addpeer=              Add a peer to connect with at startup
      --connect=              Connect only to the specified peers at startup
      --nolisten              Disable listening for incoming connections -- NOTE: Listening is automatically disabled if the --connect option is used without also specifying listening interfaces via --listen
      --listen=               Add an interface/port to listen for connections (default all interfaces port: 8333, testnet: 18333)
      --maxpeers=             Max number of inbound and outbound peers (default: 125)
      --nobanning             Disable banning of misbehaving peers
      --banduration=          How long to ban misbehaving peers.  Valid time units are {s, m, h}.  Minimum 1 second (default: 24h0m0s)
      --banthreshold=         Maximum allowed ban score before disconnecting and banning misbehaving peers. (default: 120)
      --whitelist=            Add an IP network or IP that will not be banned. (eg. 192.168.1.0/24 or ::1)
      --agentblacklist=       A comma separated list of user-agent substrings which will cause pktd to reject any peers whose user-agent contains any of the blacklisted substrings.
      --agentwhitelist=       A comma separated list of user-agent substrings which will cause pktd to require all peers' user-agents to contain one of the whitelisted substrings. The blacklist is applied before the blacklist, and an empty
                              whitelist will allow all agents that do not fail the blacklist.
      --homedir=              Creates this directory at startup (default: /home/jhj/.pktd)
  -u, --rpcuser=              Username for RPC connections
  -P, --rpcpass=              Password for RPC connections
      --rpclimituser=         Username for limited RPC connections
      --rpclimitpass=         Password for limited RPC connections
      --rpclisten=            Add an interface/port to listen for RPC connections (default port: 8334, testnet: 18334)
      --rpccert=              File containing the certificate file (default: /home/jhj/.pktd/rpc.cert)
      --rpckey=               File containing the certificate key (default: /home/jhj/.pktd/rpc.key)
      --rpcmaxclients=        Max number of RPC clients for standard connections (default: 10)
      --rpcmaxwebsockets=     Max number of RPC websocket connections (default: 25)
      --rpcmaxconcurrentreqs= Max number of concurrent RPC requests that may be processed concurrently (default: 20)
      --rpcquirks             Mirror some JSON-RPC quirks of Bitcoin Core -- NOTE: Discouraged unless interoperability issues need to be worked around
      --norpc                 Disable built-in RPC server -- NOTE: The RPC server is disabled by default if no rpcuser/rpcpass or rpclimituser/rpclimitpass is specified
      --tls                   Enable TLS for the RPC server -- default is disabled unless bound to non-localhost
      --nodnsseed             Disable DNS seeding for peers
      --externalip=           Add an ip to the list of local addresses we claim to listen on to peers
      --testnet               Use the test network
      --pkttest               Use the pkt.cash test network
      --btc                   Use the bitcoin main network
      --pkt                   Use the pkt.cash main network
      --regtest               Use the regression test network
      --simnet                Use the simulation test network
      --addcheckpoint=        Add a custom checkpoint.  Format: '<height>:<hash>'
      --nocheckpoints         Disable built-in checkpoints.  Don't do this unless you know what you're doing.
      --statsviz=             Enable StatsViz runtime visualization on given port -- NOTE port must be between 1024 and 65535
      --profile=              Enable HTTP profiling on given port -- NOTE port must be between 1024 and 65535
      --cpuprofile=           Write CPU profile to the specified file
  -d, --debuglevel=           Logging level for all subsystems {trace, debug, info, warn, error, critical} -- You may also specify <subsystem>=<level>,<subsystem2>=<level>,... to set the log level for individual subsystems -- Use show to list
                              available subsystems (default: info)
      --upnp                  Use UPnP to map our listening port outside of NAT
      --minrelaytxfee=        The minimum transaction fee in BTC/kB to be considered a non-zero fee. (default: -1)
      --limitfreerelay=       Limit relay of transactions with no transaction fee to the given amount in thousands of bytes per minute (default: 15)
      --norelaypriority       Do not require free or low-fee transactions to have high priority for relaying
      --trickleinterval=      Minimum time between attempts to send new inventory to a connected peer (default: 2s)
      --maxorphantx=          Max number of orphan transactions to keep in memory (default: 100)
      --generate              Generate (mine) bitcoins using the CPU
      --miningaddr=           Add the specified payment address to the list of addresses to use for generated blocks -- At least one address is required if the generate option is set
      --blockminsize=         Mininum block size in bytes to be used when creating a block
      --blockmaxsize=         Maximum block size in bytes to be used when creating a block (default: 750000)
      --blockminweight=       Mininum block weight to be used when creating a block
      --blockmaxweight=       Maximum block weight to be used when creating a block (default: 3000000)
      --blockprioritysize=    Size in bytes for high-priority/low-fee transactions when creating a block (default: 50000)
      --uacomment=            Comment to add to the user agent -- See BIP 14 for more information.
      --nopeerbloomfilters    Disable bloom filtering support
      --nocfilters            Disable committed filtering (CF) support
      --dropcfindex           Deletes the index used for committed filtering (CF) support from the database on start up and then exits.
      --sigcachemaxsize=      The maximum number of entries in the signature verification cache (default: 100000)
      --blocksonly            Do not accept transactions from remote peers.
      --txindex               Maintain a full hash-based transaction index which makes all transactions available via the getrawtransaction RPC
      --droptxindex           Deletes the hash-based transaction index from the database on start up and then exits.
      --addrindex             Maintain a full address-based transaction index which makes the searchrawtransactions RPC available
      --dropaddrindex         Deletes the address-based transaction index from the database on start up and then exits.
      --relaynonstd           Relay non-standard transactions regardless of the default settings for the active network.
      --rejectnonstd          Reject non-standard transactions regardless of the default settings for the active network.
      --rejectreplacement     Reject transactions that attempt to replace existing transactions within the mempool through the Replace-By-Fee (RBF) signaling policy.
      --miningskipchecks=     Either 'txns', 'template' or 'both', skips certain time-consuming checks during mining process, be careful as you might create invalid block templates!

Help Options:
  -h, --help                  Show this help message

*/
package main
