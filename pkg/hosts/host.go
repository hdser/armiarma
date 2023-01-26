package hosts

import (
	"context"
	"fmt"

	"github.com/migalabs/armiarma/pkg/db/models"
	psql "github.com/migalabs/armiarma/pkg/db/postgresql"
	"github.com/migalabs/armiarma/pkg/utils"
	"github.com/migalabs/armiarma/pkg/utils/apis"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	noise "github.com/libp2p/go-libp2p-noise"
	"github.com/libp2p/go-libp2p/p2p/protocol/identify"
	tcp_transport "github.com/libp2p/go-tcp-transport"

	log "github.com/sirupsen/logrus"

	ma "github.com/multiformats/go-multiaddr"

	"github.com/pkg/errors"
)

var (
	ModuleName       = "LIBP2P-HOST"
	ConnNotChannSize = 200
)

// Struct that defines the Basic Struct asociated to the Libtp2p host
type BasicLibp2pHost struct {
	ctx     context.Context
	Network utils.NetworkType
	// Basic sevices related with the libp2p host
	host      host.Host
	identify  identify.IDService
	DBClient  *psql.DBClient
	IpLocator *apis.IpLocator

	// Basic Host Metadata
	multiAddr     ma.Multiaddr
	fullMultiAddr ma.Multiaddr

	connEventNotChannel chan *models.EventTrace
	identNotChannel     chan IdentificationEvent
	peerID              peer.ID
}

// NewBasicLibp2pEth2Host generate a new Libp2p host from the given context and Options, for Eth2 network (or similar).
func NewBasicLibp2pEth2Host(
	ctx context.Context,
	ip string,
	port int,
	privKey *crypto.Secp256k1PrivateKey,
	userAgent string,
	network utils.NetworkType,
	ipLocator *apis.IpLocator,
	ps *psql.DBClient) (*BasicLibp2pHost, error) {

	// generate de multiaddress
	multiaddr := fmt.Sprintf("/ip4/%s/tcp/%d", ip, port)
	muladdr, err := ma.NewMultiaddr(multiaddr)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("couldn't generate multiaddress from ip %s and tcp %s", ip, port))
	}
	log.Debugf("setting multiaddress to %s", muladdr)

	// Generate the main Libp2p host that will be exposed to the network
	host, err := libp2p.New(
		libp2p.ListenAddrs(muladdr),
		libp2p.Identity(privKey),
		libp2p.UserAgent(userAgent),
		libp2p.Transport(tcp_transport.NewTCPTransport),
		libp2p.Security(noise.ID, noise.New),
		libp2p.NATPortMap(),
	)
	if err != nil {
		return nil, err
	}
	peerId := host.ID().String()
	fmaddr := host.Addrs()[0].String() + "/p2p/" + host.ID().String()
	localMultiaddr, _ := ma.NewMultiaddr(fmaddr)
	log.Debugf("full multiaddress %s", localMultiaddr)
	// generate the identify service
	ids, err := identify.NewIDService(
		host,
		identify.UserAgent(userAgent),
		identify.DisableSignedPeerRecord(),
	)
	if err != nil {
		return nil, err
	}

	// Gererate the struct that contains all the configuration and structs surrounding the Libp2p Host
	basicHost := &BasicLibp2pHost{
		ctx:                 ctx,
		Network:             network,
		host:                host,
		identify:            ids,
		DBClient:            ps,
		IpLocator:           ipLocator,
		multiAddr:           muladdr,
		fullMultiAddr:       localMultiaddr,
		peerID:              peer.ID(peerId),
		connEventNotChannel: make(chan *models.EventTrace, ConnNotChannSize),
		identNotChannel:     make(chan IdentificationEvent, ConnNotChannSize),
	}
	log.Debug("setting custom notification functions")
	basicHost.SetCustomNotifications()

	return basicHost, nil
}

// This should be move to to a range of options inside the NewBasicHost

