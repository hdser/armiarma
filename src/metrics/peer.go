package metrics

import (
	"fmt"
	"github.com/pkg/errors"
	"github.com/protolambda/rumor/metrics/utils"
	pgossip "github.com/protolambda/rumor/p2p/gossip"
	log "github.com/sirupsen/logrus"
	"strconv"
	"time"
)

// Base Struct for the topic name and the received messages on the different topics
// TODO: In the future we might reuse the Rumor struct and add the missing fields
type Peer struct {
	PeerId        string
	NodeId        string
	UserAgent     string
	ClientName    string
	ClientOS      string //TODO:
	ClientVersion string
	Pubkey        string
	Addrs         string
	Ip            string
	Country       string
	City          string
	Latency       float64
	// TODO: Store Enr

	ConnectedDirection string
	IsConnected        bool
	Attempted          bool   // If the peer has been attempted to stablish a connection
	Succeed            bool   // If the connection attempt has been successful
	Attempts           uint64 // Number of attempts done
	Error              string // Type of error that we detected. TODO: We are just storing the last one
	ConnectionTimes    []time.Time
	DisconnectionTimes []time.Time

	MetadataRequest bool  // If the peer has been attempted to request its metadata
	MetadataSucceed bool  // If the peer has been successfully requested its metadata
	LastExport      int64 //(timestamp in seconds of the last exported time (backup for when we are loading the Peer)

	// Counters for the different topics
	BeaconBlock          MessageMetrics
	BeaconAggregateProof MessageMetrics
	VoluntaryExit        MessageMetrics
	ProposerSlashing     MessageMetrics
	AttesterSlashing     MessageMetrics
	// Variables related to the SubNets (only needed for when Shards will be implemented)
}

func NewPeer(peerId string) Peer {
	pm := Peer{
		// TODO Check. What is the difference between Unknown and "" empty.
		PeerId:    peerId,
		NodeId:    "",
		UserAgent: "",
		Pubkey:    "",
		Addrs:     "",
		Ip:        "",
		Country:   "",
		City:      "",
		Latency:   0,

		Attempted: false,
		Succeed:   false,
		Attempts:  0,
		Error:     "None",

		MetadataRequest:    false,
		MetadataSucceed:    false,
		IsConnected:        false,
		ConnectedDirection: "",
		LastExport:         0,

		ConnectionTimes:    make([]time.Time, 0),
		DisconnectionTimes: make([]time.Time, 0),

		// Counters for the different topics
		BeaconBlock:          NewMessageMetrics(),
		BeaconAggregateProof: NewMessageMetrics(),
		VoluntaryExit:        NewMessageMetrics(),
		ProposerSlashing:     NewMessageMetrics(),
		AttesterSlashing:     NewMessageMetrics(),
	}
	return pm
}

func (pm *Peer) ResetDynamicMetrics() {
	pm.Attempts = 0
	pm.BeaconBlock = NewMessageMetrics()
	pm.BeaconAggregateProof = NewMessageMetrics()
	pm.VoluntaryExit = NewMessageMetrics()
	pm.ProposerSlashing = NewMessageMetrics()
	pm.AttesterSlashing = NewMessageMetrics()
}

func (pm *Peer) GetAllMessagesCount() uint64 {
	return (pm.BeaconBlock.Count +
		pm.BeaconAggregateProof.Count +
		pm.VoluntaryExit.Count +
		pm.AttesterSlashing.Count +
		pm.ProposerSlashing.Count)
}

// Register when a new connection was detected
func (pm *Peer) AddConnectionEvent(direction string, time time.Time) {
	pm.ConnectionTimes = append(pm.ConnectionTimes, time)
	pm.IsConnected = true
	pm.ConnectedDirection = direction
}

// Register when a disconnection was detected
func (pm *Peer) AddDisconnectionEvent(time time.Time) {
	pm.DisconnectionTimes = append(pm.DisconnectionTimes, time)
	pm.IsConnected = false
	pm.ConnectedDirection = ""
}

