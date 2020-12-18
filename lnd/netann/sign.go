package netann

import (
	"github.com/pkt-cash/pktd/btcec"
	"github.com/pkt-cash/pktd/btcutil/er"
	"github.com/pkt-cash/pktd/lnd/input"
	"github.com/pkt-cash/pktd/lnd/lnwallet"
	"github.com/pkt-cash/pktd/lnd/lnwire"
)

// SignAnnouncement signs any type of gossip message that is announced on the
// network.
func SignAnnouncement(signer lnwallet.MessageSigner, pubKey *btcec.PublicKey,
	msg lnwire.Message) (input.Signature, er.R) {
	var (
		data []byte
		err  er.R
	)

	switch m := msg.(type) {
	case *lnwire.ChannelAnnouncement:
		data, err = m.DataToSign()
	case *lnwire.ChannelUpdate:
		data, err = m.DataToSign()
	case *lnwire.NodeAnnouncement:
		data, err = m.DataToSign()
	default:
		return nil, er.Errorf("can't sign %T message", m)
	}
	if err != nil {
		return nil, er.Errorf("unable to get data to sign: %v", err)
	}

	return signer.SignMessage(pubKey, data)
}