// NewBasicLibp2pFilecoinHost:
// Generate a new Libp2p host from the given context and Options, for Filecoin network.
// func NewBasicLibp2pIpfsHost(ctx context.Context, infObj info.IpfsInfoData, ipLocator *apis., ps *db.PeerStore) (*BasicLibp2pHost, error) {
// 	ip := infObj.IP.String()
// 	tcp := fmt.Sprintf("%d", infObj.TcpPort)
// 	privkey := infObj.PrivateKey
// 	userAgent := infObj.UserAgent

// 	// generate de multiaddress
// 	multiaddr := fmt.Sprintf("/ip4/%s/tcp/%s", ip, tcp)
// 	muladdr, err := ma.NewMultiaddr(multiaddr)
// 	if err != nil {
// 		log.Debugf("couldn't generate multiaddress from ip %s and tcp %s", ip, tcp)
// 		multiaddr = fmt.Sprintf("/ip4/%s/tcp/%s", DefaultIP, DefaultTCP)
// 		muladdr, _ = ma.NewMultiaddr(multiaddr)
// 	}
// 	log.Debugf("setting multiaddress to %s", muladdr)

// 	// Generate the main Libp2p host that will be exposed to the network
// 	host, err := libp2p.New(
// 		libp2p.Identity(privkey),
// 		libp2p.ListenAddrs(muladdr),
// 		libp2p.Ping(true),
// 		libp2p.UserAgent(userAgent),
// 		libp2p.NATPortMap(),
// 	)
// 	if err != nil {
// 		return nil, err
// 	}
// 	peerId := host.ID().String()
// 	fmaddr := host.Addrs()[0].String() + "/p2p/" + host.ID().String()
// 	localMultiaddr, _ := ma.NewMultiaddr(fmaddr)
// 	log.Debugf("full multiaddress %s", localMultiaddr)
// 	// generate the identify service
// 	ids, err := identify.NewIDService(host, identify.UserAgent(userAgent), identify.DisableSignedPeerRecord())
// 	if err != nil {
// 		return nil, err
// 	}
// 	// Gererate the struct that contains all the configuration and structs surrounding the Libp2p Host
// 	basicHost := &BasicLibp2pHost{
// 		ctx:                 ctx,
// 		Network:             "ipfs",
// 		host:                host,
// 		identify:            ids,
// 		PeerStore:           ps,
// 		IpLocator:           ipLocator,
// 		multiAddr:           muladdr,
// 		fullMultiAddr:       localMultiaddr,
// 		peerID:              peer.ID(peerId),
// 		connEventNotChannel: make(chan ConnectionEvent, ConnNotChannSize),
// 		identNotChannel:     make(chan IdentificationEvent, ConnNotChannSize),
// 	}
// 	log.Debug("setting custom notification functions")
// 	basicHost.SetCustomNotifications()

// 	return basicHost, nil
// }

func (b *BasicLibp2pHost) Host() host.Host {
	return b.host
}

// Start:
// Start the libp2pHost module (perhaps NOT NECESSARY).
// So far, start listening on the fullMultiAddrs.
func (b *BasicLibp2pHost) Start() {
	err := b.host.Network().Listen()
	if err != nil {
		log.Errorf("error starting. %s", err)
	} else {
		log.Infof("libp2p host module started")
	}
}

func (b *BasicLibp2pHost) Ctx() context.Context {
	return b.ctx
}

// RecConnEvent
// Record Connection Event
// @param connEvent: the event to insert in the notification channel
func (b *BasicLibp2pHost) RecConnEvent(eventTrace *models.EventTrace) {
	b.connEventNotChannel <- eventTrace
}

func (b *BasicLibp2pHost) ConnEventNotChannel() chan *models.EventTrace {
	return b.connEventNotChannel
}

// RecIdentEvent
// Record Identification Event
// @param identEvent: the event to insert in the notification channel
func (b *BasicLibp2pHost) RecIdentEvent(identEvent IdentificationEvent) {
	b.identNotChannel <- identEvent
}

func (b *BasicLibp2pHost) IdentEventNotChannel() chan IdentificationEvent {
	return b.identNotChannel
}