// Register when a connection attempt was made. Note that there is some
// overlap with AddConnectionEvent
func (pm *Peer) AddNewConnectionAttempt(succeed bool, err string) {
	pm.Attempts += 1
	if !pm.Attempted {
		pm.Attempted = true
	}
	if succeed {
		pm.Succeed = succeed
		pm.Error = "None"
	} else {
		pm.Error = utils.FilterError(err)
	}
}

// Calculate the total connected time based on con/disc timestamps
func (pm *Peer) GetConnectedTime() float64 {
	var totalConnectedTime int64
	for _, conTime := range pm.ConnectionTimes {
		for _, discTime := range pm.DisconnectionTimes {
			singleConnectionTime := discTime.Sub(conTime).Milliseconds()
			if singleConnectionTime >= 0 {
				totalConnectedTime += singleConnectionTime
				break
			}
		}
	}
	return float64(totalConnectedTime) / 60000
}

func (pm *Peer) GetMessageMetrics(topicName string) (*MessageMetrics, error) {
	// All this could be inside a different function
	switch topicName {
	case pgossip.BeaconBlock:
		return &pm.BeaconBlock, nil
	case pgossip.BeaconAggregateProof:
		return &pm.BeaconAggregateProof, nil
	case pgossip.VoluntaryExit:
		return &pm.VoluntaryExit, nil
	case pgossip.ProposerSlashing:
		return &pm.ProposerSlashing, nil
	case pgossip.AttesterSlashing:
		return &pm.AttesterSlashing, nil
	default:
		return nil, errors.New("unknown topic name: " + topicName)
	}
}

func (pm *Peer) ToCsvLine() string {
	csvRow := pm.PeerId + "," +
		pm.NodeId + "," +
		pm.UserAgent + "," +
		pm.ClientName + "," +
		pm.ClientVersion + "," +
		pm.Pubkey + "," +
		pm.Addrs + "," +
		pm.Ip + "," +
		pm.Country + "," +
		pm.City + "," +
		strconv.FormatBool(pm.MetadataRequest) + "," +
		strconv.FormatBool(pm.MetadataSucceed) + "," +
		strconv.FormatBool(pm.Attempted) + "," +
		strconv.FormatBool(pm.Succeed) + "," +
		strconv.FormatBool(pm.IsConnected) + "," +
		strconv.FormatUint(pm.Attempts, 10) + "," +
		pm.Error + "," +
		fmt.Sprint(pm.Latency) + "," +
		fmt.Sprintf("%d", len(pm.ConnectionTimes)) + "," +
		fmt.Sprintf("%d", len(pm.DisconnectionTimes)) + "," +
		fmt.Sprintf("%.3f", pm.GetConnectedTime()) + "," +
		strconv.FormatUint(pm.BeaconBlock.Count, 10) + "," +
		strconv.FormatUint(pm.BeaconAggregateProof.Count, 10) + "," +
		strconv.FormatUint(pm.VoluntaryExit.Count, 10) + "," +
		strconv.FormatUint(pm.ProposerSlashing.Count, 10) + "," +
		strconv.FormatUint(pm.AttesterSlashing.Count, 10) + "," +
		strconv.FormatUint(pm.GetAllMessagesCount(), 10) + "\n"

	return csvRow
}

func (pm *Peer) LogPeer() {
	log.WithFields(log.Fields{
		"PeerId":        pm.PeerId,
		"NodeId":        pm.NodeId,
		"UserAgent":     pm.UserAgent,
		"ClientName":    pm.ClientName,
		"ClientOS":      pm.ClientOS,
		"ClientVersion": pm.ClientVersion,
		"Pubkey":        pm.Pubkey,
		"Addrs":         pm.Addrs,
		"Ip":            pm.Ip,
		"Country":       pm.Country,
		"City":          pm.City,
		"Latency":       pm.Latency,
	}).Info("Peer Info")
}
